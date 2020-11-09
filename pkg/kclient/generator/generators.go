package generator

import (
	"fmt"

	buildv1 "github.com/openshift/api/build/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/api/resource"

	devfilev1 "github.com/devfile/api/pkg/apis/workspaces/v1alpha2"
	devfileParser "github.com/devfile/library/pkg/devfile/parser"
)

const (
	// DevfileSourceVolumeMount is the directory to mount the volume in the container
	DevfileSourceVolumeMount = "/projects"

	// EnvProjectsRoot is the env defined for project mount in a component container when component's mountSources=true
	EnvProjectsRoot = "PROJECTS_ROOT"

	// EnvProjectsSrc is the env defined for path to the project source in a component container
	EnvProjectsSrc = "PROJECT_SOURCE"

	deploymentKind       = "Deployment"
	deploymentAPIVersion = "apps/v1"
)

// GetObjectMeta creates a common object meta
func GetObjectMeta(name, namespace string, labels, annotations map[string]string) metav1.ObjectMeta {

	objectMeta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	return objectMeta
}

// GetContainers iterates through the components in the devfile and returns a slice of the corresponding containers
func GetContainers(devfileObj devfileParser.DevfileObj) ([]corev1.Container, error) {
	var containers []corev1.Container
	for _, comp := range GetDevfileContainerComponents(devfileObj.Data) {
		envVars := convertEnvs(comp.Container.Env)
		resourceReqs := getResourceReqs(comp)
		ports := convertPorts(comp.Container.Endpoints)
		containerParams := ContainerParams{
			Name:         comp.Name,
			Image:        comp.Container.Image,
			IsPrivileged: false,
			Command:      comp.Container.Command,
			Args:         comp.Container.Args,
			EnvVars:      envVars,
			ResourceReqs: resourceReqs,
			Ports:        ports,
		}
		container := getContainer(containerParams)

		// If `mountSources: true` was set, add an empty dir volume to the container to sync the source to
		// Sync to `Container.SourceMapping` and/or devfile projects if set
		if comp.Container.MountSources == nil || *comp.Container.MountSources {
			syncRootFolder := addSyncRootFolder(container, comp.Container.SourceMapping)

			err := addSyncFolder(container, syncRootFolder, devfileObj.Data.GetProjects())
			if err != nil {
				return nil, err
			}
		}
		containers = append(containers, *container)
	}
	return containers, nil
}

// PodTemplateSpecParams is a struct that contains the required data to create a pod template spec object
type PodTemplateSpecParams struct {
	ObjectMeta     metav1.ObjectMeta
	InitContainers []corev1.Container
	Containers     []corev1.Container
	Volumes        []corev1.Volume
}

// GetPodTemplateSpec creates a pod template spec that can be used to create a deployment spec
func GetPodTemplateSpec(podTemplateSpecParams PodTemplateSpecParams) *corev1.PodTemplateSpec {
	podTemplateSpec := &corev1.PodTemplateSpec{
		ObjectMeta: podTemplateSpecParams.ObjectMeta,
		Spec: corev1.PodSpec{
			InitContainers: podTemplateSpecParams.InitContainers,
			Containers:     podTemplateSpecParams.Containers,
			Volumes:        podTemplateSpecParams.Volumes,
		},
	}

	return podTemplateSpec
}

// DeploymentSpecParams is a struct that contains the required data to create a deployment spec object
type DeploymentSpecParams struct {
	PodTemplateSpec   corev1.PodTemplateSpec
	PodSelectorLabels map[string]string
}

// GetDeploymentSpec creates a deployment spec
func GetDeploymentSpec(deployParams DeploymentSpecParams) *appsv1.DeploymentSpec {
	deploymentSpec := &appsv1.DeploymentSpec{
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: deployParams.PodSelectorLabels,
		},
		Template: deployParams.PodTemplateSpec,
	}

	return deploymentSpec
}

// GetPVCSpec creates a pvc spec
func GetPVCSpec(quantity resource.Quantity) *corev1.PersistentVolumeClaimSpec {

	pvcSpec := &corev1.PersistentVolumeClaimSpec{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: quantity,
			},
		},
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
	}

	return pvcSpec
}

// GetService iterates through the components in the devfile and returns a ServiceSpec
func GetService(devfileObj devfileParser.DevfileObj, selectorLabels map[string]string) (*corev1.ServiceSpec, error) {

	var containerPorts []corev1.ContainerPort
	containerComponents := GetDevfileContainerComponents(devfileObj.Data)
	portExposureMap := GetPortExposure(containerComponents)
	containers, err := GetContainers(devfileObj)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		for _, port := range c.Ports {
			portExist := false
			for _, entry := range containerPorts {
				if entry.ContainerPort == port.ContainerPort {
					portExist = true
					break
				}
			}
			// if Exposure == none, should not create a service for that port
			if !portExist && portExposureMap[int(port.ContainerPort)] != devfilev1.NoneEndpointExposure {
				port.Name = fmt.Sprintf("port-%v", port.ContainerPort)
				containerPorts = append(containerPorts, port)
			}
		}
	}
	serviceSpecParams := ServiceSpecParams{
		ContainerPorts: containerPorts,
		SelectorLabels: selectorLabels,
	}

	return getServiceSpec(serviceSpecParams), nil
}

