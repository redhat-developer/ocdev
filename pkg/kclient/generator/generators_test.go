package generator

import (
	"reflect"
	"strings"
	"testing"

	devfilev1 "github.com/devfile/api/pkg/apis/workspaces/v1alpha2"
	devfileParser "github.com/devfile/library/pkg/devfile/parser"
	"github.com/openshift/odo/pkg/testingutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var fakeResources corev1.ResourceRequirements

func init() {
	fakeResources = *fakeResourceRequirements()
}

func TestGetContainers(t *testing.T) {

	containerNames := []string{"testcontainer1", "testcontainer2"}
	containerImages := []string{"image1", "image2"}
	trueMountSources := true
	falseMountSources := false
	tests := []struct {
		name                  string
		containerComponents   []devfilev1.Component
		wantContainerName     string
		wantContainerImage    string
		wantContainerEnv      []corev1.EnvVar
		wantContainerVolMount []corev1.VolumeMount
		wantErr               bool
	}{
		{
			name: "Case 1: Container with default project root",
			containerComponents: []devfilev1.Component{
				{
					Name: containerNames[0],
					ComponentUnion: devfilev1.ComponentUnion{
						Container: &devfilev1.ContainerComponent{
							Container: devfilev1.Container{
								Image:        containerImages[0],
								MountSources: &trueMountSources,
							},
						},
					},
				},
			},
			wantContainerName:  containerNames[0],
			wantContainerImage: containerImages[0],
			wantContainerEnv: []corev1.EnvVar{

				{
					Name:  "PROJECTS_ROOT",
					Value: "/projects",
				},
				{
					Name:  "PROJECT_SOURCE",
					Value: "/projects/test-project",
				},
			},
			wantContainerVolMount: []corev1.VolumeMount{
				{
					Name:      "devfile-projects",
					MountPath: "/projects",
				},
			},
		},
		{
			name: "Case 2: Container with source mapping",
			containerComponents: []devfilev1.Component{
				{
					Name: containerNames[0],
					ComponentUnion: devfilev1.ComponentUnion{
						Container: &devfilev1.ContainerComponent{
							Container: devfilev1.Container{
								Image:         containerImages[0],
								MountSources:  &trueMountSources,
								SourceMapping: "/myroot",
							},
						},
					},
				},
			},
			wantContainerName:  containerNames[0],
			wantContainerImage: containerImages[0],
			wantContainerEnv: []corev1.EnvVar{

				{
					Name:  "PROJECTS_ROOT",
					Value: "/myroot",
				},
				{
					Name:  "PROJECT_SOURCE",
					Value: "/myroot/test-project",
				},
			},
			wantContainerVolMount: []corev1.VolumeMount{
				{
					Name:      "devfile-projects",
					MountPath: "/myroot",
				},
			},
		},
		{
			name: "Case 3: Container with no mount source",
			containerComponents: []devfilev1.Component{
				{
					Name: containerNames[0],
					ComponentUnion: devfilev1.ComponentUnion{
						Container: &devfilev1.ContainerComponent{
							Container: devfilev1.Container{
								Image:        containerImages[0],
								MountSources: &falseMountSources,
							},
						},
					},
				},
			},
			wantContainerName:  containerNames[0],
			wantContainerImage: containerImages[0],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			devObj := devfileParser.DevfileObj{
				Data: &testingutil.TestDevfileData{
					Components: tt.containerComponents,
				},
			}

			containers, err := GetContainers(devObj)
			// Unexpected error
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetContainers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Expected error and got an err
			if tt.wantErr && err != nil {
				return
			}

			for _, container := range containers {
				if container.Name != tt.wantContainerName {
					t.Errorf("TestGetContainers error: Name mismatch - got: %s, wanted: %s", container.Name, tt.wantContainerName)
				}
				if container.Image != tt.wantContainerImage {
					t.Errorf("TestGetContainers error: Image mismatch - got: %s, wanted: %s", container.Image, tt.wantContainerImage)
				}
				if len(container.Env) > 0 && !reflect.DeepEqual(container.Env, tt.wantContainerEnv) {
					t.Errorf("TestGetContainers error: Env mismatch - got: %+v, wanted: %+v", container.Env, tt.wantContainerEnv)
				}
				if len(container.VolumeMounts) > 0 && !reflect.DeepEqual(container.VolumeMounts, tt.wantContainerVolMount) {
					t.Errorf("TestGetContainers error: Vol Mount mismatch - got: %+v, wanted: %+v", container.VolumeMounts, tt.wantContainerVolMount)
				}
			}
		})
	}

}

