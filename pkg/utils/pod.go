package utils

import (
	"context"
	"errors"
	"github.com/containerd/nri/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

func FilterPodByLabelSelect(pod *api.PodSandbox, filterLLabelSelect map[string]string) bool {
	selector := labels.SelectorFromSet(filterLLabelSelect)
	set := labels.Set(pod.GetLabels())
	return selector.Matches(set)
}

func GetEphemeralStorage(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) (uint64, error) {
	podInfo, err := GetClient().CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Get pod %s/%s info error: %s", pod.Namespace, pod.Name, err)
		return 0, err
	}
	for _, container := range podInfo.Spec.Containers {
		if container.Name == ctr.Name {
			if container.Resources.Limits != nil {
				if ephemeralStorage, ok := container.Resources.Limits["ephemeral-storage"]; ok {
					return uint64(ephemeralStorage.Value()), nil
				}
			}
			if container.Resources.Requests != nil {
				if ephemeralStorage, ok := container.Resources.Requests["ephemeral-storage"]; ok {
					return uint64(ephemeralStorage.Value()), nil
				}
			}
		}
	}
	return 0, errors.New("ephemeral-storage not found in pod spec")
}
