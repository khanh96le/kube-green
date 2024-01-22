package daemonsets

import (
	"context"
	"encoding/json"

	kubegreenv1alpha1 "github.com/kube-green/kube-green/api/v1alpha1"
	"github.com/kube-green/kube-green/controllers/sleepinfo/resource"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type daemonsets struct {
	resource.ResourceClient
	data             []appsv1.DaemonSet
	OriginalReplicas map[string]int32
	areToSuspend     bool
}

func NewResource(ctx context.Context, res resource.ResourceClient, namespace string, originalReplicas map[string]int32) (resource.Resource, error) {
	d := daemonsets{
		ResourceClient:   res,
		OriginalReplicas: originalReplicas,
		data:             []appsv1.DaemonSet{},
		areToSuspend:     res.SleepInfo.IsDaemonsetsToSuspend(),
	}
	if !d.areToSuspend {
		return d, nil
	}
	if err := d.fetch(ctx, namespace); err != nil {
		return daemonsets{}, err
	}

	return d, nil
}

func (d daemonsets) HasResource() bool {
	return len(d.data) > 0
}

func (d daemonsets) Sleep(ctx context.Context) error {
	for _, daemonset := range d.data {
		daemonset := daemonset

		newDeploy := daemonset.DeepCopy()
		if newDeploy.Spec.Template.Spec.NodeSelector != nil {
			newDeploy.Spec.Template.Spec.NodeSelector["non-existing-node-selector"] = "true"
		} else {
			newDeploy.Spec.Template.Spec.NodeSelector = map[string]string{
				"non-existing-node-selector": "true",
			}
		}

		if err := d.Patch(ctx, &daemonset, newDeploy); err != nil {
			return err
		}
	}
	return nil
}

func (d daemonsets) WakeUp(ctx context.Context) error {
	return nil
}

func (d *daemonsets) fetch(ctx context.Context, namespace string) error {
	log := d.Log.WithValues("namespace", namespace)

	daemonsetList, err := d.getListByNamespace(ctx, namespace)
	if err != nil {
		return err
	}
	log.V(1).Info("daemonsets in namespace", "number of daemonsets", len(daemonsetList))
	afterExcludedList := d.filterExcludedDaemonset(daemonsetList)
	includedList := d.filterIncludedDaemonset(afterExcludedList)
	d.data = includedList
	return nil
}

func (d daemonsets) getListByNamespace(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error) {
	listOptions := &client.ListOptions{
		Namespace: namespace,
		Limit:     500,
	}
	daemonsets := appsv1.DaemonSetList{}
	if err := d.Client.List(ctx, &daemonsets, listOptions); err != nil {
		return daemonsets.Items, client.IgnoreNotFound(err)
	}
	return daemonsets.Items, nil
}

func (d daemonsets) filterExcludedDaemonset(daemonsetList []appsv1.DaemonSet) []appsv1.DaemonSet {
	filteredList := []appsv1.DaemonSet{}
	for _, daemonset := range daemonsetList {
		if !shouldExcludeDaemonset(daemonset, d.SleepInfo) {
			filteredList = append(filteredList, daemonset)
		}
	}
	return filteredList
}

func shouldExcludeDaemonset(daemonset appsv1.DaemonSet, sleepInfo *kubegreenv1alpha1.SleepInfo) bool {
	for _, exclusion := range sleepInfo.GetExcludeRef() {
		if exclusion.Kind == "DaemonSet" && exclusion.APIVersion == "apps/v1" && exclusion.Name != "" && daemonset.Name == exclusion.Name {
			return true
		}
		if labelMatch(daemonset.Labels, exclusion.MatchLabels) {
			return true
		}
	}

	return false
}

func (d daemonsets) filterIncludedDaemonset(daemonsetList []appsv1.DaemonSet) []appsv1.DaemonSet {
	filteredList := []appsv1.DaemonSet{}
	for _, daemonset := range daemonsetList {
		if shouldIncludeDaemonset(daemonset, d.SleepInfo) {
			filteredList = append(filteredList, daemonset)
		}
	}
	return filteredList
}

func shouldIncludeDaemonset(daemonset appsv1.DaemonSet, sleepInfo *kubegreenv1alpha1.SleepInfo) bool {
	if len(sleepInfo.GetInludeRef()) == 0 {
		return true
	}

	for _, inclusion := range sleepInfo.GetInludeRef() {
		if inclusion.Kind == "Daemonset" && inclusion.APIVersion == "apps/v1" && inclusion.Name != "" && daemonset.Name == inclusion.Name {
			return true
		}
		if labelMatch(daemonset.Labels, inclusion.MatchLabels) {
			return true
		}
	}

	return false
}

func labelMatch(labels, matchLabels map[string]string) bool {
	if len(matchLabels) == 0 {
		return false
	}

	matched := true
	for key, value := range matchLabels {
		v, ok := labels[key]
		if !ok || v != value {
			matched = false
			break
		}
	}

	return matched
}

type OriginalReplicas struct {
	Name     string `json:"name"`
	Replicas int32  `json:"replicas"`
}

func (d daemonsets) GetOriginalInfoToSave() ([]byte, error) {
	if !d.areToSuspend {
		return nil, nil
	}
	return nil, nil
}

func GetOriginalInfoToRestore(data []byte) (map[string]int32, error) {
	if data == nil {
		return map[string]int32{}, nil
	}
	originalDaemonsetsReplicas := []OriginalReplicas{}
	originalDaemonsetsReplicasData := map[string]int32{}
	if err := json.Unmarshal(data, &originalDaemonsetsReplicas); err != nil {
		return nil, err
	}
	for _, replicaInfo := range originalDaemonsetsReplicas {
		if replicaInfo.Name != "" {
			originalDaemonsetsReplicasData[replicaInfo.Name] = replicaInfo.Replicas
		}
	}
	return originalDaemonsetsReplicasData, nil
}
