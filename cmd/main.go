package main

import (
	"context"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
)

func main() {
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		panic(err)
	}
	defer client.Close()
	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")
	c, err := client.ContainerService().Get(ctx, "a9d920cfaeb0f68ac89938e160f04427f306b01ad5c80554e45d7b7b54cbab34")
	if err != nil {
		panic(err)
	}
	fmt.Println(c.SnapshotKey, c.Snapshotter)
	mounts, err := client.SnapshotService(c.Snapshotter).Mounts(ctx, c.SnapshotKey)
	if err != nil {
		panic(err)
	}
	fmt.Println(mounts)
}
