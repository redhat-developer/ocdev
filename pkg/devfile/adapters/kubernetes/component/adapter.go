package component

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"

	"github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/devfile/adapters/kubernetes/utils"
	"github.com/openshift/odo/pkg/kclient"
)

// New instantiantes a component adapter
func New(adapterContext common.AdapterContext, client kclient.Client) Adapter {
	return Adapter{
		Client:         client,
		AdapterContext: adapterContext,
	}
}

// Adapter is a component adapter implementation for Kubernetes
type Adapter struct {
	Client kclient.Client
	common.AdapterContext
}

// Start updates the component if a matching component exists or creates one if it doesn't exist
func (a Adapter) Start() (err error) {
	componentName := a.ComponentName

	labels := map[string]string{
		"component": componentName,
	}

	containers := utils.GetContainers(a.Devfile)
	if len(containers) == 0 {
		return fmt.Errorf("No valid components found in the devfile")
	}

	objectMeta := kclient.CreateObjectMeta(componentName, a.Client.Namespace, labels, nil)
	podTemplateSpec := kclient.GeneratePodTemplateSpec(objectMeta, containers)
	deploymentSpec := kclient.GenerateDeploymentSpec(*podTemplateSpec)
	var containerPorts []corev1.ContainerPort
	for _, c := range deploymentSpec.Template.Spec.Containers {
		if len(containerPorts) == 0 {
			containerPorts = c.Ports
		} else {
			containerPorts = append(containerPorts, c.Ports...)
		}
	}
	serviceSpec := kclient.GenerateServiceSpec(objectMeta.Name, containerPorts)
	glog.V(3).Infof("Creating deployment %v", deploymentSpec.Template.GetName())
	glog.V(3).Infof("The component name is %v", componentName)

	if utils.ComponentExists(a.Client, componentName) {
		glog.V(3).Info("The component already exists, attempting to update it")
		_, err = a.Client.UpdateDeployment(*deploymentSpec)
		if err != nil {
			return err
		}
		glog.V(3).Infof("Successfully updated component %v", componentName)
		oldSvc, err := a.Client.KubeClient.CoreV1().Services(a.Client.Namespace).Get(componentName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		serviceSpec.ClusterIP = oldSvc.Spec.ClusterIP
		objectMetaTemp := objectMeta
		objectMetaTemp.ResourceVersion = oldSvc.GetResourceVersion()

		_, err = a.Client.UpdateService(objectMetaTemp, *serviceSpec)
		if err != nil {
			return err
		}
		glog.V(3).Infof("Successfully update Service for component %s", componentName)
	} else {
		_, err = a.Client.CreateDeployment(*deploymentSpec)
		if err != nil {
			return err
		}
		glog.V(3).Infof("Successfully created component %v", componentName)
		_, err = a.Client.CreateService(objectMeta, *serviceSpec)
		if err != nil {
			return err
		}
		glog.V(3).Infof("Successfully created Service for component %s", componentName)

	}

	podSelector := fmt.Sprintf("component=%s", componentName)
	watchOptions := metav1.ListOptions{
		LabelSelector: podSelector,
	}

	_, err = a.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for component to start")
	return err
}
