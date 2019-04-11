package occlient

import (
	"fmt"

	"github.com/golang/glog"
	appsv1 "github.com/openshift/api/apps/v1"
	appsutil "github.com/openshift/origin/pkg/apps/util"
)

func hasTag(tags []string, requiredTag string) bool {
	for _, tag := range tags {
		if tag == requiredTag {
			return true
		}
	}
	return false
}

// getDeploymentCondition returns the condition with the provided type.
// Borrowed from https://github.com/openshift/origin/blob/64349ed036ed14808124c5b4d8538b3856783b54/pkg/oc/originpolymorphichelpers/deploymentconfigs/status.go
func getDeploymentCondition(status appsv1.DeploymentConfigStatus, condType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// IsDCRolledOut indicates whether the deployment config is rolled out or not
// Borrowed from https://github.com/openshift/origin/blob/64349ed036ed14808124c5b4d8538b3856783b54/pkg/oc/originpolymorphichelpers/deploymentconfigs/status.go
func IsDCRolledOut(config *appsv1.DeploymentConfig, desiredRevision int64) bool {

	latestRevision := config.Status.LatestVersion

	if latestRevision == 0 {
		switch {
		case appsutil.HasImageChangeTrigger(config):
			glog.V(4).Infof("Deployment config %q waiting on image update", config.Name)
			return false

		case len(config.Spec.Triggers) == 0:
			fmt.Printf("Deployment config %q waiting on manual update (use 'oc rollout latest %s')", config.Name, config.Name)
			return false
		}
	}

	// We use `<` due to OpenShift at times (in rare cases) updating the DeploymentConfig multiple times via ImageTrigger
	if desiredRevision > 0 && latestRevision < desiredRevision {
		glog.V(4).Infof("Desired revision (%d) is different from the running revision (%d)", desiredRevision, latestRevision)
		return false
	}

	// Check the current condition of the deployment config
	cond := getDeploymentCondition(config.Status, appsv1.DeploymentProgressing)
	if config.Generation <= config.Status.ObservedGeneration {
		switch {
		case cond != nil && cond.Reason == "NewReplicationControllerAvailable":
			return true

		case cond != nil && cond.Reason == "ProgressDeadlineExceeded":
			return true

		case cond != nil && cond.Reason == "RolloutCancelled":
			return true

		case cond != nil && cond.Reason == "DeploymentConfigPaused":
			return true

		case config.Status.UpdatedReplicas < config.Spec.Replicas:
			glog.V(4).Infof("Waiting for rollout to finish: %d out of %d new replicas have been updated...", config.Status.UpdatedReplicas, config.Spec.Replicas)
			return false

		case config.Status.Replicas > config.Status.UpdatedReplicas:
			glog.V(4).Infof("Waiting for rollout to finish: %d old replicas are pending termination...", config.Status.Replicas-config.Status.UpdatedReplicas)
			return false

		case config.Status.AvailableReplicas < config.Status.UpdatedReplicas:
			glog.V(4).Infof("Waiting for rollout to finish: %d of %d updated replicas are available...", config.Status.AvailableReplicas, config.Status.UpdatedReplicas)
			return false
		}
	}
	return false
}
