package component

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift/odo/pkg/component"
	componentlabels "github.com/openshift/odo/pkg/component/labels"
	"github.com/openshift/odo/pkg/kclient"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/secret"
	svc "github.com/openshift/odo/pkg/service"
	"github.com/openshift/odo/pkg/util"
	servicebinding "github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const unlink = "unlink"

type commonLinkOptions struct {
	wait             bool
	port             string
	secretName       string
	isTargetAService bool
	name             string
	bindAsFiles      bool

	devfilePath string

	suppliedName  string
	operation     func(secretName, componentName, applicationName string) error
	operationName string

	// Service Binding Operator options
	serviceBinding *servicebinding.ServiceBinding
	serviceType    string
	serviceName    string
	*genericclioptions.Context
	// choose between Operator Hub and Service Catalog. If true, Operator Hub
	csvSupport bool
}

func newCommonLinkOptions() *commonLinkOptions {
	return &commonLinkOptions{}
}

func (o *commonLinkOptions) getLinkType() string {
	linkType := "component"
	if o.isTargetAService {
		linkType = "service"
	}
	return linkType
}

// Complete completes LinkOptions after they've been created
func (o *commonLinkOptions) complete(name string, cmd *cobra.Command, args []string, context string) (err error) {
	o.csvSupport, _ = svc.IsCSVSupported()

	o.operationName = name

	suppliedName := args[0]
	o.suppliedName = suppliedName

	// we need to support both devfile based component and s2i components.
	// Let's first check if creating a devfile context is possible for the
	// command provided by the user
	_, err = genericclioptions.GetValidEnvInfo(cmd)
	if err != nil {
		// error means that we can't create a devfile context for the command
		// and must create s2i context instead
		o.Context, err = genericclioptions.NewContextCreatingAppIfNeeded(cmd)
	} else {
		o.Context, err = genericclioptions.New(genericclioptions.CreateParameters{
			Cmd:              cmd,
			DevfilePath:      component.DevfilePath,
			ComponentContext: context,
		})
	}

	if err != nil {
		return err
	}

	o.Client, err = occlient.New()
	if err != nil {
		return err
	}

	if o.Context.EnvSpecificInfo != nil {
		return o.completeForOperator()
	}

	svcExists, err := svc.SvcExists(o.Client, suppliedName, o.Application)
	if err != nil {
		// we consider this error to be non-terminal since it's entirely possible to use odo without the service catalog
		klog.V(4).Infof("Unable to determine if %s is a service. This most likely means the service catalog is not installed. Processing to only use components", suppliedName)
		svcExists = false
	}

	cmpExists, err := component.Exists(o.Client, suppliedName, o.Application)
	if err != nil {
		return fmt.Errorf("Unable to determine if component exists:\n%v", err)
	}

	if !cmpExists && !svcExists {
		return fmt.Errorf("Neither a service nor a component named %s could be located. Please create one of the two before attempting to use 'odo %s'", suppliedName, o.operationName)
	}

	o.isTargetAService = svcExists

	if svcExists {
		if cmpExists {
			klog.V(4).Infof("Both a service and component with name %s - assuming a(n) %s to the service is required", suppliedName, o.operationName)
		}

		o.secretName = suppliedName
	} else {
		secretName, err := secret.DetermineSecretName(o.Client, suppliedName, o.Application, o.port)
		if err != nil {
			return err
		}
		o.secretName = secretName
	}

	return nil
}

func (o *commonLinkOptions) validate(wait bool) (err error) {
	if o.Context.EnvSpecificInfo != nil {
		return o.validateForOperator()
	}

	if o.isTargetAService {
		// if there is a ServiceBinding, then that means there is already a secret (or there will be soon)
		// which we can link to
		_, err = o.Client.GetKubeClient().GetServiceBinding(o.secretName, o.Project)
		if err != nil {
			return fmt.Errorf("The service was not created via odo. Please delete the service and recreate it using 'odo service create %s'", o.secretName)
		}

		if wait {
			// we wait until the secret has been created on the OpenShift
			// this is done because the secret is only created after the Pod that runs the
			// service is in running state.
			// This can take a long time to occur if the image of the service has yet to be downloaded
			log.Progressf("Waiting for secret of service %s to come up", o.secretName)
			_, err = o.Client.GetKubeClient().WaitAndGetSecret(o.secretName, o.Project)
		} else {
			// we also need to check whether there is a secret with the same name as the service
			// the secret should have been created along with the secret
			_, err = o.Client.GetKubeClient().GetSecret(o.secretName, o.Project)
			if err != nil {
				return fmt.Errorf("The service %s created by 'odo service create' is being provisioned. You may have to wait a few seconds until OpenShift fully provisions it before executing 'odo %s'", o.secretName, o.operationName)
			}
		}
	}

	return
}

