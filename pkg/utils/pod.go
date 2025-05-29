package utils

import (
	"context"
	"errors"
	"fmt"

	"github.com/containerd/nri/pkg/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

func NamespaceName(pod *api.PodSandbox) string {
	return fmt.Sprintf("%s/%s", pod.GetNamespace(), pod.GetName())
}

func FilterPodByLabelSelect(pod *api.PodSandbox, filterLLabelSelect map[string]string) bool {
	selector := labels.SelectorFromSet(filterLLabelSelect)
	set := labels.Set(pod.GetLabels())
	return selector.Matches(set)
}

func GetResource(ctx context.Context, pod *api.PodSandbox, ctr *api.Container, resourceName string) (uint64, error) {
	podInfo, err := GetClient().CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Get pod %s/%s info error: %s", pod.Namespace, pod.Name, err)
		return 0, err
	}
	for _, container := range podInfo.Spec.Containers {
		if container.Name == ctr.Name {
			if container.Resources.Limits != nil {
				if ephemeralStorage, ok := container.Resources.Limits[v1.ResourceName(resourceName)]; ok {
					return uint64(ephemeralStorage.Value()), nil
				}
			}
			if container.Resources.Requests != nil {
				if ephemeralStorage, ok := container.Resources.Requests[v1.ResourceName(resourceName)]; ok {
					return uint64(ephemeralStorage.Value()), nil
				}
			}
		}
		if v, ok := pod.GetAnnotations()[resourceName]; ok {
			quantity, err := resource.ParseQuantity(v)
			if err != nil {
				return 0, err
			}
			return uint64(quantity.Value()), nil
		}
	}
	return 0, errors.New("ephemeral-storage not found in pod spec")
}