func TestGetPodTemplateSpec(t *testing.T) {

	container := []corev1.Container{
		{
			Name:            "container1",
			Image:           "image1",
			ImagePullPolicy: corev1.PullAlways,

			Command: []string{"tail"},
			Args:    []string{"-f", "/dev/null"},
			Env:     []corev1.EnvVar{},
		},
	}

	volume := []corev1.Volume{
		{
			Name: "vol1",
		},
	}

	tests := []struct {
		podName        string
		namespace      string
		serviceAccount string
		labels         map[string]string
	}{
		{
			podName:        "podSpecTest",
			namespace:      "default",
			serviceAccount: "default",
			labels: map[string]string{
				"app":       "app",
				"component": "frontend",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.podName, func(t *testing.T) {

			objectMeta := GetObjectMeta(tt.podName, tt.namespace, tt.labels, nil)
			podTemplateSpecParams := PodTemplateSpecParams{
				ObjectMeta:     objectMeta,
				Containers:     container,
				Volumes:        volume,
				InitContainers: container,
			}
			podTemplateSpec := GetPodTemplateSpec(podTemplateSpecParams)

			if podTemplateSpec.Name != tt.podName {
				t.Errorf("expected %s, actual %s", tt.podName, podTemplateSpec.Name)
			}
			if podTemplateSpec.Namespace != tt.namespace {
				t.Errorf("expected %s, actual %s", tt.namespace, podTemplateSpec.Namespace)
			}
			if !hasVolumeWithName("vol1", podTemplateSpec.Spec.Volumes) {
				t.Errorf("volume with name: %s not found", "vol1")
			}
			if !reflect.DeepEqual(podTemplateSpec.Labels, tt.labels) {
				t.Errorf("expected %+v, actual %+v", tt.labels, podTemplateSpec.Labels)
			}
			if !reflect.DeepEqual(podTemplateSpec.Spec.Containers, container) {
				t.Errorf("expected %+v, actual %+v", container, podTemplateSpec.Spec.Containers)
			}
			if !reflect.DeepEqual(podTemplateSpec.Spec.InitContainers, container) {
				t.Errorf("expected %+v, actual %+v", container, podTemplateSpec.Spec.InitContainers)
			}
		})
	}
}

func TestGetPVCSpec(t *testing.T) {

	tests := []struct {
		name    string
		size    string
		wantErr bool
	}{
		{
			name:    "Case 1: Valid resource size",
			size:    "1Gi",
			wantErr: false,
		},
		{
			name:    "Case 2: Resource size missing",
			size:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			quantity, err := resource.ParseQuantity(tt.size)
			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("resource.ParseQuantity unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			pvcSpec := GetPVCSpec(quantity)
			if pvcSpec.AccessModes[0] != corev1.ReadWriteOnce {
				t.Errorf("AccessMode Error: expected %s, actual %s", corev1.ReadWriteMany, pvcSpec.AccessModes[0])
			}

			pvcSpecQuantity := pvcSpec.Resources.Requests["storage"]
			if pvcSpecQuantity.String() != quantity.String() {
				t.Errorf("pvcSpec.Resources.Requests Error: expected %v, actual %v", pvcSpecQuantity.String(), quantity.String())
			}
		})
	}
}

func TestGenerateIngressSpec(t *testing.T) {

	tests := []struct {
		name      string
		parameter IngressParams
	}{
		{
			name: "1",
			parameter: IngressParams{
				ServiceName:   "service1",
				IngressDomain: "test.1.2.3.4.nip.io",
				PortNumber: intstr.IntOrString{
					IntVal: 8080,
				},
				TLSSecretName: "testTLSSecret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ingressSpec := GetIngressSpec(tt.parameter)

			if ingressSpec.Rules[0].Host != tt.parameter.IngressDomain {
				t.Errorf("expected %s, actual %s", tt.parameter.IngressDomain, ingressSpec.Rules[0].Host)
			}

			if ingressSpec.Rules[0].HTTP.Paths[0].Backend.ServicePort != tt.parameter.PortNumber {
				t.Errorf("expected %v, actual %v", tt.parameter.PortNumber, ingressSpec.Rules[0].HTTP.Paths[0].Backend.ServicePort)
			}

			if ingressSpec.Rules[0].HTTP.Paths[0].Backend.ServiceName != tt.parameter.ServiceName {
				t.Errorf("expected %s, actual %s", tt.parameter.ServiceName, ingressSpec.Rules[0].HTTP.Paths[0].Backend.ServiceName)
			}

			if ingressSpec.TLS[0].SecretName != tt.parameter.TLSSecretName {
				t.Errorf("expected %s, actual %s", tt.parameter.TLSSecretName, ingressSpec.TLS[0].SecretName)
			}

		})
	}
}

func TestGetRouteSpec(t *testing.T) {

	tests := []struct {
		name      string
		parameter RouteParams
	}{
		{
			name: "Case 1: insecure route",
			parameter: RouteParams{
				ServiceName: "service1",
				PortNumber: intstr.IntOrString{
					IntVal: 8080,
				},
				Secure: false,
				Path:   "/test",
			},
		},
		{
			name: "Case 2: secure route",
			parameter: RouteParams{
				ServiceName: "service1",
				PortNumber: intstr.IntOrString{
					IntVal: 8080,
				},
				Secure: true,
				Path:   "/test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			routeSpec := GetRouteSpec(tt.parameter)

			if routeSpec.Port.TargetPort != tt.parameter.PortNumber {
				t.Errorf("expected %v, actual %v", tt.parameter.PortNumber, routeSpec.Port.TargetPort)
			}

			if routeSpec.To.Name != tt.parameter.ServiceName {
				t.Errorf("expected %s, actual %s", tt.parameter.ServiceName, routeSpec.To.Name)
			}

			if routeSpec.Path != tt.parameter.Path {
				t.Errorf("expected %s, actual %s", tt.parameter.Path, routeSpec.Path)
			}

			if (routeSpec.TLS != nil) != tt.parameter.Secure {
				t.Errorf("the route TLS does not match secure level %v", tt.parameter.Secure)
			}

		})
	}
}

func TestGetService(t *testing.T) {

	endpointNames := []string{"port-8080-1", "port-8080-2", "port-9090"}

	tests := []struct {
		name                string
		containerComponents []devfilev1.Component
		labels              map[string]string
		wantPorts           []corev1.ServicePort
		wantErr             bool
	}{
		{
			name: "Case 1: multiple endpoints share the same port",
			containerComponents: []devfilev1.Component{
				{
					Name: "testcontainer1",
					ComponentUnion: devfilev1.ComponentUnion{
						Container: &devfilev1.ContainerComponent{
							Endpoints: []devfilev1.Endpoint{
								{
									Name:       endpointNames[0],
									TargetPort: 8080,
								},
								{
									Name:       endpointNames[1],
									TargetPort: 8080,
								},
							},
						},
					},
				},
			},
			labels: map[string]string{},
			wantPorts: []corev1.ServicePort{
				{
					Name:       "port-8080",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
			wantErr: false,
		},
		{
			name: "Case 2: multiple endpoints have different ports",
			containerComponents: []devfilev1.Component{
				{
					Name: "testcontainer1",
					ComponentUnion: devfilev1.ComponentUnion{
						Container: &devfilev1.ContainerComponent{
							Endpoints: []devfilev1.Endpoint{
								{
									Name:       endpointNames[0],
									TargetPort: 8080,
								},
								{
									Name:       endpointNames[2],
									TargetPort: 9090,
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"component": "testcomponent",
			},
			wantPorts: []corev1.ServicePort{
				{
					Name:       "port-8080",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
				{
					Name:       "port-9090",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			devObj := devfileParser.DevfileObj{
				Data: &testingutil.TestDevfileData{
					Components: tt.containerComponents,
				},
			}

			serviceSpec, err := GetService(devObj, tt.labels)

			// Unexpected error
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Expected error and got an err
			if tt.wantErr && err != nil {
				return
			}

			if !reflect.DeepEqual(serviceSpec.Selector, tt.labels) {
				t.Errorf("expected service selector is %v, actual %v", tt.labels, serviceSpec.Selector)
			}
			if len(serviceSpec.Ports) != len(tt.wantPorts) {
				t.Errorf("expected service ports length is %v, actual %v", len(tt.wantPorts), len(serviceSpec.Ports))
			} else {
				for i := range serviceSpec.Ports {
					if serviceSpec.Ports[i].Name != tt.wantPorts[i].Name {
						t.Errorf("expected name %s, actual name %s", tt.wantPorts[i].Name, serviceSpec.Ports[i].Name)
					}
					if serviceSpec.Ports[i].Port != tt.wantPorts[i].Port {
						t.Errorf("expected port number is %v, actual %v", tt.wantPorts[i].Port, serviceSpec.Ports[i].Port)
					}
				}
			}
		})
	}

}

func TestGetBuildConfig(t *testing.T) {

	tests := []struct {
		name           string
		GitURL         string
		GitRef         string
		ImageName      string
		ImageNamespace string
	}{
		{
			name:           "Case 1: Get a Source Strategy Build Config",
			GitURL:         "url",
			GitRef:         "ref",
			ImageName:      "image",
			ImageNamespace: "namespace",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commonObjectMeta := GetObjectMeta(tt.ImageName, tt.ImageNamespace, nil, nil)
			buildStrategy := GetSourceBuildStrategy(tt.ImageName, tt.ImageNamespace)
			params := BuildConfigParams{
				CommonObjectMeta: commonObjectMeta,
				BuildStrategy:    buildStrategy,
				GitURL:           tt.GitURL,
				GitRef:           tt.GitRef,
			}
			buildConfig := GetBuildConfig(params)

			if buildConfig.Name != tt.ImageName {
				t.Error("TestGetBuildConfig error - build config name does not match")
			}

			if !strings.Contains(buildConfig.Spec.CommonSpec.Output.To.Name, tt.ImageName) {
				t.Error("TestGetBuildConfig error - build config output name does not match")
			}

			if buildConfig.Spec.Source.Git.Ref != tt.GitRef || buildConfig.Spec.Source.Git.URI != tt.GitURL {
				t.Error("TestGetBuildConfig error - build config git source does not match")
			}
		})
	}

}

func fakeResourceRequirements() *corev1.ResourceRequirements {
	var resReq corev1.ResourceRequirements

	limits := make(corev1.ResourceList)
	limits[corev1.ResourceCPU], _ = resource.ParseQuantity("0.5m")
	limits[corev1.ResourceMemory], _ = resource.ParseQuantity("300Mi")
	resReq.Limits = limits

	requests := make(corev1.ResourceList)
	requests[corev1.ResourceCPU], _ = resource.ParseQuantity("0.5m")
	requests[corev1.ResourceMemory], _ = resource.ParseQuantity("300Mi")
	resReq.Requests = requests

	return &resReq
}

func hasVolumeWithName(name string, volMounts []corev1.Volume) bool {
	for _, vm := range volMounts {
		if vm.Name == name {
			return true
		}
	}
	return false
}
