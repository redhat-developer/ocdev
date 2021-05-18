package component

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devfile/library/pkg/devfile/generator"
	"github.com/openshift/odo/pkg/component"
	componentlabels "github.com/openshift/odo/pkg/component/labels"
	"github.com/openshift/odo/pkg/envinfo"
	"github.com/openshift/odo/pkg/kclient"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/secret"
	svc "github.com/openshift/odo/pkg/service"
	"github.com/openshift/odo/pkg/util"
	servicebinding "github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const unlink = "unlink"

type commonLinkOptions struct {
	wait             bool
	port             string
	secretName       string
	isTargetAService bool

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

// Complete completes LinkOptions after they've been created
func (o *commonLinkOptions) complete(name string, cmd *cobra.Command, args []string) (err error) {
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
		o.Context, err = genericclioptions.NewDevfileContext(cmd)
	}

	if err != nil {
		return err
	}

	o.Client, err = occlient.New()
	if err != nil {
		return err
	}

	if o.csvSupport && o.Context.EnvSpecificInfo != nil {
		serviceBindingSupport, err := o.Client.GetKubeClient().IsServiceBindingSupported()
		if err != nil {
			return err
		}

		if !serviceBindingSupport {
			return fmt.Errorf("please install Service Binding Operator to be able to create/delete a link\nrefer https://odo.dev/docs/install-service-binding-operator")
		}

		o.serviceType, o.serviceName, err = svc.IsOperatorServiceNameValid(suppliedName)
		if err != nil {
			return err
		}

		if o.operationName == unlink {
			// rest of the code is specific to link operation
			return nil
		}

		componentName := o.EnvSpecificInfo.GetName()

		deployment, err := o.KClient.GetDeploymentByName(componentName)
		if err != nil {
			return err
		}

		// make this deployment the owner of the link we're creating so that link gets deleted upon doing "odo delete"
		ownerReference := generator.GetOwnerReference(deployment)

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
				Name:      getServiceBindingName(componentName, o.serviceType, o.serviceName),
				Namespace: o.EnvSpecificInfo.GetNamespace(),
			},
			Spec: servicebinding.ServiceBindingSpec{
				DetectBindingResources: true,
				BindAsFiles:            false,
				Application: &servicebinding.Application{
					Ref: servicebinding.Ref{
						Name:     componentName,
						Group:    deploymentGVR.Group,
						Version:  deploymentGVR.Version,
						Resource: deploymentGVR.Resource,
					},
				},
			},
		}
		o.serviceBinding.SetOwnerReferences(append(o.serviceBinding.GetOwnerReferences(), ownerReference))
		return nil
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
	if o.csvSupport && o.Context.EnvSpecificInfo != nil {
		// let's validate if the service exists
		svcFullName := strings.Join([]string{o.serviceType, o.serviceName}, "/")
		svcExists, err := svc.OperatorSvcExists(o.KClient, svcFullName)
		if err != nil {
			return err
		}
		if !svcExists {
			return fmt.Errorf("Couldn't find service named %q. Refer %q to see list of running services", svcFullName, "odo service list")
		}

		if o.operationName == unlink {
			componentName := o.EnvSpecificInfo.GetName()
			serviceBindingName := getServiceBindingName(componentName, o.serviceType, o.serviceName)
			links := o.EnvSpecificInfo.GetLink()

			linked := isComponentLinked(serviceBindingName, links)
			if !linked {
				// user's trying to unlink a service that's not linked with the component
				return fmt.Errorf("failed to unlink the service %q since it's not linked with the component %q", svcFullName, componentName)
			}

			// Verify if the underlying service binding request actually exists
			serviceBindingSvcFullName := strings.Join([]string{kclient.ServiceBindingKind, serviceBindingName}, "/")
			serviceBindingExists, err := svc.OperatorSvcExists(o.KClient, serviceBindingSvcFullName)
			if err != nil {
				return err
			}
			if !serviceBindingExists {
				// This could have happened if the service binding was deleted outside odo workflow (eg: oc delete sb/<sb-name>)
				// we must remove entry of the link from env.yaml in this case
				err = o.Context.EnvSpecificInfo.DeleteLink(serviceBindingName)
				if err != nil {
					return fmt.Errorf("component's link with %q has been deleted outside odo; unable to delete odo's state of the link", svcFullName)
				}
				return fmt.Errorf("component's link with %q has been deleted outside odo", svcFullName)
			}
			return nil
		}

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

		service := servicebinding.Service{
			NamespacedRef: servicebinding.NamespacedRef{
				Ref: servicebinding.Ref{
					Group:   group,
					Version: version,
					Kind:    kind,
					Name:    o.serviceName,
				},
				Namespace: &o.KClient.Namespace,
			},
		}
		o.serviceBinding.Spec.Services = []servicebinding.Service{service}

		return nil
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
	if o.csvSupport && o.Context.EnvSpecificInfo != nil {
		if o.operationName == unlink {
			serviceBindingName := getServiceBindingName(o.EnvSpecificInfo.GetName(), o.serviceType, o.serviceName)
			svcFullName := getSvcFullName(kclient.ServiceBindingKind, serviceBindingName)
			err = svc.DeleteServiceBindingRequest(o.KClient, svcFullName)
			if err != nil {
				return err
			}

			err = o.Context.EnvSpecificInfo.DeleteLink(serviceBindingName)
			if err != nil {
				return err
			}

			log.Successf("Successfully unlinked component %q from service %q\n", o.Context.EnvSpecificInfo.GetName(), o.suppliedName)
			log.Italic("To apply the changes, please use `odo push`")

			return
		}

		// convert service binding request into a map[string]interface{} type so
		// as to use it with dynamic client
		serviceBindingMap := make(map[string]interface{})
		intermediate, err := json.Marshal(o.serviceBinding)
		if err != nil {
			return err
		}
		err = json.Unmarshal(intermediate, &serviceBindingMap)
		if err != nil {
			return err
		}

		// this creates a link by creating a service of type
		// "ServiceBindingRequest" from the Operator "ServiceBindingOperator".
		err = o.KClient.CreateDynamicResource(serviceBindingMap, kclient.ServiceBindingGroup, kclient.ServiceBindingVersion, kclient.ServiceBindingResource)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("component %q is already linked with the service %q", o.Context.EnvSpecificInfo.GetName(), o.suppliedName)
			}
			return err
		}

		// once the link is created, we need to store the information in
		// env.yaml so that subsequent odo push can create a new deployment
		// based on it
		err = o.Context.EnvSpecificInfo.SetConfiguration("link", envinfo.EnvInfoLink{Name: o.serviceBinding.GetName(), ServiceKind: o.serviceType, ServiceName: o.serviceName})
		if err != nil {
			return err
		}

		log.Successf("Successfully created link between component %q and service %q\n", o.Context.EnvSpecificInfo.GetName(), o.suppliedName)
		log.Italic("To apply the link, please use `odo push`")
		return err
	}

	linkType := "Component"
	if o.isTargetAService {
		linkType = "Service"
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
		log.Successf("%s %s has been successfully linked to the component %s\n", linkType, o.suppliedName, component)
	case "unlink":
		log.Successf("%s %s has been successfully unlinked from the component %s\n", linkType, o.suppliedName, component)
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

// getSvcFullName returns service name in the format <service-type>/<service-name>
func getSvcFullName(serviceType, serviceName string) string {
	return strings.Join([]string{serviceType, serviceName}, "/")
}

// getServiceBindingName creates a name to be used for creation/deletion of SBR during link/unlink operations
func getServiceBindingName(componentName, serviceType, serviceName string) string {
	return strings.Join([]string{componentName, strings.ToLower(serviceType), serviceName}, "-")
}

// isComponentLinked checks if link with "serviceBindingName" exists in the component's
// config. It confirms if the component is linked with the service
func isComponentLinked(serviceBindingName string, links []envinfo.EnvInfoLink) bool {
	for _, link := range links {
		if link.Name == serviceBindingName {
			return true
		}
	}
	return false
}
