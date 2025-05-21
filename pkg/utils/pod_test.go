package utils

import (
	"github.com/containerd/nri/pkg/api"
	"testing"
)

func Test_FilterPodByLabelSelect(t *testing.T) {
	filterLLabelSelect := map[string]string{"app": "nginx", "env": "prod"}
	pod := &api.PodSandbox{
		Name:      "nginx",
		Namespace: "default",
		Labels: map[string]string{
			"app":     "nginx",
			"env":     "prod",
			"version": "1.0",
		},
	}
	labelSelect := FilterPodByLabelSelect(pod, filterLLabelSelect)
	if !labelSelect {
		t.Errorf("expected pod to match label selector, but it didn't")
	}
	t.Log("pod match label selector")
}
