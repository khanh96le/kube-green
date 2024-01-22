package sleepinfo

import (
	"context"

	"github.com/kube-green/kube-green/controllers/sleepinfo/cronjobs"
	"github.com/kube-green/kube-green/controllers/sleepinfo/daemonsets"
	"github.com/kube-green/kube-green/controllers/sleepinfo/deployments"
	"github.com/kube-green/kube-green/controllers/sleepinfo/resource"
)

const (
	DeploymentResourceName = "deployments"
	CronjobResourceName    = "cronjobs"
	DaemonsetResourceName  = "daemonsets"
)

type Resources struct {
	deployments resource.Resource
	cronjobs    resource.Resource
	daemonsets  resource.Resource
}

func NewResources(ctx context.Context, resourceClient resource.ResourceClient, namespace string, sleepInfoData SleepInfoData) (Resources, error) {
	if err := resourceClient.IsClientValid(); err != nil {
		return Resources{}, err
	}
	deployResource, err := deployments.NewResource(ctx, resourceClient, namespace, sleepInfoData.OriginalDeploymentsReplicas)
	if err != nil {
		resourceClient.Log.Error(err, "fails to init deployments")
		return Resources{}, err
	}
	cronJobResource, err := cronjobs.NewResource(ctx, resourceClient, namespace, sleepInfoData.OriginalCronJobStatus)
	if err != nil {
		resourceClient.Log.Error(err, "fails to init cronjobs")
		return Resources{}, err
	}
	daemonResource, err := daemonsets.NewResource(ctx, resourceClient, namespace, sleepInfoData.OriginalDeploymentsReplicas)
	if err != nil {
		resourceClient.Log.Error(err, "fails to init daemonsets")
		return Resources{}, err
	}

	return Resources{
		deployments: deployResource,
		cronjobs:    cronJobResource,
		daemonsets:  daemonResource,
	}, nil
}

func (r Resources) hasResources() bool {
	return r.deployments.HasResource() || r.cronjobs.HasResource() || r.daemonsets.HasResource()
}

func (r Resources) sleep(ctx context.Context) error {
	if err := r.deployments.Sleep(ctx); err != nil {
		return err
	}
	if err := r.daemonsets.Sleep(ctx); err != nil {
		return err
	}
	return r.cronjobs.Sleep(ctx)
}

func (r Resources) wakeUp(ctx context.Context) error {
	if err := r.deployments.WakeUp(ctx); err != nil {
		return err
	}
	return r.cronjobs.WakeUp(ctx)
}

func (r Resources) getOriginalResourceInfoToSave() (map[string][]byte, error) {
	newData := make(map[string][]byte)

	originalDeploymentInfo, err := r.deployments.GetOriginalInfoToSave()
	if err != nil {
		return nil, err
	}
	if originalDeploymentInfo != nil {
		newData[replicasBeforeSleepKey] = originalDeploymentInfo
	}

	originalCronJobStatus, err := r.cronjobs.GetOriginalInfoToSave()
	if err != nil {
		return nil, err
	}
	if originalCronJobStatus != nil {
		newData[originalCronjobStatusKey] = originalCronJobStatus
	}

	originalDaemonsetInfo, err := r.daemonsets.GetOriginalInfoToSave()
	if err != nil {
		return nil, err
	}
	if originalDaemonsetInfo != nil {
		newData[daemonsetNodeSelectorBeforeSleep] = originalDaemonsetInfo
	}

	return newData, nil
}

func setOriginalResourceInfoToRestoreInSleepInfo(data map[string][]byte, sleepInfoData *SleepInfoData) error {
	originalDeploymentsReplicasData, err := deployments.GetOriginalInfoToRestore(data[replicasBeforeSleepKey])
	if err != nil {
		return err
	}
	sleepInfoData.OriginalDeploymentsReplicas = originalDeploymentsReplicasData

	originalCronJobStatusData, err := cronjobs.GetOriginalInfoToRestore(data[originalCronjobStatusKey])
	if err != nil {
		return err
	}
	sleepInfoData.OriginalCronJobStatus = originalCronJobStatusData

	return nil
}
