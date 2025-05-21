package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	"github.com/lengrongfu/snapshots-quota/pkg/constant"
	"github.com/lengrongfu/snapshots-quota/pkg/quota"
	"github.com/lengrongfu/snapshots-quota/pkg/utils"
)

var (
	// pluginName is the name of the plugin
	pluginName string
	// pluginIdx is the index of the plugin
	pluginIdx string
	// quotaSize is the default quota size for the container
	quotaSize uint64
	// containerdStateDir is the directory where containerd stores its state
	containerdStateDir string
	// containerdRootDir is the root directory for containerd
	containerdRootDir string
	// containerdBasePath is the base path for containerd, e.g "/run/containerd" and "/var/lib/containerd" base path is "/"
	containerdBasePath string
	// containerdSocket is the socket for containerd
	containerdSocket string
	// containerdNamespace is the namespace for containerd
	containerdNamespace string
	// useEphemeralStorage to overlay quotaSize
	useEphemeralStorage bool
	// enableLabelSelect is the flag to enable label select
	enableLabelSelect bool
	// filterLabelSelect is the label select map
	filterLabelSelect = make(utils.FlagMap)
)

// our injector plugin
type plugin struct {
	stub                    stub.Stub
	quotaCtl                *quota.Control
	client                  *containerd.Client
	containerProjectMapSync sync.RWMutex
	containerProjectMap     map[string]uint32
}

func (p *plugin) PostCreateContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	klog.Infof("PostCreateContainer pod id: %s, container name: %s", pod.Id, ctr.Name)
	if enableLabelSelect {
		if filterLabelSelect == nil {
			klog.Warningf("label select map is nil")
		}
		if filterLabelSelect != nil && !utils.FilterPodByLabelSelect(pod, filterLabelSelect) {
			klog.InfoS("pod %s/%s not match label select map %v", pod.Namespace, pod.Name, filterLabelSelect)
			return nil
		}
	}
	ctx = namespaces.WithNamespace(ctx, containerdNamespace)
	c, err := p.client.ContainerService().Get(ctx, ctr.Id)
	if err != nil {
		klog.Errorf("from containerID: %s get container info error : %s", ctr.Id, err)
		return err
	}
	if c.Snapshotter != containerd.DefaultSnapshotter {
		klog.Warning("container is not use overlayfs snapshotter")
		return nil
	}
	mounts, err := p.client.SnapshotService(c.Snapshotter).Mounts(ctx, c.SnapshotKey)
	if err != nil {
		klog.Errorf("Snapshotter get mounts error: %s", err)
		return err
	}
	upperDir := parseUpperDir(mounts)
	if upperDir == "" {
		klog.Warningf("upperdir is empty string, mount info is %+v", mounts)
		return nil
	}

	upperdirProject, err := p.quotaCtl.SetProject(filepath.Join(upperDir, "fs"))
	if err != nil {
		klog.Errorf("set project by target %s error %s", upperDir, err)
		return err
	}
	p.containerProjectMapSync.Lock()
	p.containerProjectMap[ctr.Id] = upperdirProject
	p.containerProjectMapSync.Unlock()

	return p.quotaCtl.SetProjectByProject(filepath.Join(upperDir, "work"), upperdirProject)
}

func (p *plugin) PostStartContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	klog.Infof("PostStartContainer pod id: %s, container name: %s", pod.Id, ctr.Name)
	if enableLabelSelect {
		if filterLabelSelect == nil {
			klog.Warningf("label select map is nil")
		}
		if filterLabelSelect != nil && !utils.FilterPodByLabelSelect(pod, filterLabelSelect) {
			klog.InfoS("pod %s/%s not match label select map %v", pod.Namespace, pod.Name, filterLabelSelect)
			return nil
		}
	}
	rootfs := filepath.Join(containerdStateDir, "io.containerd.runtime.v2.task", containerdNamespace, ctr.Id, "rootfs")
	var size = quotaSize
	if useEphemeralStorage {
		ephemeralStorage, err := utils.GetEphemeralStorage(ctx, pod, ctr)
		if err != nil {
			klog.Errorf("get ephemeral-storage error, fallback to use global quota size: %d, %s", quotaSize, err)
		} else {
			size = ephemeralStorage
		}
	}
	q := quota.Quota{
		Size: size,
	}
	p.containerProjectMapSync.RLock()
	projectId, ok := p.containerProjectMap[ctr.Id]
	if !ok {
		klog.Errorf("container project not save")
		return nil
	}
	p.containerProjectMapSync.RUnlock()
	err := p.quotaCtl.SetProjectByProject(rootfs, projectId)
	if err != nil {
		klog.Errorf("set overlayfs quota project error: %s", err)
		return err
	}
	if err = p.quotaCtl.SetProjectQuota(q, projectId); err != nil {
		klog.Errorf("set project %d quota error %s", projectId, err)
		return err
	}
	return nil
}

func (p *plugin) RemoveContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	klog.Infof("RemoveContainer pod id: %s, container name: %s", pod.Id, ctr.Name)
	if enableLabelSelect {
		if filterLabelSelect == nil {
			klog.Warningf("label select map is nil")
		}
		if filterLabelSelect != nil && !utils.FilterPodByLabelSelect(pod, filterLabelSelect) {
			klog.InfoS("pod %s/%s not match label select map %v", pod.Namespace, pod.Name, filterLabelSelect)
			return nil
		}
	}
	p.containerProjectMapSync.RLock()
	projectId, ok := p.containerProjectMap[ctr.Id]
	if !ok {
		klog.Errorf("container project not save")
		p.containerProjectMapSync.RUnlock()
		return nil
	}
	p.containerProjectMapSync.RUnlock()
	err := p.quotaCtl.ClearQuota(projectId)
	return err
}