func (o *commonLinkOptions) run() (err error) {
	if o.Context.EnvSpecificInfo != nil {
		if o.operationName == unlink {
			return o.unlinkOperator()
		}
		return o.linkOperator()
	}

	var component string
	if o.Context.EnvSpecificInfo != nil {
		component = o.EnvSpecificInfo.GetName()
		err = o.operation(o.secretName, component, o.Application)
	} else {
		component = o.Component()
		err = o.operation(o.secretName, component, o.Application)
	}

	if err != nil {
		return err
	}

	switch o.operationName {
	case "link":
		log.Successf("The %s %s has been successfully linked to the component %s\n", o.getLinkType(), o.suppliedName, component)
	case "unlink":
		log.Successf("The %s %s has been successfully unlinked from the component %s\n", o.getLinkType(), o.suppliedName, component)
	default:
		return fmt.Errorf("unknown operation %s", o.operationName)
	}

	secret, err := o.Client.GetKubeClient().GetSecret(o.secretName, o.Project)
	if err != nil {
		return err
	}

	if len(secret.Data) == 0 {
		log.Infof("There are no secret environment variables to expose within the %s service", o.suppliedName)
	} else {
		if o.operationName == "link" {
			log.Infof("The below secret environment variables were added to the '%s' component:\n", component)
		} else {
			log.Infof("The below secret environment variables were removed from the '%s' component:\n", component)
		}

		// Output the environment variables
		for i := range secret.Data {
			fmt.Printf("· %v\n", i)
		}

		// Retrieve the first variable to use as an example.
		// Have to use a range to access the map
		var exampleEnv string
		for i := range secret.Data {
			exampleEnv = i
			break
		}

		// Output what to do next if first linking...
		if o.operationName == "link" {
			log.Italicf(`
You can now access the environment variables from within the component pod, for example:
$%s is now available as a variable within component %s`, exampleEnv, component)
		}
	}

	if o.wait {
		if err := o.waitForLinkToComplete(); err != nil {
			return err
		}
	}

	return
}

func (o *commonLinkOptions) waitForLinkToComplete() (err error) {
	var component string
	if o.csvSupport && o.Context.EnvSpecificInfo != nil {
		component = o.EnvSpecificInfo.GetName()
	} else {
		component = o.Component()
	}

	labels := componentlabels.GetLabels(component, o.Application, true)
	selectorLabels, err := util.NamespaceOpenShiftObject(labels[componentlabels.ComponentLabel], labels["app"])
	if err != nil {
		return err
	}
	podSelector := fmt.Sprintf("deploymentconfig=%s", selectorLabels)

	// first wait for the pod to be pending (meaning that the deployment is being put into effect)
	// we need this intermediate wait because there is a change that the this point could be reached
	// without Openshift having had the time to launch the new deployment
	_, err = o.Client.GetKubeClient().WaitAndGetPodWithEvents(podSelector, corev1.PodPending, "Waiting for component to redeploy")
	if err != nil {
		return err
	}

	// now wait for the pod to be running
	_, err = o.Client.GetKubeClient().WaitAndGetPodWithEvents(podSelector, corev1.PodRunning, "Waiting for component to start")
	return err
}

// getServiceBindingName creates a name to be used for creation/deletion of SBR during link/unlink operations
func (o *commonLinkOptions) getServiceBindingName(componentName string) string {
	if len(o.name) > 0 {
		return o.name
	}
	if !o.isTargetAService {
		return strings.Join([]string{componentName, o.serviceName}, "-")
	}
	return strings.Join([]string{componentName, strings.ToLower(o.serviceType), o.serviceName}, "-")
}

// completeForOperator completes the options when svc is supported
func (o *commonLinkOptions) completeForOperator() (err error) {
	o.serviceType, o.serviceName, err = svc.IsOperatorServiceNameValid(o.suppliedName)
	if err != nil {
		o.serviceName = o.suppliedName
		o.isTargetAService = false
	} else {
		o.isTargetAService = true
	}

	if o.operationName == unlink {
		// rest of the code is specific to link operation
		return nil
	}

	componentName := o.EnvSpecificInfo.GetName()

	deployment, err := o.KClient.GetOneDeployment(componentName, o.EnvSpecificInfo.GetApplication())
	if err != nil {
		return err
	}

	deploymentGVR, err := o.KClient.GetDeploymentAPIVersion()
	if err != nil {
		return err
	}

	o.serviceBinding = &servicebinding.ServiceBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: strings.Join([]string{kclient.ServiceBindingGroup, kclient.ServiceBindingVersion}, "/"),
			Kind:       kclient.ServiceBindingKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: o.getServiceBindingName(componentName),
		},
		Spec: servicebinding.ServiceBindingSpec{
			DetectBindingResources: true,
			BindAsFiles:            o.bindAsFiles,
			Application: &servicebinding.Application{
				Ref: servicebinding.Ref{
					Name:     deployment.Name,
					Group:    deploymentGVR.Group,
					Version:  deploymentGVR.Version,
					Resource: deploymentGVR.Resource,
				},
			},
		},
	}
	return nil
}

