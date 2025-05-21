package utils

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
)

var (
	KubeClient kubernetes.Interface
)

func init() {
	c, err := NewClient()
	if err != nil {
		klog.Fatalf("new client error %s", err.Error())
	}
	KubeClient = c.Client
}

func GetClient() kubernetes.Interface {
	return KubeClient
}

// Client is a kubernetes client.
type Client struct {
	Client kubernetes.Interface
	QPS    float32
	Burst  int
}

// WithQPS sets the QPS of the client.
func WithQPS(qps float32) func(*Client) {
	return func(c *Client) {
		c.QPS = qps
	}
}

func WithBurst(burst int) func(*Client) {
	return func(c *Client) {
		c.Burst = burst
	}
}

// NewClientWithConfig creates a new client with a given config.
func NewClientWithConfig(config *rest.Config, opts ...func(*Client)) (*Client, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	c := &Client{
		Client: client,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// NewClient creates a new client.
func NewClient(ops ...func(*Client)) (*Client, error) {
	kubeConfigPath := os.Getenv("KUBECONFIG")
	if kubeConfigPath == "" {
		kubeConfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		klog.Infof("BuildConfigFromFlags failed for file %s: %v. Using in-cluster config.", kubeConfigPath, err)
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	}
	c, err := NewClientWithConfig(config, ops...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	return c, err
}