func parseUpperDir(mounts []mount.Mount) string {
	for _, m := range mounts {
		for _, option := range m.Options {
			if strings.HasPrefix(option, "upperdir=") {
				upperDir := strings.Replace(option, "upperdir=", "", -1)
				return filepath.Dir(upperDir)
			}
		}
	}
	return ""
}

func parseFlag() {
	flag.StringVar(&pluginName, "name", constant.DefaultPluginName, "plugin name to register to NRI")
	flag.StringVar(&pluginIdx, "idx", constant.DefaultPluginIndex, "plugin index to register to NRI")
	flag.Uint64Var(&quotaSize, "quota", constant.DefaultQuotaSize, "quota-injector default quota size is 1G")
	flag.StringVar(&containerdStateDir, "containerd-state-dir", constant.DefaultContainerdStateDir, "containerd state dir")
	flag.StringVar(&containerdRootDir, "containerd-root-dir", constant.DefaultContainerdRootDir, "containerd root dir")
	flag.StringVar(&containerdBasePath, "containerd-base-path", constant.DefaultContainerdBasePath, "containerd base path")
	flag.StringVar(&containerdSocket, "containerd-socket", constant.DefaultContainerdSocket, "containerd socket")
	flag.StringVar(&containerdNamespace, "containerd-namespace", constant.DefaultContainerdNamespace, "containerd namespace")
	flag.BoolVar(&useEphemeralStorage, "use-ephemeral-storage", false, "use pod resource ephemeral-storage to set quota size")
	flag.BoolVar(&enableLabelSelect, "enable-label-select", true, "enable label select")
	flag.Var(&filterLabelSelect, "label-select", "label select map, key=value,key1=value1")
	flag.Parse()
}

func printFlag() {
	klog.InfoS("plugin name", "name", pluginName)
	klog.InfoS("plugin index", "index", pluginIdx)
	klog.InfoS("quota size", "size", quotaSize)
	klog.InfoS("containerd state dir", "state-dir", containerdStateDir)
	klog.InfoS("containerd root dir", "root-dir", containerdRootDir)
	klog.InfoS("containerd base path", "base-path", containerdBasePath)
	klog.InfoS("containerd socket", "socket", containerdSocket)
	klog.InfoS("containerd namespace", "namespace", containerdNamespace)
	klog.InfoS("use ephemeral storage", "use-ephemeral-storage", useEphemeralStorage)
	klog.InfoS("enable label select", "enable-label-select", enableLabelSelect)
	klog.InfoS("label select map", "label-select", filterLabelSelect)
}

func main() {
	var (
		opts []stub.Option
		err  error
	)

	logs.InitLogs()
	klog.InitFlags(nil)
	klog.EnableContextualLogging(true)
	defer logs.FlushLogs()

	parseFlag()
	printFlag()

	opts = append(opts, stub.WithPluginName(pluginName))
	opts = append(opts, stub.WithPluginIdx(pluginIdx))
	go func() {
		klog.Info("Init Probe http server")
		if err := utils.InitProbe(); err != nil {
			klog.Fatalf("failed init probe: %v", err)
		}
	}()

	enabled, err := utils.IsPrjQuotaEnabled(containerdBasePath)
	if err != nil {
		klog.Fatalf("check prjquota error %s", err)
	}
	if !enabled {
		klog.Fatalf("prjquota not enabled in mount path: %s", containerdBasePath)
	}

	if err = start(opts); err != nil {
		klog.Fatalf("failed to start plugin: %v", err)
	}
}

func start(opts []stub.Option) error {
	var err error
	/*Loading config files*/
	klog.Info("Starting OS watcher.")
	sigs := utils.NewOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	qctr, err := quota.NewControl(containerdBasePath, containerdRootDir)
	if err != nil {
		klog.Errorf("quota new control error %v", err)
		os.Exit(-1)
	}
	p := &plugin{
		quotaCtl:                qctr,
		containerProjectMap:     make(map[string]uint32),
		containerProjectMapSync: sync.RWMutex{},
	}
	closeCtx := make(chan struct{})

restart:
	ctx := context.Background()
	opts = append(opts, stub.WithOnClose(func() {
		klog.Info("Stopping plugins.")
		closeCtx <- struct{}{}
	}))
	if p.stub, err = stub.New(p, opts...); err != nil {
		klog.Fatalf("failed to create plugin stub: %v", err)
	}
	go func() {
		client, err := containerd.New(containerdSocket)
		if err != nil {
			klog.Errorf("containerd client new error :%v", err)
			closeCtx <- struct{}{}
			return
		}
		p.client = client
		err = p.stub.Run(ctx)
		if err != nil {
			klog.Errorf("plugin exited with error %v", err)
			closeCtx <- struct{}{}
		}
	}()

	for {
		select {
		case <-closeCtx:
			klog.Infof("Restarting plugin.")
			time.Sleep(3 * time.Second)
			goto restart

		case <-ctx.Done():
			goto restart

		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				klog.Info("Received SIGHUP, restarting.")
				goto restart
			default:
				klog.Infof("Received signal \"%v\", shutting down.", s)
				goto exit
			}
		}
	}
exit:
	stopPlugins(p)
	return nil
}

func stopPlugins(p *plugin) {
	klog.Info("Stopping plugins.")
	p.stub.Stop()
}
