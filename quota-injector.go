/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"github.com/lengrongfu/overlayfs-quota/quota-injector/quota"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	DefaultPluginName  = "quota-injector"
	DefaultPluginIndex = "99"
	DefaultQuotaSize   = 1 * 1024 * 1024 * 1024
)

var (
	log *logrus.Logger
)

var (
	pluginName string
	pluginIdx  string
	quotaSize  uint64
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
	//func (p *plugin) CreateContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error) {
	//log.Infof("CreateContainer pod:%s", pod)
	//log.Infof("CreateContainer ctr: %s", ctr)
	log.Infof("PostCreateContainer pod id: %s, container name: %s", pod.Id, ctr.Name)
	if ctr.Name != "nginx" {
		return nil
	}

	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	c, err := p.client.ContainerService().Get(ctx, ctr.Id)
	if err != nil {
		log.Errorf("from containerID: %s get container info error : %s", ctr.Id, err)
		return err
	}
	if c.Snapshotter != containerd.DefaultSnapshotter {
		log.Warning("container is not use overlayfs snapshotter")
		return nil
	}
	mounts, err := p.client.SnapshotService(c.Snapshotter).Mounts(ctx, c.SnapshotKey)
	if err != nil {
		log.Errorf("Snapshotter get mounts error: %s", err)
		return err
	}
	upperDir := parseUpperDir(mounts)
	if upperDir == "" {
		log.Warningf("upperdir is empty string, mount info is %+v", mounts)
		return nil
	}

	upperdirProject, err := p.quotaCtl.SetProject(filepath.Join(upperDir, "fs"))
	if err != nil {
		log.Errorf("set project by target %s error %s", upperDir, err)
		return err
	}
	//q := quota.Quota{
	//	Size: quotaSize,
	//}
	//err = p.quotaCtl.SetQuota(upperDir, q)
	//if err != nil {
	//	log.Errorf("set overlayfs quota project error: %s", err)
	//	return err
	//}
	//upperdirProject, ok := p.quotaCtl.GetProject(upperDir)
	//if ok {
	//	log.Infof("upperdir upperdirProject is %d", upperdirProject)
	//	p.containerProjectMapSync.Lock()
	//	p.containerProjectMap[ctr.Id] = upperdirProject
	//	p.containerProjectMapSync.Unlock()
	//}
	p.containerProjectMapSync.Lock()
	p.containerProjectMap[ctr.Id] = upperdirProject
	p.containerProjectMapSync.Unlock()

	//lowerPath := parseLowerPath(mounts)
	//err = p.quotaCtl.SetProjectByProject(lowerPath, upperdirProject)
	//if err != nil {
	//	return err
	//}
	return p.quotaCtl.SetProjectByProject(filepath.Join(upperDir, "work"), upperdirProject)
}

func (p *plugin) PostStartContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	log.Infof("PostStartContainer pod id: %s, container name: %s", pod.Id, ctr.Name)
	if ctr.Name != "nginx" {
		return nil
	}
	rootfs := filepath.Join(fmt.Sprintf("/data/run/containerd/io.containerd.runtime.v2.task/k8s.io/%s", ctr.Id), "rootfs")
	q := quota.Quota{
		Size: quotaSize,
	}
	p.containerProjectMapSync.RLock()
	projectId, ok := p.containerProjectMap[ctr.Id]
	if !ok {
		log.Errorf("container project not save")
	}
	p.containerProjectMapSync.RUnlock()
	var err error
	if !ok {
		err = p.quotaCtl.SetQuota(rootfs, q)
	} else {
		err = p.quotaCtl.SetProjectByProject(rootfs, projectId)
	}
	if err != nil {
		log.Errorf("set overlayfs quota project error: %s", err)
		return err
	}
	if err = p.quotaCtl.SetProjectQuota(q, projectId); err != nil {
		log.Errorf("set project %d quota error %s", projectId, err)
		return err
	}
	//upperdirProject, ok := p.quotaCtl.GetProject(rootfs)
	//if !ok {
	//	log.Errorf("rootfs upperdirProject is %d", upperdirProject)
	//}
	return nil
}

func (p *plugin) RemoveContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	log.Infof("RemoveContainer pod id: %s, container name: %s", pod.Id, ctr.Name)
	p.containerProjectMapSync.RLock()
	projectId, ok := p.containerProjectMap[ctr.Id]
	if !ok {
		log.Errorf("container project not save")
		p.containerProjectMapSync.RUnlock()
		return nil
	}
	p.containerProjectMapSync.RUnlock()
	err := p.quotaCtl.ClearQuota(projectId)
	return err
}