// IngressParams struct for function GenerateIngressSpec
// serviceName is the name of the service for the target reference
// ingressDomain is the ingress domain to use for the ingress
// portNumber is the target port of the ingress
// Path is the path of the ingress
// TLSSecretName is the target TLS Secret name of the ingress
type IngressParams struct {
	ServiceName   string
	IngressDomain string
	PortNumber    intstr.IntOrString
	TLSSecretName string
	Path          string
}

// GetIngressSpec creates an ingress spec
func GetIngressSpec(ingressParams IngressParams) *extensionsv1.IngressSpec {
	path := "/"
	if ingressParams.Path != "" {
		path = ingressParams.Path
	}
	ingressSpec := &extensionsv1.IngressSpec{
		Rules: []extensionsv1.IngressRule{
			{
				Host: ingressParams.IngressDomain,
				IngressRuleValue: extensionsv1.IngressRuleValue{
					HTTP: &extensionsv1.HTTPIngressRuleValue{
						Paths: []extensionsv1.HTTPIngressPath{
							{
								Path: path,
								Backend: extensionsv1.IngressBackend{
									ServiceName: ingressParams.ServiceName,
									ServicePort: ingressParams.PortNumber,
								},
							},
						},
					},
				},
			},
		},
	}
	secretNameLength := len(ingressParams.TLSSecretName)
	if secretNameLength != 0 {
		ingressSpec.TLS = []extensionsv1.IngressTLS{
			{
				Hosts: []string{
					ingressParams.IngressDomain,
				},
				SecretName: ingressParams.TLSSecretName,
			},
		}
	}

	return ingressSpec
}

// RouteParams struct for function GenerateRouteSpec
// serviceName is the name of the service for the target reference
// portNumber is the target port of the ingress
// Path is the path of the route
type RouteParams struct {
	ServiceName string
	PortNumber  intstr.IntOrString
	Path        string
	Secure      bool
}

// GetRouteSpec creates a route spec
func GetRouteSpec(routeParams RouteParams) *routev1.RouteSpec {
	routePath := "/"
	if routeParams.Path != "" {
		routePath = routeParams.Path
	}
	routeSpec := &routev1.RouteSpec{
		To: routev1.RouteTargetReference{
			Kind: "Service",
			Name: routeParams.ServiceName,
		},
		Port: &routev1.RoutePort{
			TargetPort: routeParams.PortNumber,
		},
		Path: routePath,
	}

	if routeParams.Secure {
		routeSpec.TLS = &routev1.TLSConfig{
			Termination:                   routev1.TLSTerminationEdge,
			InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
		}
	}

	return routeSpec
}

// GetOwnerReference generates an ownerReference  from the deployment which can then be set as
// owner for various Kubernetes objects and ensure that when the owner object is deleted from the
// cluster, all other objects are automatically removed by Kubernetes garbage collector
func GetOwnerReference(deployment *appsv1.Deployment) metav1.OwnerReference {

	ownerReference := metav1.OwnerReference{
		APIVersion: deploymentAPIVersion,
		Kind:       deploymentKind,
		Name:       deployment.Name,
		UID:        deployment.UID,
	}

	return ownerReference
}

// BuildConfigParams is a struct to create build config
type BuildConfigParams struct {
	CommonObjectMeta metav1.ObjectMeta
	GitURL           string
	GitRef           string
	BuildStrategy    buildv1.BuildStrategy
}

// GetBuildConfig gets te build config
func GetBuildConfig(buildConfigParams BuildConfigParams) buildv1.BuildConfig {

	return buildv1.BuildConfig{
		ObjectMeta: buildConfigParams.CommonObjectMeta,
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: buildConfigParams.CommonObjectMeta.Name + ":latest",
					},
				},
				Source: buildv1.BuildSource{
					Git: &buildv1.GitBuildSource{
						URI: buildConfigParams.GitURL,
						Ref: buildConfigParams.GitRef,
					},
					Type: buildv1.BuildSourceGit,
				},
				Strategy: buildConfigParams.BuildStrategy,
			},
		},
	}
}

// GetSourceBuildStrategy gets the source build strategy
func GetSourceBuildStrategy(imageName, imageNamespace string) buildv1.BuildStrategy {
	return buildv1.BuildStrategy{
		SourceStrategy: &buildv1.SourceBuildStrategy{
			From: corev1.ObjectReference{
				Kind:      "ImageStreamTag",
				Name:      imageName,
				Namespace: imageNamespace,
			},
		},
	}
}