// validateForOperator validates the options when svc is supported
func (o *commonLinkOptions) validateForOperator() (err error) {
	var svcFullName string

	if o.isTargetAService {
		if !o.csvSupport {
			return fmt.Errorf("operator hub is required for linking to services")
		}
		// let's validate if the service exists
		svcFullName = strings.Join([]string{o.serviceType, o.serviceName}, "/")
		svcExists, err := svc.OperatorSvcExists(o.KClient, svcFullName)
		if err != nil {
			return err
		}
		if !svcExists {
			return fmt.Errorf("couldn't find service named %q. Refer %q to see list of running services", svcFullName, "odo service list")
		}
	} else {
		o.serviceType = "Service"
		svcFullName = o.serviceName
		if o.suppliedName == o.EnvSpecificInfo.GetName() {
			if o.operationName == unlink {
				return fmt.Errorf("the component %q cannot be unlinked from itself", o.suppliedName)
			} else {
				return fmt.Errorf("the component %q cannot be linked with itself", o.suppliedName)
			}
		}

		// TODO find the service using an app name to link components in other apps
		// requires modification of the app flag or finding some other way
		service, err := o.Context.Client.GetKubeClient().GetOneService(o.suppliedName, o.EnvSpecificInfo.GetApplication())
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("couldn't find component named %q. Refer %q to see list of running components", o.suppliedName, "odo list")
		}
		if err != nil {
			return err
		}
		o.serviceName = service.Name
	}

	if o.operationName == unlink {
		_, found, err := svc.FindDevfileServiceBinding(o.EnvSpecificInfo.GetDevfileObj(), o.serviceType, o.serviceName)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("failed to unlink the %s %q since no link was found in the configuration referring this %s", o.getLinkType(), svcFullName, o.getLinkType())
		}
		return nil
	}

	var service servicebinding.Service
	if o.isTargetAService {
		// since the service exists, let's get more info to populate service binding request
		// first get the CR itself
		cr, err := o.KClient.GetCustomResource(o.serviceType)
		if err != nil {
			return err
		}

		// now get the group, version, kind information from CR
		group, version, kind, err := svc.GetGVKFromCR(cr)
		if err != nil {
			return err
		}

		service = servicebinding.Service{
			NamespacedRef: servicebinding.NamespacedRef{
				Ref: servicebinding.Ref{
					Group:   group,
					Version: version,
					Kind:    kind,
					Name:    o.serviceName,
				},
			},
		}
	} else {
		service = servicebinding.Service{
			NamespacedRef: servicebinding.NamespacedRef{
				Ref: servicebinding.Ref{
					Version: "v1",
					Kind:    "Service",
					Name:    o.serviceName,
				},
			},
		}
	}
	o.serviceBinding.Spec.Services = []servicebinding.Service{service}

	return nil
}

// linkOperator creates a service binding resource and links
// the current component with the given odo service or
// the current component with the given component's service
// and stores the link info in the env
func (o *commonLinkOptions) linkOperator() (err error) {
	// Convert ServiceBinding -> JSON -> Map -> YAML
	// JSON conversion step is necessary to inline TypeMeta

	intermediate, err := json.Marshal(o.serviceBinding)
	if err != nil {
		return err
	}

	serviceBindingMap := make(map[string]interface{})
	err = json.Unmarshal(intermediate, &serviceBindingMap)
	if err != nil {
		return err
	}

	yamlDesc, err := yaml.Marshal(serviceBindingMap)
	if err != nil {
		return err
	}

	_, found, err := svc.FindDevfileServiceBinding(o.EnvSpecificInfo.GetDevfileObj(), o.serviceType, o.serviceName)
	if err != nil {
		return err
	}
	if found {
		return fmt.Errorf("component %q is already linked with the %s %q", o.Context.EnvSpecificInfo.GetName(), o.getLinkType(), o.suppliedName)
	}
	err = svc.AddKubernetesComponentToDevfile(string(yamlDesc), o.serviceBinding.Name, o.EnvSpecificInfo.GetDevfileObj())
	if err != nil {
		return err
	}

	log.Successf("Successfully created link between component %q and %s %q\n", o.Context.EnvSpecificInfo.GetName(), o.getLinkType(), o.suppliedName)
	log.Italic("To apply the link, please use `odo push`")
	return err
}

// unlinkOperator deletes the service binding resource from the devfile
func (o *commonLinkOptions) unlinkOperator() (err error) {

	// We already tested `found` in `validateForOperator`
	name, _, err := svc.FindDevfileServiceBinding(o.EnvSpecificInfo.GetDevfileObj(), o.serviceType, o.serviceName)
	if err != nil {
		return err
	}

	err = svc.DeleteKubernetesComponentFromDevfile(name, o.EnvSpecificInfo.GetDevfileObj())
	if err != nil {
		return err
	}

	log.Successf("Successfully unlinked component %q from %s %q\n", o.Context.EnvSpecificInfo.GetName(), o.getLinkType(), o.suppliedName)
	log.Italic("To apply the changes, please use `odo push`")
	return nil
}