// CreateContainer handles container creation requests.
func (p *plugin) PostStartContainer1(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	//func (p *plugin) CreateContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error) {
	//log.Infof("CreateContainer pod:%s", pod)
	//log.Infof("CreateContainer ctr: %s", ctr)
	log.Infof("pod id: %s, container name: %s", pod.Id, ctr.Name)
	if ctr.Name != "nginx" {
		return nil
	}
	taskDir := fmt.Sprintf("/data/run/containerd/io.containerd.runtime.v2.task/k8s.io/%s", ctr.Id)
	if _, err := os.Stat(taskDir); err != nil {
		log.Errorf("task dirr stat error:%s", err)
		return nil
	}

	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	c, err := p.client.ContainerService().Get(ctx, ctr.Id)
	if err != nil {
		log.Errorf("from containerID: %s get container info error : %s", ctr.Id, err)
		return err
	}
	if c.Snapshotter != containerd.DefaultSnapshotter {
		log.Warning("container is not use overlayfs snapshotter")
		return nil
	}
	mounts, err := p.client.SnapshotService(c.Snapshotter).Mounts(ctx, c.SnapshotKey)
	if err != nil {
		log.Errorf("Snapshotter get mounts error: %s", err)
		return err
	}
	upperDir := parseUpperDir(mounts)
	if upperDir == "" {
		log.Warningf("upperdir is empty string, mount info is %+v", mounts)
		return nil
	}
	rootfs := filepath.Join(fmt.Sprintf("/data/run/containerd/io.containerd.runtime.v2.task/k8s.io/%s", ctr.Id), "rootfs")
	if err = os.MkdirAll(rootfs, 0711); err != nil && !os.IsExist(err) {
		log.Errorf("make container rootfs error: %s", err)
		return err
	}

	//linkPath := filepath.Join(upperDir, "merged")
	//if _, err := os.Lstat(linkPath); err != nil {
	//	if err := os.Symlink(rootfs, linkPath); err != nil {
	//		log.Errorf("make softe link error: %s", err)
	//		return err
	//	}
	//}

	q := quota.Quota{
		Size: quotaSize,
	}
	projectId, ok := p.quotaCtl.GetProject(rootfs)
	if !ok {
		err = p.quotaCtl.SetQuota(rootfs, q)
		if err != nil {
			log.Errorf("container set overlayfs quota error: %s", err)
			return err
		}
		projectId, ok = p.quotaCtl.GetProject(rootfs)
		if !ok {
			log.Errorf("get overlayfs quota project error: %s", err)
			return err
		}
	}
	log.Infof("projectId is %d", projectId)
	time.Sleep(1)
	err = p.quotaCtl.SetQuotaByProject(upperDir, q, projectId)
	if err != nil {
		log.Errorf("set overlayfs quota project error: %s", err)
		return err
	}
	upperdirProject, ok := p.quotaCtl.GetProject(upperDir)
	if upperdirProject != projectId {
		log.Errorf("upperdirProject != projectId, %d != %d", upperdirProject, projectId)
	}

	//projectId, ok := p.quotaCtl.GetProject(upperDir)
	//if !ok {
	//	err = p.quotaCtl.SetQuota(upperDir, q)
	//	if err != nil {
	//		log.Errorf("container set overlayfs quota error: %s", err)
	//		return err
	//	}
	//	projectId, ok = p.quotaCtl.GetProject(upperDir)
	//	if !ok {
	//		log.Errorf("get overlayfs quota project error: %s", err)
	//		return err
	//	}
	//}
	//log.Infof("projectId is %d", projectId)
	//err = p.quotaCtl.SetQuotaByProject(rootfs, q, projectId)
	//if err != nil {
	//	log.Errorf("set overlayfs quota project error: %s", err)
	//	return err
	//}
	return nil
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

func parseLowerPath(mounts []mount.Mount) string {
	for _, m := range mounts {
		for _, option := range m.Options {
			if strings.HasPrefix(option, "lowerdir=") {
				lowerdir := strings.Replace(option, "lowerdir=", "", -1)
				return lowerdir
			}
		}
	}
	return ""
}

func main() {
	var (
		opts []stub.Option
		err  error
	)

	log = logrus.StandardLogger()
	log.SetFormatter(&logrus.TextFormatter{
		PadLevelText: true,
	})

	flag.StringVar(&pluginName, "name", DefaultPluginName, "plugin name to register to NRI")
	flag.StringVar(&pluginIdx, "idx", DefaultPluginIndex, "plugin index to register to NRI")
	flag.Uint64Var(&quotaSize, "quota", DefaultQuotaSize, "quota-injector default quota size is 1G")
	flag.Parse()

	if pluginName != "" {
		opts = append(opts, stub.WithPluginName(pluginName))
	}
	if pluginIdx != "" {
		opts = append(opts, stub.WithPluginIdx(pluginIdx))
	}
	qctr, err := quota.NewControl("/data")
	if err != nil {
		log.Errorf("quota new control error %v", err)
		os.Exit(-1)
	}
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Errorf("containerd client new error :%v", err)
		os.Exit(-1)
	}
	defer client.Close()
	p := &plugin{
		quotaCtl:                qctr,
		client:                  client,
		containerProjectMap:     make(map[string]uint32),
		containerProjectMapSync: sync.RWMutex{},
	}
	if p.stub, err = stub.New(p, opts...); err != nil {
		log.Fatalf("failed to create plugin stub: %v", err)
	}

	err = p.stub.Run(context.Background())
	if err != nil {
		log.Errorf("plugin exited with error %v", err)
		os.Exit(1)
	}
}
