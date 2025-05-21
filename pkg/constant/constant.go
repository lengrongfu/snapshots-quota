package constant

const (
	// DefaultPluginName is the default name for the nri plugin
	DefaultPluginName = "quota-injector"
	// DefaultPluginIndex is the default index for the nri plugin
	DefaultPluginIndex = "99"
	// DefaultQuotaSize is the default quota size for the container
	DefaultQuotaSize = 1 * 1024 * 1024 * 1024
	// DefaultContainerdStateDir is the default directory for containerd state
	DefaultContainerdStateDir = "/run/containerd"
	// DefaultContainerdBasePath is the default base path for containerd
	DefaultContainerdBasePath = "/"
	// DefaultContainerdSocket is the default socket for containerd
	DefaultContainerdSocket = "/run/containerd/containerd.sock"
	// DefaultContainerdNamespace is the default namespace for containerd
	DefaultContainerdNamespace = "k8s.io"
	// DefaultContainerdRootDir is the default root directory for containerd
	DefaultContainerdRootDir = "/var/lib/containerd"
)
