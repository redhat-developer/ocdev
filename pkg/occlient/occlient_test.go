package occlient

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"

	scv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	dockerapi "github.com/openshift/api/image/docker10"
	imagev1 "github.com/openshift/api/image/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"

	dockerapiv10 "github.com/openshift/api/image/docker10"
	applabels "github.com/redhat-developer/odo/pkg/application/labels"
	componentlabels "github.com/redhat-developer/odo/pkg/component/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ktesting "k8s.io/client-go/testing"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// fakeDeploymentConfig creates a fake DC.
// we "dog food" our own functions by using our templates / functions to generate this fake deployment config
func fakeDeploymentConfig(name string, image string, envVars []corev1.EnvVar) *appsv1.DeploymentConfig {

	// save component type as label
	labels := componentlabels.GetLabels(name, name, true)
	labels[componentlabels.ComponentTypeLabel] = image
	labels[componentlabels.ComponentTypeVersion] = "latest"

	// save source path as annotation
	annotations := map[string]string{"app.kubernetes.io/url": "github.com/foo/bar.git",
		"app.kubernetes.io/component-source-type": "git",
	}

	// Create CommonObjectMeta to be passed in
	commonObjectMeta := metav1.ObjectMeta{
		Name:        name,
		Labels:      labels,
		Annotations: annotations,
	}

	commonImageMeta := CommonImageMeta{
		Name:      name,
		Tag:       "latest",
		Namespace: "openshift",
		Ports:     []corev1.ContainerPort{corev1.ContainerPort{Name: "foo", HostPort: 80, ContainerPort: 80}},
	}

	// Generate the DeploymentConfig that will be used.
	dc := generateSupervisordDeploymentConfig(commonObjectMeta, image, commonImageMeta, envVars)

	// Add the appropriate bootstrap volumes for SupervisorD
	addBootstrapVolumeCopyInitContainer(&dc, commonObjectMeta.Name)
	addBootstrapSupervisordInitContainer(&dc, commonObjectMeta.Name)
	addBootstrapVolume(&dc, commonObjectMeta.Name)
	addBootstrapVolumeMount(&dc, commonObjectMeta.Name)

	return &dc
}

// fakeImageStream gets imagestream for the reactor
func fakeImageStream(imageName string, namespace string, strTags []string) *imagev1.ImageStream {
	var tags []imagev1.NamedTagEventList
	for _, tag := range strTags {
		tags = append(tags, imagev1.NamedTagEventList{
			Tag: tag,
			Items: []imagev1.TagEvent{
				{
					DockerImageReference: "example/" + imageName + ":" + tag,
					Generation:           1,
					Image:                "sha256:9579a93ee",
				},
			},
		})
	}

	return &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imageName,
			Namespace: namespace,
		},

		Status: imagev1.ImageStreamStatus{
			Tags: tags,
		},
	}
}

// fakeImageStreams lists the imagestreams for the reactor
func fakeImageStreams(imageName string, namespace string) *imagev1.ImageStreamList {
	return &imagev1.ImageStreamList{
		Items: []imagev1.ImageStream{*fakeImageStream(imageName, namespace, []string{"latest"})},
	}
}

// fakeImageStreamImages gets imagstreamimages for the reactor
func fakeImageStreamImages(imageName string) *imagev1.ImageStreamImage {
	mdata := &dockerapi.DockerImage{
		ContainerConfig: dockerapi.DockerConfig{
			Env: []string{
				"STI_SCRIPTS_URL=http://repo/git/" + imageName,
			},

			ExposedPorts: map[string]struct{}{
				"8080/tcp": {},
			},
		},
	}

	mdataRaw, _ := json.Marshal(mdata)
	return &imagev1.ImageStreamImage{
		Image: imagev1.Image{
			DockerImageReference: "example/" + imageName + ":latest",
			DockerImageMetadata:  runtime.RawExtension{Raw: mdataRaw},
		},
	}
}

// fakeBuildStatus is used to pass fake BuildStatus to watch
func fakeBuildStatus(status buildv1.BuildPhase, buildName string) *buildv1.Build {
	return &buildv1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      buildName,
		},
		Status: buildv1.BuildStatus{
			Phase: status,
		},
	}
}

func fakePodStatus(status corev1.PodPhase, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Status: corev1.PodStatus{
			Phase: status,
		},
	}
}

func fakeImageStreamImage(imageName string, ports []string, containerConfig string) *imagev1.ImageStreamImage {
	exposedPorts := make(map[string]struct{})
	var s struct{}
	for _, port := range ports {
		exposedPorts[port] = s
	}
	builderImage := &imagev1.ImageStreamImage{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s@@sha256:9579a93ee", imageName),
		},
		Image: imagev1.Image{
			ObjectMeta: metav1.ObjectMeta{
				Name: "@sha256:9579a93ee",
			},
			DockerImageMetadata: runtime.RawExtension{
				Object: &dockerapiv10.DockerImage{
					ContainerConfig: dockerapiv10.DockerConfig{
						ExposedPorts: exposedPorts,
					},
				},
			},
			DockerImageReference: fmt.Sprintf("docker.io/centos/%s-36-centos7@s@sha256:9579a93ee", imageName),
		},
	}
	if containerConfig != "" {
		(*builderImage).Image.DockerImageMetadata.Raw = []byte(containerConfig)
	}
	return builderImage
}

func fakePlanExternalMetaDataRaw() ([][]byte, error) {
	planExternalMetaData1 := make(map[string]string)
	planExternalMetaData1["displayName"] = "plan-name-1"

	planExternalMetaData2 := make(map[string]string)
	planExternalMetaData2["displayName"] = "plan-name-2"

	planExternalMetaDataRaw1, err := json.Marshal(planExternalMetaData1)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	planExternalMetaDataRaw2, err := json.Marshal(planExternalMetaData2)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var data [][]byte
	data = append(data, planExternalMetaDataRaw1)
	data = append(data, planExternalMetaDataRaw2)

	return data, nil
}

func fakePlanServiceInstanceCreateParameterSchemasRaw() ([][]byte, error) {
	planServiceInstanceCreateParameterSchema1 := make(map[string][]string)
	planServiceInstanceCreateParameterSchema1["required"] = []string{"PLAN_DATABASE_URI", "PLAN_DATABASE_USERNAME", "PLAN_DATABASE_PASSWORD"}

	planServiceInstanceCreateParameterSchema2 := make(map[string][]string)
	planServiceInstanceCreateParameterSchema2["required"] = []string{"PLAN_DATABASE_USERNAME_2", "PLAN_DATABASE_PASSWORD"}

	planServiceInstanceCreateParameterSchemaRaw1, err := json.Marshal(planServiceInstanceCreateParameterSchema1)
	if err != nil {
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	planServiceInstanceCreateParameterSchemaRaw2, err := json.Marshal(planServiceInstanceCreateParameterSchema2)
	if err != nil {
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	var data [][]byte
	data = append(data, planServiceInstanceCreateParameterSchemaRaw1)
	data = append(data, planServiceInstanceCreateParameterSchemaRaw2)

	return data, nil
}

func TestGetPVCNameFromVolumeMountName(t *testing.T) {
	type args struct {
		volumeMountName string
		dc              *appsv1.DeploymentConfig
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test case : Deployment config with given PVC",
			args: args{
				volumeMountName: "test-pvc",
				dc: &appsv1.DeploymentConfig{
					Spec: appsv1.DeploymentConfigSpec{
						Selector: map[string]string{
							"deploymentconfig": "test",
						},
						Template: &corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "test",
										VolumeMounts: []corev1.VolumeMount{
											{
												MountPath: "/tmp",
												Name:      "test-pvc",
											},
										},
									},
								},
								Volumes: []corev1.Volume{
									{
										Name: "test-pvc",
										VolumeSource: corev1.VolumeSource{
											PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
												ClaimName: "test-pvc",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: "test-pvc",
		},
		{
			name: "Test case : Deployment config without given PVC",
			args: args{
				volumeMountName: "non-existent-pvc",
				dc: &appsv1.DeploymentConfig{
					Spec: appsv1.DeploymentConfigSpec{
						Selector: map[string]string{
							"deploymentconfig": "test",
						},
						Template: &corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "test",
										VolumeMounts: []corev1.VolumeMount{
											{
												MountPath: "/tmp",
												Name:      "test-pvc",
											},
										},
									},
								},
								Volumes: []corev1.Volume{
									{
										Name: "test-pvc",
										VolumeSource: corev1.VolumeSource{
											PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
												ClaimName: "test-pvc",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, _ := FakeNew()

			returnValue := fakeClient.GetPVCNameFromVolumeMountName(tt.args.volumeMountName, tt.args.dc)

			// Check for validating return value
			if returnValue != tt.want {
				t.Errorf("error in return value got: %v, expected %v", returnValue, tt.want)
			}

		})
	}
}

func TestAddPVCToDeploymentConfig(t *testing.T) {
	type args struct {
		dc   *appsv1.DeploymentConfig
		pvc  string
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test case 1: valid dc",
			args: args{
				dc: &appsv1.DeploymentConfig{
					Spec: appsv1.DeploymentConfigSpec{
						Selector: map[string]string{
							"deploymentconfig": "nodejs-app",
						},
						Template: &corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "test",
										VolumeMounts: []corev1.VolumeMount{
											{
												MountPath: "/tmp",
												Name:      "test",
											},
										},
									},
								},
							},
						},
					},
				},
				pvc:  "test volume",
				path: "/mnt",
			},
			wantErr: false,
		},
		{
			name: "Test case 2: dc without Containers defined",
			args: args{
				dc: &appsv1.DeploymentConfig{
					Spec: appsv1.DeploymentConfigSpec{
						Selector: map[string]string{
							"deploymentconfig": "nodejs-app",
						},
						Template: &corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{},
						},
					},
				},
				pvc:  "test-voulme",
				path: "/mnt",
			},
			wantErr: true,
		},
		{
			name: "Test case 3: dc without Template defined",
			args: args{
				dc: &appsv1.DeploymentConfig{
					Spec: appsv1.DeploymentConfigSpec{
						Selector: map[string]string{
							"deploymentconfig": "nodejs-app",
						},
					},
				},
				pvc:  "test-voulme",
				path: "/mnt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientset := FakeNew()

			fakeClientset.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				dc := action.(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				if dc.Name != tt.args.dc.Name {
					t.Errorf("dc Name mismatch got: %s, expected %s", dc.Name, tt.args.dc.Name)
				}
				return true, nil, nil
			})
			err := fakeClient.AddPVCToDeploymentConfig(tt.args.dc, tt.args.pvc, tt.args.path)

			// Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("Client.AddPVCToDeploymentConfig() unexpected error = %v, wantErr %v", err, tt.wantErr)
			}

			// Checks for number of actions performed in positive cases
			if err == nil {
				// Check for validating actions performed
				if (len(fakeClientset.AppsClientset.Actions()) != 1) && (tt.wantErr != true) {
					t.Errorf("expected 1 action in GetPVCFromName got: %v", fakeClientset.AppsClientset.Actions())
				}

				updatedDc := fakeClientset.AppsClientset.Actions()[0].(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				found := false // creating a flag
				// iterating over the VolumeMounts for finding the one specified during func call
				for bb := range updatedDc.Spec.Template.Spec.Containers[0].VolumeMounts {
					if tt.args.path == updatedDc.Spec.Template.Spec.Containers[0].VolumeMounts[bb].MountPath {
						found = true
						if !strings.Contains(updatedDc.Spec.Template.Spec.Containers[0].VolumeMounts[bb].Name, tt.args.pvc) {
							t.Errorf("pvc name not matching with the specified value got: %v, expected %v", updatedDc.Spec.Template.Spec.Containers[0].VolumeMounts[bb].Name, tt.args.pvc)
						}
					}
				}
				if found == false {
					t.Errorf("expected Volume mount path %v not found in VolumeMounts", tt.args.path)
				}

				found = false // resetting the flag
				// iterating over the volume claims to find the one specified during func call
				for bb := range updatedDc.Spec.Template.Spec.Volumes {
					if tt.args.pvc == updatedDc.Spec.Template.Spec.Volumes[bb].VolumeSource.PersistentVolumeClaim.ClaimName {
						found = true
						if !strings.Contains(updatedDc.Spec.Template.Spec.Volumes[bb].Name, tt.args.pvc) {
							t.Errorf("pvc name not matching in PersistentVolumeClaim, got: %v, expected %v", updatedDc.Spec.Template.Spec.Volumes[bb].Name, tt.args.pvc)
						}
					}
				}
				if found == false {
					t.Errorf("expected volume %s not found in DeploymentConfig.Spec.Template.Spec.Volumes", tt.args.pvc)
				}

			}

		})
	}
}

func TestRemoveVolumeFromDeploymentConfig(t *testing.T) {
	type args struct {
		pvc    string
		dcName string
	}
	tests := []struct {
		name     string
		dcBefore *appsv1.DeploymentConfig
		args     args
		wantErr  bool
	}{
		{
			name: "Test case : 1",
			dcBefore: &appsv1.DeploymentConfig{
				Spec: appsv1.DeploymentConfigSpec{
					Selector: map[string]string{
						"deploymentconfig": "test",
					},
					Template: &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test",
									VolumeMounts: []corev1.VolumeMount{
										{
											MountPath: "/tmp",
											Name:      "test-pvc",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "test-pvc",
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: "test-pvc",
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				pvc:    "test-pvc",
				dcName: "test",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, tt.dcBefore, nil
			})
			fakeClientSet.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, nil, nil
			})
			err := fakeClient.RemoveVolumeFromDeploymentConfig(tt.args.pvc, tt.args.dcName)

			// Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.RemoveVolumeFromDeploymentConfig(pvc, dcName) unexpected error %v, wantErr %v", err, tt.wantErr)
			}
			// Check for validating number of actions performed
			if (len(fakeClientSet.AppsClientset.Actions()) != 2) && (tt.wantErr != true) {
				t.Errorf("expected 2 actions in GetPVCFromName got: %v", fakeClientSet.Kubernetes.Actions())
			}
			updatedDc := fakeClientSet.AppsClientset.Actions()[1].(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
			//	validating volume got removed from dc
			for _, volume := range updatedDc.Spec.Template.Spec.Volumes {
				if volume.PersistentVolumeClaim.ClaimName == tt.args.pvc {
					t.Errorf("expected volume with name : %v to be removed from dc", tt.args.pvc)
				}
			}
		})
	}
}

func TestGetPVCFromName(t *testing.T) {
	tests := []struct {
		name    string
		pvcName string
		wantErr bool
	}{
		{
			name:    "storage 10Gi",
			pvcName: "postgresql",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			fakeClientSet.Kubernetes.PrependReactor("get", "persistentvolumeclaims", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, nil, nil
			})

			_, err := fakeClient.GetPVCFromName(tt.pvcName)

			//Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.GetPVCFromName(name) unexpected error %v, wantErr %v", err, tt.wantErr)
			}
			// Check for validating actions performed
			if (len(fakeClientSet.Kubernetes.Actions()) != 1) && (tt.wantErr != true) {
				t.Errorf("expected 1 action in GetPVCFromName got: %v", fakeClientSet.Kubernetes.Actions())
			}
			// Check for value with which the function has called
			PVCname := fakeClientSet.Kubernetes.Actions()[0].(ktesting.GetAction).GetName()
			if PVCname != tt.pvcName {
				t.Errorf("Get action is performed with wrong pvcName, expected: %s, got %s", tt.pvcName, PVCname)

			}
		})
	}
}

func TestCreatePVC(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		labels  map[string]string
		wantErr bool
	}{
		{
			name: "storage 10Gi",
			size: "10Gi",
			labels: map[string]string{
				"name":      "mongodb",
				"namespace": "blog",
			},
			wantErr: false,
		},
		{
			name: "storage 1024",
			size: "1024",
			labels: map[string]string{
				"name":      "PostgreSQL",
				"namespace": "backend",
			},
			wantErr: false,
		},
		{
			name: "storage invalid size",
			size: "4#0",
			labels: map[string]string{
				"name":      "MySQL",
				"namespace": "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()

			_, err := fkclient.CreatePVC(tt.name, tt.size, tt.labels)

			// Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.CreatePVC(name, size, labels) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			// Check for validating actions performed
			if (len(fkclientset.Kubernetes.Actions()) != 1) && (tt.wantErr != true) {
				t.Errorf("expected 1 action in CreatePVC got: %v", fkclientset.RouteClientset.Actions())
			}
			// Checks for return values in positive cases
			if err == nil {
				createdPVC := fkclientset.Kubernetes.Actions()[0].(ktesting.CreateAction).GetObject().(*corev1.PersistentVolumeClaim)
				quantity, err := resource.ParseQuantity(tt.size)
				if err != nil {
					t.Errorf("failed to create quantity by calling resource.ParseQuantity(%v)", tt.size)
				}

				// created PVC should be labeled with labels passed to CreatePVC
				if !reflect.DeepEqual(createdPVC.Labels, tt.labels) {
					t.Errorf("labels in created route is not matching expected labels, expected: %v, got: %v", tt.labels, createdPVC.Labels)
				}
				// name, size of createdPVC should be matching to size, name passed to CreatePVC
				if !reflect.DeepEqual(createdPVC.Spec.Resources.Requests["storage"], quantity) {
					t.Errorf("size of PVC is not matching to expected size, expected: %v, got %v", quantity, createdPVC.Spec.Resources.Requests["storage"])
				}
				if createdPVC.Name != tt.name {
					t.Errorf("PVC name is not matching to expected name, expected: %s, got %s", tt.name, createdPVC.Name)
				}
			}
		})
	}
}

func TestDeletePVC(t *testing.T) {
	tests := []struct {
		name    string
		pvcName string
		wantErr bool
	}{
		{
			name:    "storage 10Gi",
			pvcName: "postgresql",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			fakeClientSet.Kubernetes.PrependReactor("delete", "persistentvolumeclaims", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, nil, nil
			})

			err := fakeClient.DeletePVC(tt.pvcName)

			//Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.DeletePVC(name) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			// Check for validating actions performed
			if (len(fakeClientSet.Kubernetes.Actions()) != 1) && (tt.wantErr != true) {
				t.Errorf("expected 1 action in DeletePVC got: %v", fakeClientSet.Kubernetes.Actions())
			}

			// Check for value with which the function has called
			DeletedPVC := fakeClientSet.Kubernetes.Actions()[0].(ktesting.DeleteAction).GetName()
			if DeletedPVC != tt.pvcName {
				t.Errorf("Delete action is performed with wrong pvcName, expected: %s, got %s", tt.pvcName, DeletedPVC)

			}
		})
	}
}

func TestCreateRoute(t *testing.T) {
	tests := []struct {
		name       string
		urlName    string
		service    string
		portNumber intstr.IntOrString
		labels     map[string]string
		wantErr    bool
	}{
		{
			name:       "Case : mailserver",
			urlName:    "mailserver",
			service:    "mailserver",
			portNumber: intstr.FromInt(8080),
			labels: map[string]string{
				"SLA":                              "High",
				"app.kubernetes.io/component-name": "backend",
				"app.kubernetes.io/component-type": "python",
			},
			wantErr: false,
		},

		{
			name:       "Case : blog (urlName is different than service)",
			urlName:    "example",
			service:    "blog",
			portNumber: intstr.FromInt(9100),
			labels: map[string]string{
				"SLA":                              "High",
				"app.kubernetes.io/component-name": "backend",
				"app.kubernetes.io/component-type": "golang",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// initialising the fakeclient
			fkclient, fkclientset := FakeNew()

			_, err := fkclient.CreateRoute(tt.urlName, tt.service, tt.portNumber, tt.labels)

			// Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.CreateRoute(string, labels) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			// Check for validating actions performed
			if len(fkclientset.RouteClientset.Actions()) != 1 {
				t.Errorf("expected 1 action in CreateRoute got: %v", fkclientset.RouteClientset.Actions())
			}
			// Checks for return values in positive cases
			if err == nil {
				createdRoute := fkclientset.RouteClientset.Actions()[0].(ktesting.CreateAction).GetObject().(*routev1.Route)
				// created route should be labeled with labels passed to CreateRoute
				if !reflect.DeepEqual(createdRoute.Labels, tt.labels) {
					t.Errorf("labels in created route is not matching expected labels, expected: %v, got: %v", tt.labels, createdRoute.Labels)
				}
				// route name and service that route is pointg to should match
				if createdRoute.Spec.To.Name != tt.service {
					t.Errorf("route is not matching to expected service name, expected: %s, got %s", tt.service, createdRoute)
				}
				if createdRoute.Name != tt.urlName {
					t.Errorf("route name is not matching to expected route name, expected: %s, got %s", tt.urlName, createdRoute.Name)
				}
				if createdRoute.Spec.To.Name != tt.service {
					t.Errorf("service name is not matching to expected service name, expected: %s, got %s", tt.service, createdRoute.Spec.To.Name)
				}
				if createdRoute.Spec.Port.TargetPort != tt.portNumber {
					t.Errorf("port number is not matching to expected port number, expected: %v, got %v", tt.portNumber, createdRoute.Spec.Port.TargetPort)
				}
			}
		})
	}
}

func TestAddLabelsToArgs(t *testing.T) {
	tests := []struct {
		name     string
		argsIn   []string
		labels   map[string]string
		argsOut1 []string
		argsOut2 []string
	}{
		{
			name:   "one label in empty args",
			argsIn: []string{},
			labels: map[string]string{
				"label1": "value1",
			},
			argsOut1: []string{
				"--labels", "label1=value1",
			},
		},
		{
			name: "one label with existing args",
			argsIn: []string{
				"--foo", "bar",
			},
			labels: map[string]string{
				"label1": "value1",
			},
			argsOut1: []string{
				"--foo", "bar",
				"--labels", "label1=value1",
			},
		},
		{
			name: "multiple label with existing args",
			argsIn: []string{
				"--foo", "bar",
			},
			labels: map[string]string{
				"label1": "value1",
				"label2": "value2",
			},
			argsOut1: []string{
				"--foo", "bar",
				"--labels", "label1=value1,label2=value2",
			},
			argsOut2: []string{
				"--foo", "bar",
				"--labels", "label2=value2,label1=value1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsGot := addLabelsToArgs(tt.labels, tt.argsIn)

			if !reflect.DeepEqual(argsGot, tt.argsOut1) && !reflect.DeepEqual(argsGot, tt.argsOut2) {
				t.Errorf("addLabelsToArgs() \ngot:  %#v \nwant: %#v or %#v", argsGot, tt.argsOut1, tt.argsOut2)
			}
		})
	}
}

func TestParseImageName(t *testing.T) {

	tests := []struct {
		arg     string
		want1   string
		want2   string
		want3   string
		want4   string
		wantErr bool
	}{
		{
			arg:     "nodejs:8",
			want1:   "",
			want2:   "nodejs",
			want3:   "8",
			want4:   "",
			wantErr: false,
		},
		{
			arg:     "nodejs@sha256:7e56ca37d1db225ebff79dd6d9fd2a9b8f646007c2afc26c67962b85dd591eb2",
			want2:   "nodejs",
			want1:   "",
			want3:   "",
			want4:   "sha256:7e56ca37d1db225ebff79dd6d9fd2a9b8f646007c2afc26c67962b85dd591eb2",
			wantErr: false,
		},
		{
			arg:     "nodejs@sha256:asdf@",
			wantErr: true,
		},
		{
			arg:     "nodejs@@",
			wantErr: true,
		},
		{
			arg:     "nodejs::",
			wantErr: true,
		},
		{
			arg:     "nodejs",
			want1:   "",
			want2:   "nodejs",
			want3:   "latest",
			want4:   "",
			wantErr: false,
		},
		{
			arg:     "",
			wantErr: true,
		},
		{
			arg:     ":",
			wantErr: true,
		},
		{
			arg:     "myproject/nodejs:8",
			want1:   "myproject",
			want2:   "nodejs",
			want3:   "8",
			want4:   "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("image name: '%s'", tt.arg)
		t.Run(name, func(t *testing.T) {
			got1, got2, got3, got4, err := ParseImageName(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseImageName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got1 != tt.want1 {
				t.Errorf("ParseImageName() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("ParseImageName() got2 = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("ParseImageName() got3 = %v, want %v", got3, tt.want3)
			}
			if got4 != tt.want4 {
				t.Errorf("ParseImageName() got4 = %v, want %v", got4, tt.want4)
			}
		})
	}
}

func TestUpdateDCAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		dcName      string
		annotations map[string]string
		existingDc  appsv1.DeploymentConfig
		wantErr     bool
	}{
		{
			name:   "existing dc",
			dcName: "nodejs",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs",
					Annotations: map[string]string{"app.kubernetes.io/url": "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "non existing dc",
			dcName: "nodejs",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildfly",
					Annotations: map[string]string{"app.kubernetes.io/url": "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()
			fkclientset.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				dcName := action.(ktesting.GetAction).GetName()
				if dcName != tt.dcName {
					return true, nil, fmt.Errorf("'get' called with a different dcName")
				}

				if tt.dcName != tt.existingDc.Name {
					return true, nil, fmt.Errorf("got different dc")
				}
				return true, &tt.existingDc, nil
			})

			fkclientset.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				dc := action.(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				if dc.Name != tt.existingDc.Name {
					return true, nil, fmt.Errorf("got different dc")
				}
				return true, dc, nil
			})

			err := fkclient.UpdateDCAnnotations(tt.dcName, tt.annotations)

			if err == nil && !tt.wantErr {
				// Check for validating actions performed
				if (len(fkclientset.AppsClientset.Actions()) != 2) && (tt.wantErr != true) {
					t.Errorf("expected 2 action in UpdateDeploymentConfig got: %v", fkclientset.AppsClientset.Actions())
				}

				updatedDc := fkclientset.AppsClientset.Actions()[1].(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				if updatedDc.Name != tt.dcName {
					t.Errorf("deploymentconfig name is not matching with expected value, expected: %s, got %s", tt.dcName, updatedDc.Name)
				}

				if !reflect.DeepEqual(updatedDc.Annotations, tt.annotations) {
					t.Errorf("deployment Config annotations not matching with expected values, expected: %s, got %s", tt.annotations, updatedDc.Annotations)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestSetupForSupervisor(t *testing.T) {
	quantity, _ := resource.ParseQuantity("1Gi")
	errQuantity, _ := resource.ParseQuantity("2Gi")

	tests := []struct {
		name        string
		dcName      string
		annotations map[string]string
		labels      map[string]string
		existingDc  appsv1.DeploymentConfig
		createdPVC  corev1.PersistentVolumeClaim
		wantErr     bool
	}{
		{
			name:   "setup with normal correct values",
			dcName: "wildfly",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			labels: map[string]string{
				"app":                              "apptmp",
				"app.kubernetes.io/component-name": "ruby",
				"app.kubernetes.io/component-type": "ruby",
				"app.kubernetes.io/name":           "apptmp",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildfly",
					Annotations: map[string]string{"app.kubernetes.io/url": "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"deploymentconfig": "wildfly",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "wildfly:latest",
									Name:  "wildfly",
								},
							},
						},
					},
				},
			},
			createdPVC: corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-s2idata", "wildfly"),
					Labels: map[string]string{
						"app":                              "apptmp",
						"app.kubernetes.io/component-name": "wildfly",
						"app.kubernetes.io/component-type": "wildfly",
						"app.kubernetes.io/name":           "apptmp",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: quantity,
						},
					},
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
				},
			},

			wantErr: false,
		},
		{
			name:   "setup with wrong pvc name",
			dcName: "wildfly",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			labels: map[string]string{
				"app":                              "apptmp",
				"app.kubernetes.io/component-name": "ruby",
				"app.kubernetes.io/component-type": "ruby",
				"app.kubernetes.io/name":           "apptmp",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildfly",
					Annotations: map[string]string{
						"app.kubernetes.io/url":                   "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"deploymentconfig": "wildfly",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "wildfly:latest",
									Name:  "wildfly",
								},
							},
						},
					},
				},
			},
			createdPVC: corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildfly",
					Labels: map[string]string{
						"app":                              "apptmp",
						"app.kubernetes.io/component-name": "wildfly",
						"app.kubernetes.io/component-type": "wildfly",
						"app.kubernetes.io/name":           "apptmp",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: quantity,
						},
					},
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
				},
			},

			wantErr: true,
		},
		{
			name:   "setup with wrong pvc specs",
			dcName: "wildfly",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			labels: map[string]string{
				"app":                              "apptmp",
				"app.kubernetes.io/component-name": "ruby",
				"app.kubernetes.io/component-type": "ruby",
				"app.kubernetes.io/name":           "apptmp",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildfly",
					Annotations: map[string]string{
						"app.kubernetes.io/url":                   "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"deploymentconfig": "wildfly",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "wildfly:latest",
									Name:  "wildfly",
								},
							},
						},
					},
				},
			},
			createdPVC: corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-s2idata", "wildfly"),
					Labels: map[string]string{
						"app":                              "apptmp",
						"app.kubernetes.io/component-name": "wildfly",
						"app.kubernetes.io/component-type": "wildfly",
						"app.kubernetes.io/name":           "apptmp",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: errQuantity,
						},
					},
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "setup with non existing dc",
			dcName: "wildfly",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			labels: map[string]string{
				"app":                              "apptmp",
				"app.kubernetes.io/component-name": "ruby",
				"app.kubernetes.io/component-type": "ruby",
				"app.kubernetes.io/name":           "apptmp",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()
			fkclientset.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				dcName := action.(ktesting.GetAction).GetName()
				if dcName != tt.dcName {
					return true, nil, fmt.Errorf("'get' called with different dcName")
				}
				if tt.dcName != tt.existingDc.Name {
					return true, nil, fmt.Errorf("got different dc")
				}
				return true, &tt.existingDc, nil
			})

			fkclientset.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				dc := action.(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				if dc.Name != tt.existingDc.Name {
					return true, nil, fmt.Errorf("got different dc")
				}
				return true, dc, nil
			})

			fkclientset.Kubernetes.PrependReactor("create", "persistentvolumeclaims", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				createdPvc := action.(ktesting.CreateAction).GetObject().(*corev1.PersistentVolumeClaim)
				if createdPvc.ObjectMeta.Name != tt.createdPVC.ObjectMeta.Name {
					return true, nil, fmt.Errorf("got a different pvc name")
				}
				if !reflect.DeepEqual(createdPvc.Spec, tt.createdPVC.Spec) {
					return true, nil, fmt.Errorf("got a different pvc spec")
				}
				return true, &tt.createdPVC, nil
			})

			err := fkclient.SetupForSupervisor(tt.dcName, tt.annotations, tt.labels)

			if err == nil && !tt.wantErr {
				// Check for validating actions performed
				if (len(fkclientset.AppsClientset.Actions()) != 2) && (tt.wantErr != true) {
					t.Errorf("expected 2 action in UpdateDeploymentConfig got: %v", fkclientset.AppsClientset.Actions())
				}

				// Check for validating actions performed
				if (len(fkclientset.Kubernetes.Actions()) != 1) && (tt.wantErr != true) {
					t.Errorf("expected 1 action in CreatePVC got: %v", fkclientset.AppsClientset.Actions())
				}

				updatedDc := fkclientset.AppsClientset.Actions()[1].(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				if updatedDc.Name != tt.dcName {
					t.Errorf("deploymentconfig name is not matching with expected value, expected: %s, got %s", tt.dcName, updatedDc.Name)
				}

				createdPVC := fkclientset.Kubernetes.Actions()[0].(ktesting.CreateAction).GetObject().(*corev1.PersistentVolumeClaim)
				if createdPVC.ObjectMeta.Name != fmt.Sprintf("%s-s2idata", tt.dcName) {
					t.Errorf("pvc name is not matching with expected value, expected: %s, got %s", tt.createdPVC.ObjectMeta.Name, createdPVC.ObjectMeta.Name)
				}

				if !reflect.DeepEqual(updatedDc.Annotations, tt.annotations) {
					t.Errorf("deployment Config annotations not matching with expected values, expected: %s, got %s", tt.annotations, updatedDc.Annotations)
				}

				if !reflect.DeepEqual(createdPVC.Name, tt.createdPVC.Name) {
					t.Errorf("created PVC not matching with expected values, expected: %v, got %v", tt.createdPVC, createdPVC)
				}

				if !reflect.DeepEqual(createdPVC.Spec, tt.createdPVC.Spec) {
					t.Errorf("created PVC spec not matching with expected values, expected: %v, got %v", createdPVC.Spec, tt.createdPVC.Spec)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestCleanupAfterSupervisor(t *testing.T) {
	tests := []struct {
		name        string
		dcName      string
		annotations map[string]string
		existingDc  appsv1.DeploymentConfig
		pvcName     string
		wantErr     bool
	}{
		{
			name:   "proper parameters and one volume,volumeMount and initContainer",
			dcName: "wildfly",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildfly",
					Annotations: map[string]string{"app.kubernetes.io/url": "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "wildfly:latest",
									Name:  "wildfly",
									VolumeMounts: []corev1.VolumeMount{
										{
											Name: getAppRootVolumeName("wildfly"),
										},
									},
								},
							},
							InitContainers: []corev1.Container{
								{
									Name:  "copy-files-to-volume",
									Image: "wildfly:latest",
									Command: []string{
										"copy-files-to-volume",
										"/opt/app-root",
										"/mnt/app-root"},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      getAppRootVolumeName("wildfly"),
											MountPath: "/mnt",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: getAppRootVolumeName("wildfly"),
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: getAppRootVolumeName("wildfly"),
										},
									},
								},
							},
						},
					},
				},
			},
			pvcName: fmt.Sprintf("%s-s2idata", "wildfly"),
			wantErr: false,
		},
		{
			name:   "proper parameters and two volume,volumeMount and initContainer",
			dcName: "wildfly",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildfly",
					Annotations: map[string]string{"app.kubernetes.io/url": "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "wildfly:latest",
									Name:  "wildfly",
									VolumeMounts: []corev1.VolumeMount{
										{
											Name: getAppRootVolumeName("wildfly"),
										},
										{
											Name: "backend",
										},
									},
								},
							},
							InitContainers: []corev1.Container{
								{
									Name:  "copy-files-to-volume",
									Image: "wildfly:latest",
									Command: []string{
										"copy-files-to-volume",
										"/opt/app-root",
										"/mnt/app-root"},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      getAppRootVolumeName("wildfly"),
											MountPath: "/mnt",
										},
									},
								},
								{
									Name:  "xyz",
									Image: "xyz:latest",
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: getAppRootVolumeName("wildfly"),
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: getAppRootVolumeName("wildfly"),
										},
									},
								},
								{
									Name: "backend",
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: "backend",
										},
									},
								},
							},
						},
					},
				},
			},
			pvcName: fmt.Sprintf("%s-s2idata", "wildfly"),
			wantErr: false,
		},
		{
			name:   "proper parameters and one volume,volumeMount and initContainer and non existing dc",
			dcName: "wildfly",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "file:///temp/nodejs-ex",
				"app.kubernetes.io/component-source-type": "local",
			},
			existingDc: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs",
					Annotations: map[string]string{"app.kubernetes.io/url": "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "wildfly:latest",
									Name:  "wildfly",
									VolumeMounts: []corev1.VolumeMount{
										{
											Name: getAppRootVolumeName("wildfly"),
										},
									},
								},
							},
							InitContainers: []corev1.Container{
								{
									Name:  "copy-files-to-volume",
									Image: "wildfly:latest",
									Command: []string{
										"copy-files-to-volume",
										"/opt/app-root",
										"/mnt/app-root"},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      getAppRootVolumeName("wildfly"),
											MountPath: "/mnt",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: getAppRootVolumeName("wildfly"),
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: getAppRootVolumeName("wildfly"),
										},
									},
								},
							},
						},
					},
				},
			},
			pvcName: fmt.Sprintf("%s-s2idata", "wildfly"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lenVolumes := len(tt.existingDc.Spec.Template.Spec.Volumes)
			lenVolumeMounts := len(tt.existingDc.Spec.Template.Spec.Containers[0].VolumeMounts)
			lenInitContainer := len(tt.existingDc.Spec.Template.Spec.InitContainers)

			fkclient, fkclientset := FakeNew()
			fkclientset.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				dcName := action.(ktesting.GetAction).GetName()
				if tt.dcName != dcName {
					return true, nil, fmt.Errorf("got different dc")
				}
				return true, &tt.existingDc, nil
			})

			fkclientset.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				dc := action.(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				if dc.Name != tt.dcName {
					return true, nil, fmt.Errorf("got different dc")
				}
				return true, dc, nil
			})

			fkclientset.Kubernetes.PrependReactor("delete", "persistentvolumeclaims", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				pvcName := action.(ktesting.DeleteAction).GetName()
				if pvcName != getAppRootVolumeName(tt.dcName) {
					return true, nil, fmt.Errorf("the pvc name is not matching the required pvc name")
				}
				return true, nil, nil
			})

			err := fkclient.CleanupAfterSupervisor(tt.dcName, tt.annotations)
			if err == nil && !tt.wantErr {
				// Check for validating actions performed
				if (len(fkclientset.AppsClientset.Actions()) != 2) && (tt.wantErr != true) {
					t.Errorf("expected 2 action in UpdateDeploymentConfig got: %v", fkclientset.AppsClientset.Actions())
				}

				// Check for validating actions performed
				if (len(fkclientset.Kubernetes.Actions()) != 1) && (tt.wantErr != true) {
					t.Errorf("expected 1 action in CreatePVC got: %v", fkclientset.Kubernetes.Actions())
				}

				updatedDc := fkclientset.AppsClientset.Actions()[1].(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				if updatedDc.Name != tt.dcName {
					t.Errorf("deploymentconfig name is not matching with expected value, expected: %s, got %s", tt.dcName, updatedDc.Name)
				}

				if !reflect.DeepEqual(updatedDc.Annotations, tt.annotations) {
					t.Errorf("deployment Config annotations not matching with expected values, expected: %s, got %s", tt.annotations, updatedDc.Annotations)
				}

				if lenVolumes == len(updatedDc.Spec.Template.Spec.Volumes) {
					t.Errorf("could not remove any volumes, expected: %s, got %s", "0 volume", "1 volume")
				}

				if lenVolumeMounts == len(updatedDc.Spec.Template.Spec.Containers[0].VolumeMounts) {
					t.Errorf("could not remove any volumeMounts, expected: %s, got %s", "0 volume", "1 volume")
				}

				if lenInitContainer == len(updatedDc.Spec.Template.Spec.InitContainers) {
					t.Errorf("could not remove any volumeMounts, expected: %s, got %s", "0 volume", "1 volume")
				}

				if lenVolumes-1 != len(updatedDc.Spec.Template.Spec.Volumes) {
					t.Errorf("wrong number of Volumes deleted, expected: %s, removed %d", "1 volume", lenVolumes-len(updatedDc.Spec.Template.Spec.Volumes))
				}

				if lenInitContainer-1 != len(updatedDc.Spec.Template.Spec.InitContainers) {
					t.Errorf("wrong number of InitContainer deleted, expected: %s, removed %d", "1 initContainer", lenInitContainer-len(updatedDc.Spec.Template.Spec.InitContainers))
				}

				if lenVolumeMounts-1 != len(updatedDc.Spec.Template.Spec.Containers[0].VolumeMounts) {
					t.Errorf("wrong number of VolumeMounts deleted, expected: %s, removed %d", "1 volumeMount", lenInitContainer-len(updatedDc.Spec.Template.Spec.Containers[0].VolumeMounts))
				}

				for _, initContainer := range updatedDc.Spec.Template.Spec.InitContainers {
					if initContainer.Name == "copy-files-to-volume" {
						t.Errorf("could not remove 'copy-volume-to-volume' InitContainer and instead some other InitContainer was removed")
					}
				}

				for _, volume := range updatedDc.Spec.Template.Spec.Volumes {
					if volume.Name == getAppRootVolumeName(tt.dcName) {
						t.Errorf("could not remove %s volume", getAppRootVolumeName(tt.dcName))
					}
				}

				for _, volumeMount := range updatedDc.Spec.Template.Spec.Containers[0].VolumeMounts {
					if volumeMount.Name == getAppRootVolumeName(tt.dcName) {
						t.Errorf("could not remove %s volumeMount", getAppRootVolumeName(tt.dcName))
					}
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestGetBuildConfigFromName(t *testing.T) {
	tests := []struct {
		name                string
		buildName           string
		returnedBuildConfig buildv1.BuildConfig
		wantErr             bool
	}{
		{
			name:      "buildConfig with existing bc",
			buildName: "nodejs",
			returnedBuildConfig: buildv1.BuildConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()

			fkclientset.BuildClientset.PrependReactor("get", "buildconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				buildName := action.(ktesting.GetAction).GetName()
				if buildName != tt.buildName {
					return true, nil, fmt.Errorf("'get' was called with wrong buildName")
				}
				return true, &tt.returnedBuildConfig, nil
			})

			build, err := fkclient.GetBuildConfigFromName(tt.buildName)
			if err == nil && !tt.wantErr {
				// Check for validating actions performed
				if (len(fkclientset.BuildClientset.Actions()) != 1) && (tt.wantErr != true) {
					t.Errorf("expected 1 action in GetBuildConfigFromName got: %v", fkclientset.AppsClientset.Actions())
				}
				if build.Name != tt.buildName {
					t.Errorf("wrong GetBuildConfigFromName got: %v, expected: %v", build.Name, tt.buildName)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestUpdateBuildConfig(t *testing.T) {
	tests := []struct {
		name                string
		buildConfigName     string
		gitURL              string
		annotations         map[string]string
		existingBuildConfig buildv1.BuildConfig
		updatedBuildConfig  buildv1.BuildConfig
		wantErr             bool
	}{
		{
			name:            "local to git with proper parameters",
			buildConfigName: "nodejs",
			gitURL:          "https://github.com/sclorg/nodejs-ex",
			annotations: map[string]string{
				"app.kubernetes.io/url":                   "https://github.com/sclorg/nodejs-ex",
				"app.kubernetes.io/component-source-type": "git",
			},
			existingBuildConfig: buildv1.BuildConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs",
				},
				Spec: buildv1.BuildConfigSpec{
					CommonSpec: buildv1.CommonSpec{},
				},
			},
			updatedBuildConfig: buildv1.BuildConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs",
					Annotations: map[string]string{
						"app.kubernetes.io/url":                   "https://github.com/sclorg/nodejs-ex",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				Spec: buildv1.BuildConfigSpec{
					CommonSpec: buildv1.CommonSpec{
						Source: buildv1.BuildSource{
							Git: &buildv1.GitBuildSource{
								URI: "https://github.com/sclorg/nodejs-ex",
							},
							Type: buildv1.BuildSourceGit,
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()
			fkclientset.BuildClientset.PrependReactor("get", "buildconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				buildConfigName := action.(ktesting.GetAction).GetName()
				if buildConfigName != tt.buildConfigName {
					return true, nil, fmt.Errorf("'update' was called with wrong buildConfig name")
				}
				return true, &tt.existingBuildConfig, nil
			})

			fkclientset.BuildClientset.PrependReactor("update", "buildconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				buildConfig := action.(ktesting.UpdateAction).GetObject().(*buildv1.BuildConfig)
				if buildConfig.Name != tt.buildConfigName {
					return true, nil, fmt.Errorf("'update' was called with wrong buildConfig name")
				}
				return true, &tt.updatedBuildConfig, nil
			})

			err := fkclient.UpdateBuildConfig(tt.buildConfigName, tt.gitURL, tt.annotations)
			if err == nil && !tt.wantErr {
				// Check for validating actions performed
				if (len(fkclientset.BuildClientset.Actions()) != 2) && (tt.wantErr != true) {
					t.Errorf("expected 2 action in GetBuildConfigFromName got: %v", fkclientset.BuildClientset.Actions())
				}

				updatedDc := fkclientset.BuildClientset.Actions()[1].(ktesting.UpdateAction).GetObject().(*buildv1.BuildConfig)
				if !reflect.DeepEqual(updatedDc.Annotations, tt.annotations) {
					t.Errorf("deployment Config annotations not matching with expected values, expected: %s, got %s", tt.annotations, updatedDc.Annotations)
				}

				if !reflect.DeepEqual(updatedDc.Spec, tt.updatedBuildConfig.Spec) {
					t.Errorf("deployment Config Spec not matching with expected values, expected: %v, got %v", tt.updatedBuildConfig.Spec, updatedDc.Spec)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestNewAppS2I(t *testing.T) {
	type args struct {
		commonObjectMeta metav1.ObjectMeta
		namespace        string
		builderImage     string
		gitURL           string
		inputPorts       []string
		envVars          []string
	}

	tests := []struct {
		name          string
		args          args
		wantedService map[int32]corev1.Protocol
		wantErr       bool
	}{
		{
			name: "case 1: with valid gitURL and two env vars",
			args: args{
				builderImage: "ruby:latest",
				namespace:    "testing",
				gitURL:       "https://github.com/openshift/ruby",
				commonObjectMeta: metav1.ObjectMeta{
					Name: "ruby",
					Labels: map[string]string{
						"app":                              "apptmp",
						"app.kubernetes.io/component-name": "ruby",
						"app.kubernetes.io/component-type": "ruby",
						"app.kubernetes.io/name":           "apptmp",
					},
					Annotations: map[string]string{
						"app.kubernetes.io/url":                   "https://github.com/openshift/ruby",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				envVars: []string{"key=value", "key1=value1"},
			},
			wantedService: map[int32]corev1.Protocol{
				8080: corev1.ProtocolTCP,
			},
			wantErr: false,
		},
		{
			name: "case 2 : binary buildSource with gitURL empty and no env vars",
			args: args{
				builderImage: "ruby:latest",
				namespace:    "testing",
				gitURL:       "",
				commonObjectMeta: metav1.ObjectMeta{
					Name: "ruby",
					Labels: map[string]string{
						"app":                              "apptmp",
						"app.kubernetes.io/component-name": "ruby",
						"app.kubernetes.io/component-type": "ruby",
						"app.kubernetes.io/name":           "apptmp",
					},
					Annotations: map[string]string{
						"app.kubernetes.io/url":                   "https://github.com/openshift/ruby",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				inputPorts: []string{"8081/tcp", "9100/udp"},
			},
			wantedService: map[int32]corev1.Protocol{
				8081: corev1.ProtocolTCP,
				9100: corev1.ProtocolUDP,
			},
			wantErr: true,
		},
		{
			name: "case 3 : with a invalid port protocol",
			args: args{
				builderImage: "ruby:latest",
				namespace:    "testing",
				gitURL:       "https://github.com/openshift/ruby",
				commonObjectMeta: metav1.ObjectMeta{
					Name: "ruby",
					Labels: map[string]string{
						"app":                              "apptmp",
						"app.kubernetes.io/component-name": "ruby",
						"app.kubernetes.io/component-type": "ruby",
						"app.kubernetes.io/name":           "apptmp",
					},
					Annotations: map[string]string{
						"app.kubernetes.io/url":                   "https://github.com/openshift/ruby",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				inputPorts: []string{"8081", "9100/blah"},
			},
			wantedService: map[int32]corev1.Protocol{
				8081: corev1.ProtocolTCP,
				9100: corev1.ProtocolUDP,
			},
			wantErr: true,
		},
		{
			name: "case 4 : with a invalid port number",
			args: args{
				builderImage: "ruby:latest",
				namespace:    "testing",
				gitURL:       "https://github.com/openshift/ruby",
				commonObjectMeta: metav1.ObjectMeta{
					Name: "ruby",
					Labels: map[string]string{
						"app":                              "apptmp",
						"app.kubernetes.io/component-name": "ruby",
						"app.kubernetes.io/component-type": "ruby",
						"app.kubernetes.io/name":           "apptmp",
					},
					Annotations: map[string]string{
						"app.kubernetes.io/url":                   "https://github.com/openshift/ruby",
						"app.kubernetes.io/component-source-type": "git",
					},
				},
				inputPorts: []string{"8ad1", "9100/Udp"},
			},
			wantedService: map[int32]corev1.Protocol{
				8081: corev1.ProtocolTCP,
				9100: corev1.ProtocolUDP,
			},
			wantErr: true,
		},

		// TODO: Currently fails. Enable this case once fixed
		// {
		// 	name: "case 3: with empty builderImage",
		// 	args: args{
		// 		name:         "ruby",
		// 		builderImage: "",
		// 		gitURL:       "https://github.com/openshift/ruby",
		// 		labels: map[string]string{
		// 			"app": "apptmp",
		// 			"app.kubernetes.io/component-name": "ruby",
		// 			"app.kubernetes.io/component-type": "ruby",
		// 			"app.kubernetes.io/name":           "apptmp",
		// 		},
		// 		annotations: map[string]string{
		// 			"app.kubernetes.io/url":                   "https://github.com/openshift/ruby",
		// 			"app.kubernetes.io/component-source-type": "git",
		// 		},
		// 	},
		// 	wantErr: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()

			fkclientset.ImageClientset.PrependReactor("list", "imagestreams", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, fakeImageStreams(tt.args.commonObjectMeta.Name, tt.args.commonObjectMeta.Namespace), nil
			})

			fkclientset.ImageClientset.PrependReactor("get", "imagestreams", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, fakeImageStream(tt.args.commonObjectMeta.Name, tt.args.commonObjectMeta.Namespace, []string{"latest"}), nil
			})

			fkclientset.ImageClientset.PrependReactor("get", "imagestreamimages", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, fakeImageStreamImages(tt.args.commonObjectMeta.Name), nil
			})

			err := fkclient.NewAppS2I(tt.args.commonObjectMeta.Name,
				tt.args.commonObjectMeta,
				tt.args.builderImage,
				tt.args.gitURL,
				tt.args.inputPorts,
				tt.args.envVars)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewAppS2I() error = %#v, wantErr %#v", err, tt.wantErr)
			}

			if err == nil {

				if len(fkclientset.BuildClientset.Actions()) != 1 {
					t.Errorf("expected 1 BuildClientset.Actions() in NewAppS2I, got %v: %v", len(fkclientset.BuildClientset.Actions()), fkclientset.BuildClientset.Actions())
				}

				if len(fkclientset.AppsClientset.Actions()) != 1 {
					t.Errorf("expected 1 AppsClientset.Actions() in NewAppS2I, got: %v", fkclientset.AppsClientset.Actions())
				}

				if len(fkclientset.Kubernetes.Actions()) != 2 {
					t.Errorf("expected 2 Kubernetes.Actions() in NewAppS2I, got: %v", fkclientset.Kubernetes.Actions())
				}

				var createdIS *imagev1.ImageStream

				if len(tt.args.inputPorts) <= 0 {
					if len(fkclientset.ImageClientset.Actions()) != 4 {
						t.Errorf("expected 4 ImageClientset.Actions() in NewAppS2I, got %v: %v", len(fkclientset.ImageClientset.Actions()), fkclientset.ImageClientset.Actions())
					}

					// Check for imagestream objects
					createdIS = fkclientset.ImageClientset.Actions()[2].(ktesting.CreateAction).GetObject().(*imagev1.ImageStream)
				} else {
					if len(fkclientset.ImageClientset.Actions()) != 1 {
						t.Errorf("expected 3 ImageClientset.Actions() in NewAppS2I, got: %v", fkclientset.ImageClientset.Actions())
					}

					// Check for imagestream objects
					createdIS = fkclientset.ImageClientset.Actions()[0].(ktesting.CreateAction).GetObject().(*imagev1.ImageStream)
				}

				if createdIS.Name != tt.args.commonObjectMeta.Name {
					t.Errorf("imagestream name is not matching with expected name, expected: %s, got %s", tt.args.commonObjectMeta.Name, createdIS.Name)
				}

				if !reflect.DeepEqual(createdIS.Labels, tt.args.commonObjectMeta.Labels) {
					t.Errorf("imagestream labels not matching with expected values, expected: %s, got %s", tt.args.commonObjectMeta.Labels, createdIS.Labels)
				}

				if !reflect.DeepEqual(createdIS.Annotations, tt.args.commonObjectMeta.Annotations) {
					t.Errorf("imagestream annotations not matching with expected values, expected: %s, got %s", tt.args.commonObjectMeta.Annotations, createdIS.Annotations)
				}

				// Check buildconfig objects
				createdBC := fkclientset.BuildClientset.Actions()[0].(ktesting.CreateAction).GetObject().(*buildv1.BuildConfig)

				if tt.args.gitURL != "" {
					if createdBC.Spec.CommonSpec.Source.Git.URI != tt.args.gitURL {
						t.Errorf("git url is not matching with expected value, expected: %s, got %s", tt.args.gitURL, createdBC.Spec.CommonSpec.Source.Git.URI)
					}

					if createdBC.Spec.CommonSpec.Source.Type != "Git" {
						t.Errorf("BuildSource type is not Git as expected")
					}
				}

				// TODO: Enable once Issue #594 fixed
				// } else if createdBC.Spec.CommonSpec.Source.Type != "Binary" {
				// 	t.Errorf("BuildSource type is not Binary as expected")
				// }

				// Check deploymentconfig objects
				createdDC := fkclientset.AppsClientset.Actions()[0].(ktesting.CreateAction).GetObject().(*appsv1.DeploymentConfig)
				if createdDC.Spec.Selector["deploymentconfig"] != tt.args.commonObjectMeta.Name {
					t.Errorf("deploymentconfig name is not matching with expected value, expected: %s, got %s", tt.args.commonObjectMeta.Name, createdDC.Spec.Selector["deploymentconfig"])
				}

				createdSvc := fkclientset.Kubernetes.Actions()[0].(ktesting.CreateAction).GetObject().(*corev1.Service)

				for port, protocol := range tt.wantedService {
					found := false
					for _, servicePort := range createdSvc.Spec.Ports {
						if servicePort.Port == port {
							found = true
							if servicePort.Protocol != protocol {
								t.Errorf("port protocol not matching, expected: %v, got %v", protocol, servicePort.Protocol)
							}
						}
					}
					if !found {
						t.Errorf("%v port with %v protocol not found", port, protocol)
						break
					}
				}
			}
		})
	}
}

func TestIsTagInImageStream(t *testing.T) {
	tests := []struct {
		name        string
		imagestream imagev1.ImageStream
		imageTag    string
		wantErr     bool
		want        bool
	}{
		{
			name:        "Case: Valid image and image tag",
			imagestream: *fakeImageStream("foo", "openshift", []string{"latest", "3.5"}),
			imageTag:    "3.5",
			want:        true,
		},
		{
			name:        "Case: Invalid image tag",
			imagestream: *fakeImageStream("bar", "testing", []string{"latest"}),
			imageTag:    "0.1",
			want:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := isTagInImageStream(tt.imagestream, tt.imageTag)

			if got != tt.want {
				t.Errorf("GetImageStream() = %#v, want %#v\n\n", got, tt)
			}
		})
	}
}

func TestGetExposedPorts(t *testing.T) {
	tests := []struct {
		name             string
		imageTag         string
		imageStreamImage *imagev1.ImageStreamImage
		wantErr          bool
		want             []corev1.ContainerPort
	}{
		{
			name:             "Case: Valid image ports",
			imageTag:         "3.5",
			imageStreamImage: fakeImageStreamImage("python", []string{"8080/tcp"}, ""),
			want: []corev1.ContainerPort{
				{
					Name:          fmt.Sprintf("%d-%s", 8080, strings.ToLower(string("tcp"))),
					ContainerPort: 8080,
					Protocol:      "TCP",
				},
			},
		},
		{
			name:             "Case: Invalid image tag",
			imageTag:         "0.1",
			imageStreamImage: fakeImageStreamImage("python", []string{"8080---tcp"}, ""),
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, _ := FakeNew()
			fkclient.Namespace = "testing"
			got, err := fkclient.GetExposedPorts(tt.imageStreamImage)

			if !tt.wantErr == (err != nil) {
				t.Errorf("client.GetExposedPorts(imagestream imageTag) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.GetExposedPorts = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestGetSecret(t *testing.T) {
	tests := []struct {
		name       string
		secretNS   string
		secretName string
		wantErr    bool
		want       *corev1.Secret
	}{
		{
			name:       "Case: Valid request for retrieving a secret",
			secretNS:   "",
			secretName: "foo",
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			wantErr: false,
		},
		{
			name:       "Case: Invalid request for retrieving a secret",
			secretNS:   "",
			secretName: "foo2",
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			// Fake getting Secret
			fakeClientSet.Kubernetes.PrependReactor("get", "secrets", func(action ktesting.Action) (bool, runtime.Object, error) {
				if tt.want.Name != tt.secretName {
					return true, nil, fmt.Errorf("'get' called with a different secret name")
				}
				return true, tt.want, nil
			})

			returnValue, err := fakeClient.GetSecret(tt.secretName, tt.secretNS)

			// Check for validating return value
			if err == nil && returnValue != tt.want {
				t.Errorf("error in return value got: %v, expected %v", returnValue, tt.want)
			}

			if !tt.wantErr == (err != nil) {
				t.Errorf("\nclient.GetSecret(secretNS, secretName) unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateServiceBinding(t *testing.T) {
	tests := []struct {
		name        string
		bindingNS   string
		bindingName string
		wantErr     bool
	}{
		{
			name:        "Case: Valid request for creating a secret",
			bindingNS:   "",
			bindingName: "foo",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			err := fakeClient.CreateServiceBinding(tt.bindingName, tt.bindingNS)

			if err == nil && !tt.wantErr {
				if len(fakeClientSet.ServiceCatalogClientSet.Actions()) != 1 {
					t.Errorf("expected 1 ServiceCatalogClientSet.Actions() in CreateServiceBinding, got: %v", fakeClientSet.ServiceCatalogClientSet.Actions())
				}
				createdBinding := fakeClientSet.ServiceCatalogClientSet.Actions()[0].(ktesting.CreateAction).GetObject().(*scv1beta1.ServiceBinding)
				if createdBinding.Name != tt.bindingName {
					t.Errorf("the name of servicebinding was not correct, expected: %s, got: %s", tt.bindingName, createdBinding.Name)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}

		})
	}
}

func TestLinkSecret(t *testing.T) {
	tests := []struct {
		name            string
		namespace       string
		secretName      string
		componentName   string
		applicationName string
		wantErr         bool
	}{
		{
			name:            "Case: Unable to locate DeploymentConfig",
			namespace:       "foo",
			secretName:      "foo",
			componentName:   "foo",
			applicationName: "",
			wantErr:         true,
		},
		{
			name:            "Case: Unable to update DeploymentConfig",
			namespace:       "",
			secretName:      "foo",
			componentName:   "foo",
			applicationName: "foo",
			wantErr:         true,
		},
		//TODO fix this somehow? It doesn't seem to work because of Instantiate doesn't play nice with fake
		//{
		//	name:        "Case: Valid creation of link",
		//	namespace: "foo",
		//	secretName: "foo",
		//	applicationName: "foo",
		//	wantErr: false,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			dcName := fmt.Sprintf("%s-%s", tt.componentName, tt.applicationName)

			// Fake getting DC
			fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				if len(tt.applicationName) == 0 {
					return true, nil, fmt.Errorf("could not find dc")
				}
				return true, fakeDeploymentConfig(dcName, "foo", []corev1.EnvVar{}), nil
			})

			// Fake updating DC
			fakeClientSet.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				if len(tt.namespace) == 0 {
					return true, nil, fmt.Errorf("could not update dc")
				}
				return true, fakeDeploymentConfig(dcName, "foo", []corev1.EnvVar{}), nil
			})

			err := fakeClient.LinkSecret(tt.secretName, tt.componentName, tt.applicationName, tt.namespace)
			if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			} else if err == nil && !tt.wantErr {
				if len(fakeClientSet.AppsClientset.Actions()) != 2 {
					t.Errorf("expected 2 AppsClientset.Actions() in LinkSecret, got: %v", fakeClientSet.AppsClientset.Actions())
				}
				//TODO enhance test if the Instantiate issue can be overcome
			}
		})
	}
}

func TestGetImageStream(t *testing.T) {
	tests := []struct {
		name           string
		imageNS        string
		imageName      string
		imageTag       string
		wantErr        bool
		want           *imagev1.ImageStream
		wantActionsCnt int
	}{
		{
			name:           "Case: Valid request for imagestream of latest version and not namespace qualified",
			imageNS:        "",
			imageName:      "foo",
			imageTag:       "latest",
			want:           fakeImageStream("foo", "testing", []string{"latest"}),
			wantActionsCnt: 1,
		},
		{
			name:           "Case: Valid explicit request for specific namespace qualified imagestream of specific version",
			imageNS:        "openshift",
			imageName:      "foo",
			imageTag:       "latest",
			want:           fakeImageStream("foo", "openshift", []string{"latest", "3.5"}),
			wantActionsCnt: 1,
		},
		{
			name:           "Case: Valid request for specific imagestream of specific version not in current namespace",
			imageNS:        "",
			imageName:      "foo",
			imageTag:       "3.5",
			want:           fakeImageStream("foo", "openshift", []string{"latest", "3.5"}),
			wantActionsCnt: 1, // Ideally supposed to be 2 but bcoz prependreactor is not parameter sensitive, the way it is mocked makes it 1
		},
		{
			name:           "Case: Invalid request for non-current and non-openshift namespace imagestream/Non-existant imagestream",
			imageNS:        "foo",
			imageName:      "bar",
			imageTag:       "3.5",
			wantErr:        true,
			wantActionsCnt: 1,
		},
		{
			name:           "Case: Request for non-existant tag",
			imageNS:        "",
			imageName:      "foo",
			imageTag:       "3.6",
			wantErr:        true,
			wantActionsCnt: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()
			fkclient.Namespace = "testing"
			openshiftIS := fakeImageStream(tt.imageName, "openshift", []string{"latest", "3.5"})
			currentNSIS := fakeImageStream(tt.imageName, "testing", []string{"latest"})

			fkclientset.ImageClientset.PrependReactor("get", "imagestreams", func(action ktesting.Action) (bool, runtime.Object, error) {
				if tt.imageNS == "" {
					if isTagInImageStream(*fakeImageStream("foo", "testing", []string{"latest"}), tt.imageTag) {
						return true, currentNSIS, nil
					} else if isTagInImageStream(*fakeImageStream("foo", "openshift", []string{"latest", "3.5"}), tt.imageTag) {
						return true, openshiftIS, nil
					}
					return true, nil, fmt.Errorf("Requested imagestream %s with tag %s not found", tt.imageName, tt.imageTag)
				}
				if tt.imageNS == "testing" {
					return true, currentNSIS, nil
				}
				if tt.imageNS == "openshift" {
					return true, openshiftIS, nil
				}
				return true, nil, fmt.Errorf("Requested imagestream %s with tag %s not found", tt.imageName, tt.imageTag)
			})

			got, err := fkclient.GetImageStream(tt.imageNS, tt.imageName, tt.imageTag)
			if len(fkclientset.ImageClientset.Actions()) != tt.wantActionsCnt {
				t.Errorf("expected %d ImageClientset.Actions() in GetImageStream, got %v", tt.wantActionsCnt, fkclientset.ImageClientset.Actions())
			}
			if !tt.wantErr == (err != nil) {
				t.Errorf("\nclient.GetImageStream(imageNS, imageName, imageTag) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetImageStream() = %#v, want %#v and the current project name is %s\n\n", got, tt, fkclient.GetCurrentProjectName())
			}
		})
	}
}

func TestGetImageStreams(t *testing.T) {

	type args struct {
		name      string
		namespace string
	}

	tests := []struct {
		name    string
		args    args
		want    []imagev1.ImageStream
		wantErr bool
	}{
		{
			name: "case 1: testing a valid imagestream",
			args: args{
				name:      "ruby",
				namespace: "testing",
			},
			want: []imagev1.ImageStream{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ruby",
						Namespace: "testing",
					},
					Status: imagev1.ImageStreamStatus{
						Tags: []imagev1.NamedTagEventList{
							{
								Tag: "latest",
								Items: []imagev1.TagEvent{
									{
										DockerImageReference: "example/ruby:latest",
										Generation:           1,
										Image:                "sha256:9579a93ee",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// TODO: Currently fails. Enable once fixed
		// {
		//         name: "case 2: empty namespace",
		//         args: args{
		//                 name:      "ruby",
		//                 namespace: "",
		//         },
		//         wantErr: true,
		// },

		// {
		// 	name: "case 3: empty name",
		// 	args: args{
		// 		name:      "",
		// 		namespace: "testing",
		// 	},
		// 	wantErr: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			client, fkclientset := FakeNew()

			fkclientset.ImageClientset.PrependReactor("list", "imagestreams", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, fakeImageStreams(tt.args.name, tt.args.namespace), nil
			})

			got, err := client.GetImageStreams(tt.args.namespace)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetImageStreams() error = %#v, wantErr %#v", err, tt.wantErr)
				return
			}

			if len(fkclientset.ImageClientset.Actions()) != 1 {
				t.Errorf("expected 1 action in GetImageStreams got: %v", fkclientset.ImageClientset.Actions())
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetImageStreams() = %#v, want %#v", got, tt.want)
			}

		})
	}
}

func TestStartBuild(t *testing.T) {
	tests := []struct {
		name    string
		bcName  string
		wantErr bool
	}{
		{
			name:    "Case 1: Testing valid name",
			bcName:  "ruby",
			wantErr: false,
		},

		// TODO: Currently fails. Enable once fixed.
		// {
		// 	name:    "Case 2: Testing empty name",
		// 	bcName:  "",
		// 	wantErr: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fkclient, fkclientset := FakeNew()

			fkclientset.BuildClientset.PrependReactor("create", "buildconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				build := buildv1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name: tt.bcName,
					},
				}

				return true, &build, nil
			})

			_, err := fkclient.StartBuild(tt.bcName)
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.StartBuild(string) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {

				if len(fkclientset.BuildClientset.Actions()) != 1 {
					t.Errorf("expected 1 action in StartBuild got: %v", fkclientset.BuildClientset.Actions())
				}

				startedBuild := fkclientset.BuildClientset.Actions()[0].(ktesting.CreateAction).GetObject().(*buildv1.BuildRequest)

				if startedBuild.Name != tt.bcName {
					t.Errorf("buildconfig name is not matching to expected name, expected: %s, got %s", tt.bcName, startedBuild.Name)
				}
			}
		})
	}

}

func TestWaitForBuildToFinish(t *testing.T) {

	tests := []struct {
		name      string
		buildName string
		status    buildv1.BuildPhase
		wantErr   bool
	}{
		{
			name:      "phase: complete",
			buildName: "ruby",
			status:    buildv1.BuildPhaseComplete,
			wantErr:   false,
		},

		{
			name:      "phase: failed",
			buildName: "ruby",
			status:    buildv1.BuildPhaseFailed,
			wantErr:   true,
		},

		{
			name:      "phase: cancelled",
			buildName: "ruby",
			status:    buildv1.BuildPhaseCancelled,
			wantErr:   true,
		},

		{
			name:      "phase: error",
			buildName: "ruby",
			status:    buildv1.BuildPhaseError,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fkclient, fkclientset := FakeNew()
			fkWatch := watch.NewFake()

			go func() {
				fkWatch.Modify(fakeBuildStatus(tt.status, tt.buildName))
			}()

			fkclientset.BuildClientset.PrependWatchReactor("builds", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			err := fkclient.WaitForBuildToFinish(tt.buildName)
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.WaitForBuildToFinish(string) unexpected error %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(fkclientset.BuildClientset.Actions()) != 1 {
				t.Errorf("expected 1 action in WaitForBuildToFinish got: %v", fkclientset.BuildClientset.Actions())
			}

			if err == nil {
				expectedFields := fields.OneTermEqualSelector("metadata.name", tt.buildName)
				gotFields := fkclientset.BuildClientset.Actions()[0].(ktesting.WatchAction).GetWatchRestrictions().Fields

				if !reflect.DeepEqual(expectedFields, gotFields) {
					t.Errorf("Fields not matching: expected: %s, got %s", expectedFields, gotFields)
				}
			}
		})
	}

}

func TestWaitAndGetPod(t *testing.T) {

	tests := []struct {
		name    string
		podName string
		status  corev1.PodPhase
		wantErr bool
	}{
		{
			name:    "phase: running",
			podName: "ruby",
			status:  corev1.PodRunning,
			wantErr: false,
		},

		{
			name:    "phase: failed",
			podName: "ruby",
			status:  corev1.PodFailed,
			wantErr: true,
		},

		{
			name: "phase:	unknown",
			podName: "ruby",
			status:  corev1.PodUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fkclient, fkclientset := FakeNew()
			fkWatch := watch.NewFake()

			// Change the status
			go func() {
				fkWatch.Modify(fakePodStatus(tt.status, tt.podName))
			}()

			fkclientset.Kubernetes.PrependWatchReactor("pods", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			podSelector := fmt.Sprintf("deploymentconfig=%s", tt.podName)
			pod, err := fkclient.WaitAndGetPod(podSelector)

			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.WaitAndGetPod(string) unexpected error %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(fkclientset.Kubernetes.Actions()) != 1 {
				t.Errorf("expected 1 action in WaitAndGetPod got: %v", fkclientset.Kubernetes.Actions())
			}

			if err == nil {
				if pod.Name != tt.podName {
					t.Errorf("pod name is not matching to expected name, expected: %s, got %s", tt.podName, pod.Name)
				}
			}

		})
	}
}

func TestCreateNewProject(t *testing.T) {
	tests := []struct {
		name     string
		projName string
		wantErr  bool
	}{
		{
			name:     "Case 1: valid project name",
			projName: "testing",
			wantErr:  false,
		},

		// TODO: Currently fails. Enable once fixed.
		// {
		// 	name:     "Case 2: empty project name",
		// 	projName: "",
		// 	wantErr:  true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()

			fkclientset.ProjClientset.PrependReactor("create", "projectrequests", func(action ktesting.Action) (bool, runtime.Object, error) {
				proj := projectv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: tt.projName,
					},
				}
				return true, &proj, nil
			})

			err := fkclient.CreateNewProject(tt.projName)
			if !tt.wantErr == (err != nil) {
				t.Errorf("client.CreateNewProject(string) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if len(fkclientset.ProjClientset.Actions()) != 1 {
				t.Errorf("expected 1 action in CreateNewProject got: %v", fkclientset.ProjClientset.Actions())
			}

			if err == nil {
				createdProj := fkclientset.ProjClientset.Actions()[0].(ktesting.CreateAction).GetObject().(*projectv1.ProjectRequest)

				if createdProj.Name != tt.projName {
					t.Errorf("project name does not match the expected name, expected: %s, got: %s", tt.projName, createdProj.Name)
				}
			}

		})
	}
}

func Test_getContainerPortsFromStrings(t *testing.T) {
	tests := []struct {
		name           string
		ports          []string
		containerPorts []corev1.ContainerPort
		wantErr        bool
	}{
		{
			name:  "with normal port values and normal protocol values in lowercase",
			ports: []string{"8080/tcp", "9090/udp"},
			containerPorts: []corev1.ContainerPort{
				{
					Name:          "8080-tcp",
					ContainerPort: 8080,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "9090-udp",
					ContainerPort: 9090,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantErr: false,
		},
		{
			name:  "with normal port values and normal protocol values in mixed case",
			ports: []string{"8080/TcP", "9090/uDp"},
			containerPorts: []corev1.ContainerPort{
				{
					Name:          "8080-tcp",
					ContainerPort: 8080,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "9090-udp",
					ContainerPort: 9090,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantErr: false,
		},
		{
			name:  "with normal port values and with one protocol value not mentioned",
			ports: []string{"8080", "9090/Udp"},
			containerPorts: []corev1.ContainerPort{
				{
					Name:          "8080-tcp",
					ContainerPort: 8080,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "9090-udp",
					ContainerPort: 9090,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantErr: false,
		},
		{
			name:    "with normal port values and with one invalid protocol value",
			ports:   []string{"8080/blah", "9090/Udp"},
			wantErr: true,
		},
		{
			name:    "with invalid port values and normal protocol",
			ports:   []string{"ads/Tcp", "9090/Udp"},
			wantErr: true,
		},
		{
			name:    "with invalid port values and one missing protocol value",
			ports:   []string{"ads", "9090/Udp"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports, err := getContainerPortsFromStrings(tt.ports)
			if err == nil && !tt.wantErr {
				if !reflect.DeepEqual(tt.containerPorts, ports) {
					t.Errorf("the ports are not matching, expected %#v, got %#v", tt.containerPorts, ports)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestCreateService(t *testing.T) {
	tests := []struct {
		name             string
		commonObjectMeta metav1.ObjectMeta
		containerPorts   []corev1.ContainerPort
		wantErr          bool
	}{
		{
			name: "Test case: with valid commonObjectName and containerPorts",
			commonObjectMeta: metav1.ObjectMeta{
				Name: "nodejs",
				Labels: map[string]string{
					"app":                              "apptmp",
					"app.kubernetes.io/component-name": "ruby",
					"app.kubernetes.io/component-type": "ruby",
					"app.kubernetes.io/name":           "apptmp",
				},
				Annotations: map[string]string{
					"app.kubernetes.io/url":                   "https://github.com/openshift/ruby",
					"app.kubernetes.io/component-source-type": "git",
				},
			},
			containerPorts: []corev1.ContainerPort{
				{
					Name:          "8080-tcp",
					ContainerPort: 8080,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "9100-udp",
					ContainerPort: 9100,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()

			_, err := fkclient.CreateService(tt.commonObjectMeta, tt.containerPorts)

			if err == nil && !tt.wantErr {
				if len(fkclientset.Kubernetes.Actions()) != 1 {
					t.Errorf("expected 1 Kubernetes.Actions() in CreateService, got: %v", fkclientset.ImageClientset.Actions())
				}
				createdSvc := fkclientset.Kubernetes.Actions()[0].(ktesting.CreateAction).GetObject().(*corev1.Service)
				if !reflect.DeepEqual(tt.commonObjectMeta, createdSvc.ObjectMeta) {
					t.Errorf("ObjectMeta does not match the expected name, expected: %v, got: %v", tt.commonObjectMeta, createdSvc.ObjectMeta)
				}
				if !reflect.DeepEqual(tt.commonObjectMeta.Name, createdSvc.Spec.Selector["deploymentconfig"]) {
					t.Errorf("selector value does not match the expected name, expected: %s, got: %s", tt.commonObjectMeta.Name, createdSvc.Spec.Selector["deploymentconfig"])
				}
				for _, port := range tt.containerPorts {
					found := false
					for _, servicePort := range createdSvc.Spec.Ports {
						if servicePort.Port == port.ContainerPort {
							found = true
							if servicePort.Protocol != port.Protocol {
								t.Errorf("service protocol does not match the expected name, expected: %s, got: %s", port.Protocol, servicePort.Protocol)
							}
							if servicePort.Name != port.Name {
								t.Errorf("service name does not match the expected name, expected: %s, got: %s", port.Name, servicePort.Name)
							}
							if servicePort.TargetPort != intstr.FromInt(int(port.ContainerPort)) {
								t.Errorf("target port does not match the expected name, expected: %v, got: %v", intstr.FromInt(int(port.ContainerPort)), servicePort.TargetPort)
							}
						}
					}
					if found == false {
						t.Errorf("expected service port %s not found in the created Service", tt.name)
						break
					}
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestGetDeploymentConfigsFromSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		label    map[string]string
		wantErr  bool
	}{
		{
			name:     "true case",
			selector: "app.kubernetes.io/name=app",
			label: map[string]string{
				"app.kubernetes.io/name": "app",
			},
			wantErr: false,
		},
		{
			name:     "true case",
			selector: "app.kubernetes.io/name=app1",
			label: map[string]string{
				"app.kubernetes.io/name": "app",
			},
			wantErr: false,
		},
	}

	listOfDC := appsv1.DeploymentConfigList{
		Items: []appsv1.DeploymentConfig{

			{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "app",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			fakeClientSet.AppsClientset.PrependReactor("list", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				if !reflect.DeepEqual(action.(ktesting.ListAction).GetListRestrictions().Labels.String(), tt.selector) {
					return true, nil, fmt.Errorf("labels not matching with expected values, expected:%s, got:%s", tt.selector, action.(ktesting.ListAction).GetListRestrictions())
				}
				return true, &listOfDC, nil
			})
			dc, err := fakeClient.GetDeploymentConfigsFromSelector(tt.selector)

			if len(fakeClientSet.AppsClientset.Actions()) != 1 {
				t.Errorf("expected 1 AppsClientset.Actions() in GetDeploymentConfigsFromSelector, got: %v", fakeClientSet.AppsClientset.Actions())
			}

			if tt.wantErr == false && err != nil {
				t.Errorf("test failed, %#v", dc[0].Labels)
			}

			for _, dc1 := range dc {
				if !reflect.DeepEqual(dc1.Labels, tt.label) {
					t.Errorf("labels are not matching with expected labels, expected: %s, got %s", tt.label, dc1.Labels)
				}
			}

		})
	}
}

func TestCreateServiceInstance(t *testing.T) {
	type args struct {
		serviceName string
		serviceType string
		labels      map[string]string
		plan        string
		parameters  map[string]string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Create service instance",
			args: args{
				serviceName: "jenkins",
				serviceType: "jenkins",
				labels: map[string]string{
					"name":      "mongodb",
					"namespace": "blog",
				},
				plan:       "dev",
				parameters: map[string]string{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()

			err := fkclient.CreateServiceInstance(tt.args.serviceName, tt.args.serviceType, tt.args.plan, tt.args.parameters, tt.args.labels)
			// Checks for error in positive cases
			if tt.wantErr == false && (err != nil) {
				t.Errorf(" client.CreateServiceInstance(serviceName,serviceType, labels) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			// Check for validating actions performed
			// creating a service instance also means creating a serviceBinding
			// which is why we expect 2 actions
			if len(fkclientset.ServiceCatalogClientSet.Actions()) != 2 && !tt.wantErr {
				t.Errorf("expected 1 action in CreateServiceInstace got: %v", fkclientset.ServiceCatalogClientSet.Actions())
			}

			createdServiceInstance := fkclientset.ServiceCatalogClientSet.Actions()[0].(ktesting.CreateAction).GetObject().(*scv1beta1.ServiceInstance)
			if !reflect.DeepEqual(createdServiceInstance.Labels, tt.args.labels) {
				t.Errorf("labels in created serviceInstance is not matching expected labels, expected: %v, got: %v", tt.args.labels, createdServiceInstance.Labels)
			}
			if createdServiceInstance.Name != tt.args.serviceName {
				t.Errorf("labels in created serviceInstance is not matching expected labels, expected: %v, got: %v", tt.args.serviceName, createdServiceInstance.Name)
			}
			if !reflect.DeepEqual(createdServiceInstance.Spec.ClusterServiceClassExternalName, tt.args.serviceType) {
				t.Errorf("labels in created serviceInstance is not matching expected labels, expected: %v, got: %v", tt.args.serviceType, createdServiceInstance.Spec.ClusterServiceClassExternalName)
			}
		})
	}
}

func TestGetClusterServiceClass(t *testing.T) {
	classExternalMetaData := make(map[string]interface{})
	classExternalMetaData["longDescription"] = "example long description"
	classExternalMetaData["dependencies"] = []string{"docker.io/centos/7", "docker.io/centos/8"}

	classExternalMetaDataRaw, err := json.Marshal(classExternalMetaData)
	if err != nil {
		fmt.Printf("error occured %v during marshalling", err)
		return
	}

	type args struct {
		serviceName string
	}
	tests := []struct {
		name                    string
		args                    args
		returnedServicesClasses *scv1beta1.ClusterServiceClassList
		wantedServiceClass      *scv1beta1.ClusterServiceClass
		wantErr                 bool
	}{
		{
			name: "test case 1: with one valid service class returned",
			args: args{
				serviceName: "class name",
			},
			returnedServicesClasses: &scv1beta1.ClusterServiceClassList{
				Items: []scv1beta1.ClusterServiceClass{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "1dda1477cace09730bd8ed7a6505607e"},
						Spec: scv1beta1.ClusterServiceClassSpec{
							CommonServiceClassSpec: scv1beta1.CommonServiceClassSpec{
								ExternalName:     "class name",
								Bindable:         false,
								Description:      "example description",
								Tags:             []string{"php", "java"},
								ExternalMetadata: &runtime.RawExtension{Raw: classExternalMetaDataRaw},
							},
							ClusterServiceBrokerName: "broker name",
						},
					},
				},
			},
			wantedServiceClass: &scv1beta1.ClusterServiceClass{
				ObjectMeta: metav1.ObjectMeta{Name: "1dda1477cace09730bd8ed7a6505607e"},
				Spec: scv1beta1.ClusterServiceClassSpec{
					CommonServiceClassSpec: scv1beta1.CommonServiceClassSpec{
						ExternalName:     "class name",
						Bindable:         false,
						Description:      "example description",
						Tags:             []string{"php", "java"},
						ExternalMetadata: &runtime.RawExtension{Raw: classExternalMetaDataRaw},
					},
					ClusterServiceBrokerName: "broker name",
				},
			},
			wantErr: false,
		},
		{
			name: "test case 2: with two service classes returned",
			args: args{
				serviceName: "class name",
			},
			returnedServicesClasses: &scv1beta1.ClusterServiceClassList{
				Items: []scv1beta1.ClusterServiceClass{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "1dda1477cace09730bd8ed7a6505607e"},
						Spec: scv1beta1.ClusterServiceClassSpec{
							CommonServiceClassSpec: scv1beta1.CommonServiceClassSpec{
								ExternalName:     "class name",
								Bindable:         false,
								Description:      "example description",
								Tags:             []string{"php", "java"},
								ExternalMetadata: &runtime.RawExtension{Raw: classExternalMetaDataRaw},
							},
							ClusterServiceBrokerName: "broker name",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "1dda1477cace09730bd8ed7a6505607e"},
						Spec: scv1beta1.ClusterServiceClassSpec{
							CommonServiceClassSpec: scv1beta1.CommonServiceClassSpec{
								ExternalName:     "class name",
								Bindable:         false,
								Description:      "example description",
								Tags:             []string{"java"},
								ExternalMetadata: &runtime.RawExtension{Raw: classExternalMetaDataRaw},
							},
							ClusterServiceBrokerName: "broker name 1",
						},
					},
				},
			},
			wantedServiceClass: &scv1beta1.ClusterServiceClass{},
			wantErr:            true,
		},
		{
			name: "test case 3: with no service classes returned",
			args: args{
				serviceName: "class name",
			},
			returnedServicesClasses: &scv1beta1.ClusterServiceClassList{
				Items: []scv1beta1.ClusterServiceClass{},
			},
			wantedServiceClass: &scv1beta1.ClusterServiceClass{},
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := FakeNew()

		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "clusterserviceclasses", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			if action.(ktesting.ListAction).GetListRestrictions().Fields.String() != fmt.Sprintf("spec.externalName=%v", tt.args.serviceName) {
				t.Errorf("got a different service name got: %v , expected: %v", action.(ktesting.ListAction).GetListRestrictions().Fields.String(), fmt.Sprintf("spec.externalName=%v", tt.args.serviceName))
			}
			return true, tt.returnedServicesClasses, nil
		})

		gotServiceClass, err := client.GetClusterServiceClass(tt.args.serviceName)
		if err == nil && !tt.wantErr {
			if len(fakeClientSet.ServiceCatalogClientSet.Actions()) != 1 {
				t.Errorf("expected 1 action in GetServiceClassAndPlans got: %v", fakeClientSet.ServiceCatalogClientSet.Actions())
			}

			if !reflect.DeepEqual(gotServiceClass.Spec, tt.wantedServiceClass.Spec) {
				t.Errorf("different service class spec value expected got: %v , expected: %v", gotServiceClass.Spec, tt.wantedServiceClass.Spec)
			}
			if !reflect.DeepEqual(gotServiceClass.Name, tt.wantedServiceClass.Name) {
				t.Errorf("different service class name value expected got: %v , expected: %v", gotServiceClass.Name, tt.wantedServiceClass.Name)
			}
		} else if err == nil && tt.wantErr {
			t.Error("test failed, expected: false, got true")
		} else if err != nil && !tt.wantErr {
			t.Errorf("test failed, expected: no error, got error: %s", err.Error())
		}
	}

}

func TestGetClusterPlansFromServiceName(t *testing.T) {
	planExternalMetaDataRaw, err := fakePlanExternalMetaDataRaw()
	if err != nil {
		fmt.Printf("error occured %v during marshalling", err)
		return
	}

	planServiceInstanceCreateParameterSchemasRaw, err := fakePlanServiceInstanceCreateParameterSchemasRaw()
	if err != nil {
		fmt.Printf("error occured %v during marshalling", err)
		return
	}

	type args struct {
		serviceClassName string
	}
	tests := []struct {
		name    string
		args    args
		want    []scv1beta1.ClusterServicePlan
		wantErr bool
	}{
		{
			name:    "test case 1 : plans found for the service class",
			args:    args{serviceClassName: "1dda1477cace09730bd8ed7a6505607e"},
			wantErr: false,
			want: []scv1beta1.ClusterServicePlan{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "67042296c7c95e84142f21f58da2ebfe",
					},
					Spec: scv1beta1.ClusterServicePlanSpec{
						ClusterServiceClassRef: scv1beta1.ClusterObjectReference{
							Name: "1dda1477cace09730bd8ed7a6505607e",
						},
						CommonServicePlanSpec: scv1beta1.CommonServicePlanSpec{
							ExternalName:                         "dev",
							Description:                          "this is a example description 1",
							ExternalMetadata:                     &runtime.RawExtension{Raw: planExternalMetaDataRaw[0]},
							ServiceInstanceCreateParameterSchema: &runtime.RawExtension{Raw: planServiceInstanceCreateParameterSchemasRaw[0]},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "7f88be6129622f72554c20af879a8ce0",
					},
					Spec: scv1beta1.ClusterServicePlanSpec{
						ClusterServiceClassRef: scv1beta1.ClusterObjectReference{
							Name: "1dda1477cace09730bd8ed7a6505607e",
						},
						CommonServicePlanSpec: scv1beta1.CommonServicePlanSpec{
							ExternalName:                         "prod",
							Description:                          "this is a example description 2",
							ExternalMetadata:                     &runtime.RawExtension{Raw: planExternalMetaDataRaw[1]},
							ServiceInstanceCreateParameterSchema: &runtime.RawExtension{Raw: planServiceInstanceCreateParameterSchemasRaw[1]},
						},
					},
				},
			},
		},
		{
			name:    "test case 2 : no plans found for the service class",
			args:    args{serviceClassName: "1dda1477cace09730bd8"},
			wantErr: false,
			want:    []scv1beta1.ClusterServicePlan{},
		},
	}

	planList := scv1beta1.ClusterServicePlanList{
		Items: []scv1beta1.ClusterServicePlan{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "67042296c7c95e84142f21f58da2ebfe",
				},
				Spec: scv1beta1.ClusterServicePlanSpec{
					ClusterServiceClassRef: scv1beta1.ClusterObjectReference{
						Name: "1dda1477cace09730bd8ed7a6505607e",
					},
					CommonServicePlanSpec: scv1beta1.CommonServicePlanSpec{
						ExternalName:                         "dev",
						Description:                          "this is a example description 1",
						ExternalMetadata:                     &runtime.RawExtension{Raw: planExternalMetaDataRaw[0]},
						ServiceInstanceCreateParameterSchema: &runtime.RawExtension{Raw: planServiceInstanceCreateParameterSchemasRaw[0]},
					},
				},
			},

			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "7f88be6129622f72554c20af879a8ce0",
				},
				Spec: scv1beta1.ClusterServicePlanSpec{
					ClusterServiceClassRef: scv1beta1.ClusterObjectReference{
						Name: "1dda1477cace09730bd8ed7a6505607e",
					},
					CommonServicePlanSpec: scv1beta1.CommonServicePlanSpec{
						ExternalName:                         "prod",
						Description:                          "this is a example description 2",
						ExternalMetadata:                     &runtime.RawExtension{Raw: planExternalMetaDataRaw[1]},
						ServiceInstanceCreateParameterSchema: &runtime.RawExtension{Raw: planServiceInstanceCreateParameterSchemasRaw[1]},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, fakeClientSet := FakeNew()

			fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "clusterserviceplans", func(action ktesting.Action) (bool, runtime.Object, error) {
				var pList []scv1beta1.ClusterServicePlan
				for _, plan := range planList.Items {
					if plan.Spec.ClusterServiceClassRef.Name == strings.Split(action.(ktesting.ListAction).GetListRestrictions().Fields.String(), "=")[1] {
						pList = append(pList, plan)
					}
				}

				return true, &scv1beta1.ClusterServicePlanList{Items: pList}, nil
			})

			gotPlans, err := client.GetClusterPlansFromServiceName(tt.args.serviceClassName)
			if err == nil && !tt.wantErr {
				if len(fakeClientSet.ServiceCatalogClientSet.Actions()) != 1 {
					t.Errorf("expected 2 actions in GetServiceClassAndPlans got: %v", fakeClientSet.ServiceCatalogClientSet.Actions())
				}

				for _, wantedServicePlan := range tt.want {
					found := false
					for _, gotServicePlan := range gotPlans {
						if reflect.DeepEqual(wantedServicePlan.Spec.ExternalName, gotServicePlan.Spec.ExternalName) {
							found = true
						} else {
							continue
						}

						if !reflect.DeepEqual(wantedServicePlan.Name, gotServicePlan.Name) {
							t.Errorf("different plan name expected got: %v , expected: %v", wantedServicePlan.Name, gotServicePlan.Name)
						}

						if !reflect.DeepEqual(wantedServicePlan.Spec, gotServicePlan.Spec) {
							t.Errorf("different plan spec value expected got: %v , expected: %v", wantedServicePlan.Spec, gotServicePlan.Spec)
						}
					}

					if !found {
						t.Errorf("service plan %v not found", wantedServicePlan.Spec.ExternalName)
					}
				}
			} else if err == nil && tt.wantErr {
				t.Error("test failed, expected: false, got true")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, expected: no error, got error: %s", err.Error())
			}
		})
	}
}

func TestGetServiceInstanceList(t *testing.T) {

	type args struct {
		Project  string
		Selector string
	}

	tests := []struct {
		name        string
		args        args
		serviceList scv1beta1.ServiceInstanceList
		output      []scv1beta1.ServiceInstance
		wantErr     bool
	}{
		{
			name: "test case 1",
			args: args{
				Project:  "myproject",
				Selector: "app.kubernetes.io/component-name=mysql-persistent,app.kubernetes.io/name=app",
			},
			serviceList: scv1beta1.ServiceInstanceList{
				Items: []scv1beta1.ServiceInstance{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "mysql-persistent",
							Finalizers: []string{"kubernetes-incubator/service-catalog"},
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "mysql-persistent",
								componentlabels.ComponentTypeLabel: "mysql-persistent",
							},
							Namespace: "myproject",
						},
						Spec: scv1beta1.ServiceInstanceSpec{
							PlanReference: scv1beta1.PlanReference{
								ClusterServiceClassExternalName: "mysql-persistent",
								ClusterServicePlanExternalName:  "default",
							},
						},
						Status: scv1beta1.ServiceInstanceStatus{
							Conditions: []scv1beta1.ServiceInstanceCondition{
								{
									Reason: "ProvisionedSuccessfully",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "jenkins-persistent",
							Finalizers: []string{"kubernetes-incubator/service-catalog"},
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "jenkins-persistent",
								componentlabels.ComponentTypeLabel: "jenkins-persistent",
							},
							Namespace: "myproject",
						},
						Spec: scv1beta1.ServiceInstanceSpec{
							PlanReference: scv1beta1.PlanReference{
								ClusterServiceClassExternalName: "jenkins-persistent",
								ClusterServicePlanExternalName:  "default",
							},
						},
						Status: scv1beta1.ServiceInstanceStatus{
							Conditions: []scv1beta1.ServiceInstanceCondition{
								{
									Reason: "ProvisionedSuccessfully",
								},
							},
						},
					},
				},
			},
			output: []scv1beta1.ServiceInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "mysql-persistent",
						Finalizers: []string{"kubernetes-incubator/service-catalog"},
						Labels: map[string]string{
							applabels.ApplicationLabel:         "app",
							componentlabels.ComponentLabel:     "mysql-persistent",
							componentlabels.ComponentTypeLabel: "mysql-persistent",
						},
						Namespace: "myproject",
					},
					Spec: scv1beta1.ServiceInstanceSpec{
						PlanReference: scv1beta1.PlanReference{
							ClusterServiceClassExternalName: "mysql-persistent",
							ClusterServicePlanExternalName:  "default",
						},
					},
					Status: scv1beta1.ServiceInstanceStatus{
						Conditions: []scv1beta1.ServiceInstanceCondition{
							{
								Reason: "ProvisionedSuccessfully",
							},
						},
					},
				},
			},

			wantErr: false,
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := FakeNew()

		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "serviceinstances", func(action ktesting.Action) (bool, runtime.Object, error) {
			if !reflect.DeepEqual(action.(ktesting.ListAction).GetListRestrictions().Labels.String(), tt.args.Selector) {
				return true, nil, fmt.Errorf("labels not matching with expected values, expected:%s, got:%s", tt.args.Selector, action.(ktesting.ListAction).GetListRestrictions())
			}
			return true, &tt.serviceList, nil
		})

		svcInstanceList, err := client.GetServiceInstanceList(tt.args.Selector)

		if !reflect.DeepEqual(tt.output, svcInstanceList) {
			t.Errorf("expected output: %#v,got: %#v", tt.serviceList, svcInstanceList)
		}

		if err == nil && !tt.wantErr {
			if (len(fakeClientSet.ServiceCatalogClientSet.Actions()) != 1) && (tt.wantErr != true) {
				t.Errorf("expected 1 action in ListServicecatalog got: %v", fakeClientSet.ServiceCatalogClientSet.Actions())
			}
		} else if err == nil && tt.wantErr {
			t.Error("test failed, expected: false, got true")
		} else if err != nil && !tt.wantErr {
			t.Errorf("test failed, expected: no error, got error: %s", err.Error())
		}
	}
}

func TestPatchCurrentDC(t *testing.T) {
	type args struct {
		name     string
		dcBefore appsv1.DeploymentConfig
		dcPatch  appsv1.DeploymentConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		actions int
	}{
		{
			name: "Case 1: Test patching",
			args: args{
				name:     "foo",
				dcBefore: *fakeDeploymentConfig("foo", "foo", []corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
				dcPatch: generateGitDeploymentConfig(metav1.ObjectMeta{Name: "foo", Annotations: map[string]string{"app.kubernetes.io/component-source-type": "git"}}, "bar",
					[]corev1.ContainerPort{{Name: "foo", HostPort: 80, ContainerPort: 80}},
					[]corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
			},
			wantErr: false,
			actions: 3,
		},
		{
			name: "Case 2: Test patching with the wrong name",
			args: args{
				name:     "foo",
				dcBefore: *fakeDeploymentConfig("foo", "foo", []corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
				dcPatch: generateGitDeploymentConfig(metav1.ObjectMeta{Name: "foo2"}, "bar",
					[]corev1.ContainerPort{corev1.ContainerPort{Name: "foo", HostPort: 80, ContainerPort: 80}},
					[]corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
			},
			wantErr: true,
			actions: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			// Fake "watch"
			fkWatch := watch.NewFake()
			go func() {
				fkWatch.Modify(&tt.args.dcPatch)
			}()
			fakeClientSet.AppsClientset.PrependWatchReactor("deploymentconfigs", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			// Fake getting DC
			fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, &tt.args.dcBefore, nil
			})

			// Fake the "update"
			fakeClientSet.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				dc := action.(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)
				if dc.Name != tt.args.dcPatch.Name {
					return true, nil, fmt.Errorf("got different dc")
				}
				return true, nil, nil
			})

			// Run function PatchCurrentDC
			err := fakeClient.PatchCurrentDC(tt.args.name, tt.args.dcPatch)

			// Error checking PatchCurrentDC
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.PatchCurrentDC() unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && !tt.wantErr {
				// Check to see how many actions are being ran
				if (len(fakeClientSet.AppsClientset.Actions()) != tt.actions) && !tt.wantErr {
					t.Errorf("expected %v action(s) in PatchCurrentDC got %v: %v", tt.actions, len(fakeClientSet.AppsClientset.Actions()), fakeClientSet.AppsClientset.Actions())
				}
			} else if err == nil && tt.wantErr {
				t.Error("test failed, expected: false, got true")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, expected: no error, got error: %s", err.Error())
			}

		})
	}
}

func TestUpdateDCToGit(t *testing.T) {
	type args struct {
		name     string
		newImage string
		dc       appsv1.DeploymentConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		actions int
	}{
		{
			name: "Case 1: Check the function works",
			args: args{
				name:     "foo",
				newImage: "bar",
				dc:       *fakeDeploymentConfig("foo", "foo", []corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
			},
			wantErr: false,
			actions: 4,
		},
		{
			name: "Case 2: Fail if the variable passed in is blank",
			args: args{
				name:     "foo",
				newImage: "",
				dc:       *fakeDeploymentConfig("foo", "foo", []corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
			},
			wantErr: true,
			actions: 4,
		},
		{
			name: "Case 3: Fail if image retrieved doesn't match the one we want to patch",
			args: args{
				name:     "foo",
				newImage: "",
				dc:       *fakeDeploymentConfig("foo2", "foo", []corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
			},
			wantErr: true,
			actions: 3,
		},
		{
			name: "Case 4: Check we can patch with a tag",
			args: args{
				name:     "foo",
				newImage: "bar:latest",
				dc:       *fakeDeploymentConfig("foo", "foo", []corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
			},
			wantErr: false,
			actions: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			// Fake "watch"
			fkWatch := watch.NewFake()
			go func() {
				fkWatch.Modify(&tt.args.dc)
			}()
			fakeClientSet.AppsClientset.PrependWatchReactor("deploymentconfigs", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			// Fake getting DC
			fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, &tt.args.dc, nil
			})

			// Fake the "update"
			fakeClientSet.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				dc := action.(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)

				// Check name
				if dc.Name != tt.args.dc.Name {
					return true, nil, fmt.Errorf("got different dc")
				}

				// Check that the new patch actually has the new "image"
				if !tt.wantErr == (dc.Spec.Template.Spec.Containers[0].Image != tt.args.newImage) {
					return true, nil, fmt.Errorf("got %s image, suppose to get %s", dc.Spec.Template.Spec.Containers[0].Image, tt.args.newImage)
				}

				return true, nil, nil
			})

			// Run function UpdateDCToGit
			err := fakeClient.UpdateDCToGit(metav1.ObjectMeta{Name: tt.args.name}, tt.args.newImage)

			// Error checking UpdateDCToGit
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.UpdateDCToGit() unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && !tt.wantErr {
				// Check to see how many actions are being ran
				if (len(fakeClientSet.AppsClientset.Actions()) != tt.actions) && !tt.wantErr {
					t.Errorf("expected %v action(s) in UpdateDCToGit got %v: %v", tt.actions, len(fakeClientSet.AppsClientset.Actions()), fakeClientSet.AppsClientset.Actions())
				}
			} else if err == nil && tt.wantErr {
				t.Error("test failed, expected: false, got true")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, expected: no error, got error: %s", err.Error())
			}

		})
	}
}

func TestUniqueAppendOrOverwriteEnvVars(t *testing.T) {
	tests := []struct {
		name            string
		existingEnvVars []corev1.EnvVar
		envVars         []corev1.EnvVar
		want            []corev1.EnvVar
	}{
		{
			name: "Case: Overlapping env vars appends",
			existingEnvVars: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "key1",
					Value: "value1",
				},
				corev1.EnvVar{
					Name:  "key2",
					Value: "value2",
				},
			},
			envVars: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "key1",
					Value: "value3",
				},
				corev1.EnvVar{
					Name:  "key2",
					Value: "value4",
				},
			},
			want: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "key1",
					Value: "value3",
				},
				corev1.EnvVar{
					Name:  "key2",
					Value: "value4",
				},
			},
		},
		{
			name: "New env vars append",
			existingEnvVars: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "key1",
					Value: "value1",
				},
				corev1.EnvVar{
					Name:  "key2",
					Value: "value2",
				},
			},
			envVars: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "key3",
					Value: "value3",
				},
				corev1.EnvVar{
					Name:  "key4",
					Value: "value4",
				},
			},
			want: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "key1",
					Value: "value1",
				},
				corev1.EnvVar{
					Name:  "key2",
					Value: "value2",
				},
				corev1.EnvVar{
					Name:  "key3",
					Value: "value3",
				},
				corev1.EnvVar{
					Name:  "key4",
					Value: "value4",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnvVars := uniqueAppendOrOverwriteEnvVars(tt.existingEnvVars, tt.envVars...)
			if len(tt.want) != len(gotEnvVars) {
				t.Errorf("Tc: %s, expected %+v, got %+v", tt.name, tt.want, gotEnvVars)
			}
			matchFound := false
			for _, wantEnv := range tt.want {
				for _, gotEnv := range gotEnvVars {
					if reflect.DeepEqual(wantEnv, gotEnv) {
						matchFound = true
					}
				}
				if !matchFound {
					t.Errorf("Tc: %s, expected %+v, got %+v", tt.name, tt.want, gotEnvVars)
				}
			}
		})
	}
}

func TestGetS2IScriptsPathFromBuilderImg(t *testing.T) {
	tests := []struct {
		name             string
		imageStreamImage *imagev1.ImageStreamImage
		wantProtocol     string
		wantPath         string
		wantErr          bool
	}{
		{
			name: "Case 1: Valid test case with image protocol access",
			imageStreamImage: fakeImageStreamImage(
				"nodejs",
				[]string{"8080/tcp"},
				`{"kind":"DockerImage","apiVersion":"1.0","Id":"sha256:93de1230c12b512ebbaf28b159f450a44c632eda06bdc0754236f403f5876234","Created":"2018-10-19T15:43:13Z","ContainerConfig":{"Hostname":"8911994b686d","User":"1001","ExposedPorts":{"8080/tcp":{}},"Env":["PATH=/opt/app-root/src/node_modules/.bin/:/opt/app-root/src/.npm-global/bin/:/opt/app-root/src/bin:/opt/app-root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","SUMMARY=Platform for building and running Node.js 10.12.0 applications","DESCRIPTION=Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","STI_SCRIPTS_URL=image:///usr/libexec/s2i","STI_SCRIPTS_PATH=/usr/libexec/s2i","APP_ROOT=/opt/app-root","HOME=/opt/app-root/src","BASH_ENV=/opt/app-root/etc/scl_enable","ENV=/opt/app-root/etc/scl_enable","PROMPT_COMMAND=. /opt/app-root/etc/scl_enable","NODEJS_SCL=rh-nodejs8","NPM_RUN=start","NODE_VERSION=10.12.0","NPM_VERSION=6.4.1","NODE_LTS=false","NPM_CONFIG_LOGLEVEL=info","NPM_CONFIG_PREFIX=/opt/app-root/src/.npm-global","NPM_CONFIG_TARBALL=/usr/share/node/node-v10.12.0-headers.tar.gz","DEBUG_PORT=5858"],"Cmd":["/bin/sh","-c","#(nop) ","CMD [\"/bin/sh\" \"-c\" \"${STI_SCRIPTS_PATH}/usage\"]"],"Image":"sha256:d353b3f467c2d2ff59a1e09bb91cff1c493aedd6c8041ebb273346e892f8ee85","WorkingDir":"/opt/app-root/src","Entrypoint":["container-entrypoint"],"Labels":{"com.redhat.component":"s2i-base-container","com.redhat.deployments-dir":"/opt/app-root/src","com.redhat.dev-mode":"DEV_MODE:false","com.rehdat.dev-mode.port":"DEBUG_PORT:5858","description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.display-name":"Node.js 10.12.0","io.openshift.builder-version":"\"190ef14\"","io.openshift.expose-services":"8080:http","io.openshift.s2i.scripts-url":"image:///usr/libexec/s2i","io.openshift.tags":"builder,nodejs,nodejs-10.12.0","io.s2i.scripts-url":"image:///usr/libexec/s2i","maintainer":"Lance Ball \u003clball@redhat.com\u003e","name":"bucharestgold/centos7-s2i-nodejs","org.label-schema.build-date":"20180804","org.label-schema.license":"GPLv2","org.label-schema.name":"CentOS Base Image","org.label-schema.schema-version":"1.0","org.label-schema.vendor":"CentOS","release":"1","summary":"Platform for building and running Node.js 10.12.0 applications","version":"10.12.0"}},"DockerVersion":"18.06.0-ce","Config":{"Hostname":"8911994b686d","User":"1001","ExposedPorts":{"8080/tcp":{}},"Env":["PATH=/opt/app-root/src/node_modules/.bin/:/opt/app-root/src/.npm-global/bin/:/opt/app-root/src/bin:/opt/app-root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","SUMMARY=Platform for building and running Node.js 10.12.0 applications","DESCRIPTION=Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","STI_SCRIPTS_URL=image:///usr/libexec/s2i","STI_SCRIPTS_PATH=/usr/libexec/s2i","APP_ROOT=/opt/app-root","HOME=/opt/app-root/src","BASH_ENV=/opt/app-root/etc/scl_enable","ENV=/opt/app-root/etc/scl_enable","PROMPT_COMMAND=. /opt/app-root/etc/scl_enable","NODEJS_SCL=rh-nodejs8","NPM_RUN=start","NODE_VERSION=10.12.0","NPM_VERSION=6.4.1","NODE_LTS=false","NPM_CONFIG_LOGLEVEL=info","NPM_CONFIG_PREFIX=/opt/app-root/src/.npm-global","NPM_CONFIG_TARBALL=/usr/share/node/node-v10.12.0-headers.tar.gz","DEBUG_PORT=5858"],"Cmd":["/bin/sh","-c","${STI_SCRIPTS_PATH}/usage"],"Image":"57a00ab03a3f8c3af19e91c284e0c499b16d5115c925aa845b5d14eace949c34","WorkingDir":"/opt/app-root/src","Entrypoint":["container-entrypoint"],"Labels":{"com.redhat.component":"s2i-base-container","com.redhat.deployments-dir":"/opt/app-root/src","com.redhat.dev-mode":"DEV_MODE:false","com.rehdat.dev-mode.port":"DEBUG_PORT:5858","description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.display-name":"Node.js 10.12.0","io.openshift.builder-version":"\"190ef14\"","io.openshift.expose-services":"8080:http","io.openshift.s2i.scripts-url":"image:///usr/libexec/s2i","io.openshift.tags":"builder,nodejs,nodejs-10.12.0","io.s2i.scripts-url":"image:///usr/libexec/s2i","maintainer":"Lance Ball \u003clball@redhat.com\u003e","name":"bucharestgold/centos7-s2i-nodejs","org.label-schema.build-date":"20180804","org.label-schema.license":"GPLv2","org.label-schema.name":"CentOS Base Image","org.label-schema.schema-version":"1.0","org.label-schema.vendor":"CentOS","release":"1","summary":"Platform for building and running Node.js 10.12.0 applications","version":"10.12.0"}},"Architecture":"amd64","Size":221580439}`,
			),
			wantProtocol: "image://",
			wantPath:     "/usr/libexec/s2i",
			wantErr:      false,
		},
		{
			name: "Case 2: Valid test case with file protocol access",
			imageStreamImage: fakeImageStreamImage(
				"nodejs",
				[]string{"8080/tcp"},
				`{"kind":"DockerImage","apiVersion":"1.0","Id":"sha256:93de1230c12b512ebbaf28b159f450a44c632eda06bdc0754236f403f5876234","Created":"2018-10-19T15:43:13Z","ContainerConfig":{"Hostname":"8911994b686d","User":"1001","ExposedPorts":{"8080/tcp":{}},"Env":["PATH=/opt/app-root/src/node_modules/.bin/:/opt/app-root/src/.npm-global/bin/:/opt/app-root/src/bin:/opt/app-root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","SUMMARY=Platform for building and running Node.js 10.12.0 applications","DESCRIPTION=Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","STI_SCRIPTS_URL=file:///usr/libexec/s2i","STI_SCRIPTS_PATH=/usr/libexec/s2i","APP_ROOT=/opt/app-root","HOME=/opt/app-root/src","BASH_ENV=/opt/app-root/etc/scl_enable","ENV=/opt/app-root/etc/scl_enable","PROMPT_COMMAND=. /opt/app-root/etc/scl_enable","NODEJS_SCL=rh-nodejs8","NPM_RUN=start","NODE_VERSION=10.12.0","NPM_VERSION=6.4.1","NODE_LTS=false","NPM_CONFIG_LOGLEVEL=info","NPM_CONFIG_PREFIX=/opt/app-root/src/.npm-global","NPM_CONFIG_TARBALL=/usr/share/node/node-v10.12.0-headers.tar.gz","DEBUG_PORT=5858"],"Cmd":["/bin/sh","-c","#(nop) ","CMD [\"/bin/sh\" \"-c\" \"${STI_SCRIPTS_PATH}/usage\"]"],"Image":"sha256:d353b3f467c2d2ff59a1e09bb91cff1c493aedd6c8041ebb273346e892f8ee85","WorkingDir":"/opt/app-root/src","Entrypoint":["container-entrypoint"],"Labels":{"com.redhat.component":"s2i-base-container","com.redhat.deployments-dir":"/opt/app-root/src","com.redhat.dev-mode":"DEV_MODE:false","com.rehdat.dev-mode.port":"DEBUG_PORT:5858","description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.display-name":"Node.js 10.12.0","io.openshift.builder-version":"\"190ef14\"","io.openshift.expose-services":"8080:http","io.openshift.s2i.scripts-url":"file:///usr/libexec/s2i","io.openshift.tags":"builder,nodejs,nodejs-10.12.0","io.s2i.scripts-url":"image:///usr/libexec/s2i","maintainer":"Lance Ball \u003clball@redhat.com\u003e","name":"bucharestgold/centos7-s2i-nodejs","org.label-schema.build-date":"20180804","org.label-schema.license":"GPLv2","org.label-schema.name":"CentOS Base Image","org.label-schema.schema-version":"1.0","org.label-schema.vendor":"CentOS","release":"1","summary":"Platform for building and running Node.js 10.12.0 applications","version":"10.12.0"}},"DockerVersion":"18.06.0-ce","Config":{"Hostname":"8911994b686d","User":"1001","ExposedPorts":{"8080/tcp":{}},"Env":["PATH=/opt/app-root/src/node_modules/.bin/:/opt/app-root/src/.npm-global/bin/:/opt/app-root/src/bin:/opt/app-root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","SUMMARY=Platform for building and running Node.js 10.12.0 applications","DESCRIPTION=Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","STI_SCRIPTS_URL=image:///usr/libexec/s2i","STI_SCRIPTS_PATH=/usr/libexec/s2i","APP_ROOT=/opt/app-root","HOME=/opt/app-root/src","BASH_ENV=/opt/app-root/etc/scl_enable","ENV=/opt/app-root/etc/scl_enable","PROMPT_COMMAND=. /opt/app-root/etc/scl_enable","NODEJS_SCL=rh-nodejs8","NPM_RUN=start","NODE_VERSION=10.12.0","NPM_VERSION=6.4.1","NODE_LTS=false","NPM_CONFIG_LOGLEVEL=info","NPM_CONFIG_PREFIX=/opt/app-root/src/.npm-global","NPM_CONFIG_TARBALL=/usr/share/node/node-v10.12.0-headers.tar.gz","DEBUG_PORT=5858"],"Cmd":["/bin/sh","-c","${STI_SCRIPTS_PATH}/usage"],"Image":"57a00ab03a3f8c3af19e91c284e0c499b16d5115c925aa845b5d14eace949c34","WorkingDir":"/opt/app-root/src","Entrypoint":["container-entrypoint"],"Labels":{"com.redhat.component":"s2i-base-container","com.redhat.deployments-dir":"/opt/app-root/src","com.redhat.dev-mode":"DEV_MODE:false","com.rehdat.dev-mode.port":"DEBUG_PORT:5858","description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.display-name":"Node.js 10.12.0","io.openshift.builder-version":"\"190ef14\"","io.openshift.expose-services":"8080:http","io.openshift.s2i.scripts-url":"image:///usr/libexec/s2i","io.openshift.tags":"builder,nodejs,nodejs-10.12.0","io.s2i.scripts-url":"image:///usr/libexec/s2i","maintainer":"Lance Ball \u003clball@redhat.com\u003e","name":"bucharestgold/centos7-s2i-nodejs","org.label-schema.build-date":"20180804","org.label-schema.license":"GPLv2","org.label-schema.name":"CentOS Base Image","org.label-schema.schema-version":"1.0","org.label-schema.vendor":"CentOS","release":"1","summary":"Platform for building and running Node.js 10.12.0 applications","version":"10.12.0"}},"Architecture":"amd64","Size":221580439}`,
			),
			wantProtocol: "file://",
			wantPath:     "/usr/libexec/s2i",
			wantErr:      false,
		},
		{
			name: "Case 3: Valid test case with http(s) protocol access",
			imageStreamImage: fakeImageStreamImage(
				"nodejs",
				[]string{"8080/tcp"},
				`{"kind":"DockerImage","apiVersion":"1.0","Id":"sha256:93de1230c12b512ebbaf28b159f450a44c632eda06bdc0754236f403f5876234","Created":"2018-10-19T15:43:13Z","ContainerConfig":{"Hostname":"8911994b686d","User":"1001","ExposedPorts":{"8080/tcp":{}},"Env":["PATH=/opt/app-root/src/node_modules/.bin/:/opt/app-root/src/.npm-global/bin/:/opt/app-root/src/bin:/opt/app-root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","SUMMARY=Platform for building and running Node.js 10.12.0 applications","DESCRIPTION=Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","STI_SCRIPTS_URL=http(s):///usr/libexec/s2i","STI_SCRIPTS_PATH=/usr/libexec/s2i","APP_ROOT=/opt/app-root","HOME=/opt/app-root/src","BASH_ENV=/opt/app-root/etc/scl_enable","ENV=/opt/app-root/etc/scl_enable","PROMPT_COMMAND=. /opt/app-root/etc/scl_enable","NODEJS_SCL=rh-nodejs8","NPM_RUN=start","NODE_VERSION=10.12.0","NPM_VERSION=6.4.1","NODE_LTS=false","NPM_CONFIG_LOGLEVEL=info","NPM_CONFIG_PREFIX=/opt/app-root/src/.npm-global","NPM_CONFIG_TARBALL=/usr/share/node/node-v10.12.0-headers.tar.gz","DEBUG_PORT=5858"],"Cmd":["/bin/sh","-c","#(nop) ","CMD [\"/bin/sh\" \"-c\" \"${STI_SCRIPTS_PATH}/usage\"]"],"Image":"sha256:d353b3f467c2d2ff59a1e09bb91cff1c493aedd6c8041ebb273346e892f8ee85","WorkingDir":"/opt/app-root/src","Entrypoint":["container-entrypoint"],"Labels":{"com.redhat.component":"s2i-base-container","com.redhat.deployments-dir":"/opt/app-root/src","com.redhat.dev-mode":"DEV_MODE:false","com.rehdat.dev-mode.port":"DEBUG_PORT:5858","description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.display-name":"Node.js 10.12.0","io.openshift.builder-version":"\"190ef14\"","io.openshift.expose-services":"8080:http","io.openshift.s2i.scripts-url":"http(s):///usr/libexec/s2i","io.openshift.tags":"builder,nodejs,nodejs-10.12.0","io.s2i.scripts-url":"image:///usr/libexec/s2i","maintainer":"Lance Ball \u003clball@redhat.com\u003e","name":"bucharestgold/centos7-s2i-nodejs","org.label-schema.build-date":"20180804","org.label-schema.license":"GPLv2","org.label-schema.name":"CentOS Base Image","org.label-schema.schema-version":"1.0","org.label-schema.vendor":"CentOS","release":"1","summary":"Platform for building and running Node.js 10.12.0 applications","version":"10.12.0"}},"DockerVersion":"18.06.0-ce","Config":{"Hostname":"8911994b686d","User":"1001","ExposedPorts":{"8080/tcp":{}},"Env":["PATH=/opt/app-root/src/node_modules/.bin/:/opt/app-root/src/.npm-global/bin/:/opt/app-root/src/bin:/opt/app-root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","SUMMARY=Platform for building and running Node.js 10.12.0 applications","DESCRIPTION=Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","STI_SCRIPTS_URL=image:///usr/libexec/s2i","STI_SCRIPTS_PATH=/usr/libexec/s2i","APP_ROOT=/opt/app-root","HOME=/opt/app-root/src","BASH_ENV=/opt/app-root/etc/scl_enable","ENV=/opt/app-root/etc/scl_enable","PROMPT_COMMAND=. /opt/app-root/etc/scl_enable","NODEJS_SCL=rh-nodejs8","NPM_RUN=start","NODE_VERSION=10.12.0","NPM_VERSION=6.4.1","NODE_LTS=false","NPM_CONFIG_LOGLEVEL=info","NPM_CONFIG_PREFIX=/opt/app-root/src/.npm-global","NPM_CONFIG_TARBALL=/usr/share/node/node-v10.12.0-headers.tar.gz","DEBUG_PORT=5858"],"Cmd":["/bin/sh","-c","${STI_SCRIPTS_PATH}/usage"],"Image":"57a00ab03a3f8c3af19e91c284e0c499b16d5115c925aa845b5d14eace949c34","WorkingDir":"/opt/app-root/src","Entrypoint":["container-entrypoint"],"Labels":{"com.redhat.component":"s2i-base-container","com.redhat.deployments-dir":"/opt/app-root/src","com.redhat.dev-mode":"DEV_MODE:false","com.rehdat.dev-mode.port":"DEBUG_PORT:5858","description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.display-name":"Node.js 10.12.0","io.openshift.builder-version":"\"190ef14\"","io.openshift.expose-services":"8080:http","io.openshift.s2i.scripts-url":"image:///usr/libexec/s2i","io.openshift.tags":"builder,nodejs,nodejs-10.12.0","io.s2i.scripts-url":"image:///usr/libexec/s2i","maintainer":"Lance Ball \u003clball@redhat.com\u003e","name":"bucharestgold/centos7-s2i-nodejs","org.label-schema.build-date":"20180804","org.label-schema.license":"GPLv2","org.label-schema.name":"CentOS Base Image","org.label-schema.schema-version":"1.0","org.label-schema.vendor":"CentOS","release":"1","summary":"Platform for building and running Node.js 10.12.0 applications","version":"10.12.0"}},"Architecture":"amd64","Size":221580439}`,
			),
			wantProtocol: "http(s)://",
			wantPath:     "http(s):///usr/libexec/s2i",
			wantErr:      false,
		},
		{
			name: "Case 4: Inalid test case with invalid protocol access",
			imageStreamImage: fakeImageStreamImage(
				"nodejs",
				[]string{"8080/tcp"},
				`{"kind":"DockerImage","apiVersion":"1.0","Id":"sha256:93de1230c12b512ebbaf28b159f450a44c632eda06bdc0754236f403f5876234","Created":"2018-10-19T15:43:13Z","ContainerConfig":{"Hostname":"8911994b686d","User":"1001","ExposedPorts":{"8080/tcp":{}},"Env":["PATH=/opt/app-root/src/node_modules/.bin/:/opt/app-root/src/.npm-global/bin/:/opt/app-root/src/bin:/opt/app-root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","SUMMARY=Platform for building and running Node.js 10.12.0 applications","DESCRIPTION=Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","STI_SCRIPTS_URL=something:///usr/libexec/s2i","STI_SCRIPTS_PATH=/usr/libexec/s2i","APP_ROOT=/opt/app-root","HOME=/opt/app-root/src","BASH_ENV=/opt/app-root/etc/scl_enable","ENV=/opt/app-root/etc/scl_enable","PROMPT_COMMAND=. /opt/app-root/etc/scl_enable","NODEJS_SCL=rh-nodejs8","NPM_RUN=start","NODE_VERSION=10.12.0","NPM_VERSION=6.4.1","NODE_LTS=false","NPM_CONFIG_LOGLEVEL=info","NPM_CONFIG_PREFIX=/opt/app-root/src/.npm-global","NPM_CONFIG_TARBALL=/usr/share/node/node-v10.12.0-headers.tar.gz","DEBUG_PORT=5858"],"Cmd":["/bin/sh","-c","#(nop) ","CMD [\"/bin/sh\" \"-c\" \"${STI_SCRIPTS_PATH}/usage\"]"],"Image":"sha256:d353b3f467c2d2ff59a1e09bb91cff1c493aedd6c8041ebb273346e892f8ee85","WorkingDir":"/opt/app-root/src","Entrypoint":["container-entrypoint"],"Labels":{"com.redhat.component":"s2i-base-container","com.redhat.deployments-dir":"/opt/app-root/src","com.redhat.dev-mode":"DEV_MODE:false","com.rehdat.dev-mode.port":"DEBUG_PORT:5858","description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.display-name":"Node.js 10.12.0","io.openshift.builder-version":"\"190ef14\"","io.openshift.expose-services":"8080:http","io.openshift.s2i.scripts-url":"something:///usr/libexec/s2i","io.openshift.tags":"builder,nodejs,nodejs-10.12.0","io.s2i.scripts-url":"image:///usr/libexec/s2i","maintainer":"Lance Ball \u003clball@redhat.com\u003e","name":"bucharestgold/centos7-s2i-nodejs","org.label-schema.build-date":"20180804","org.label-schema.license":"GPLv2","org.label-schema.name":"CentOS Base Image","org.label-schema.schema-version":"1.0","org.label-schema.vendor":"CentOS","release":"1","summary":"Platform for building and running Node.js 10.12.0 applications","version":"10.12.0"}},"DockerVersion":"18.06.0-ce","Config":{"Hostname":"8911994b686d","User":"1001","ExposedPorts":{"8080/tcp":{}},"Env":["PATH=/opt/app-root/src/node_modules/.bin/:/opt/app-root/src/.npm-global/bin/:/opt/app-root/src/bin:/opt/app-root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin","SUMMARY=Platform for building and running Node.js 10.12.0 applications","DESCRIPTION=Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","STI_SCRIPTS_URL=image:///usr/libexec/s2i","STI_SCRIPTS_PATH=/usr/libexec/s2i","APP_ROOT=/opt/app-root","HOME=/opt/app-root/src","BASH_ENV=/opt/app-root/etc/scl_enable","ENV=/opt/app-root/etc/scl_enable","PROMPT_COMMAND=. /opt/app-root/etc/scl_enable","NODEJS_SCL=rh-nodejs8","NPM_RUN=start","NODE_VERSION=10.12.0","NPM_VERSION=6.4.1","NODE_LTS=false","NPM_CONFIG_LOGLEVEL=info","NPM_CONFIG_PREFIX=/opt/app-root/src/.npm-global","NPM_CONFIG_TARBALL=/usr/share/node/node-v10.12.0-headers.tar.gz","DEBUG_PORT=5858"],"Cmd":["/bin/sh","-c","${STI_SCRIPTS_PATH}/usage"],"Image":"57a00ab03a3f8c3af19e91c284e0c499b16d5115c925aa845b5d14eace949c34","WorkingDir":"/opt/app-root/src","Entrypoint":["container-entrypoint"],"Labels":{"com.redhat.component":"s2i-base-container","com.redhat.deployments-dir":"/opt/app-root/src","com.redhat.dev-mode":"DEV_MODE:false","com.rehdat.dev-mode.port":"DEBUG_PORT:5858","description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.description":"Node.js  available as docker container is a base platform for building and running various Node.js  applications and frameworks. Node.js is a platform built on Chrome's JavaScript runtime for easily building fast, scalable network applications. Node.js uses an event-driven, non-blocking I/O model that makes it lightweight and efficient, perfect for data-intensive real-time applications that run across distributed devices.","io.k8s.display-name":"Node.js 10.12.0","io.openshift.builder-version":"\"190ef14\"","io.openshift.expose-services":"8080:http","io.openshift.s2i.scripts-url":"image:///usr/libexec/s2i","io.openshift.tags":"builder,nodejs,nodejs-10.12.0","io.s2i.scripts-url":"image:///usr/libexec/s2i","maintainer":"Lance Ball \u003clball@redhat.com\u003e","name":"bucharestgold/centos7-s2i-nodejs","org.label-schema.build-date":"20180804","org.label-schema.license":"GPLv2","org.label-schema.name":"CentOS Base Image","org.label-schema.schema-version":"1.0","org.label-schema.vendor":"CentOS","release":"1","summary":"Platform for building and running Node.js 10.12.0 applications","version":"10.12.0"}},"Architecture":"amd64","Size":221580439}`,
			),
			wantProtocol: "",
			wantPath:     "",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProtocol, gotPath, err := GetS2IScriptsPathFromBuilderImg(tt.imageStreamImage)
			if tt.wantProtocol != gotProtocol {
				t.Errorf("Tc: %s, Expected protocol: %s, Got protocol: %s", tt.name, tt.wantProtocol, gotProtocol)
			}
			if tt.wantPath != gotPath {
				t.Errorf("Tc: %s, Expected path: %s, Got path: %s", tt.name, tt.wantPath, gotPath)
			}
			if !tt.wantErr == (err != nil) {
				t.Errorf(" GetS2IScriptsPathFromBuilderImg() unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateDCToSupervisor(t *testing.T) {
	type args struct {
		name           string
		imageName      string
		expectedImage  string
		imageNamespace string
		dc             appsv1.DeploymentConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		actions int
	}{
		{
			name: "Case 1: Check the function works",
			args: args{
				name:           "foo",
				imageName:      "nodejs",
				expectedImage:  "nodejs",
				imageNamespace: "openshift",
				dc:             *fakeDeploymentConfig("foo", "foo", []corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
			},
			wantErr: false,
			actions: 4,
		},
		{
			name: "Case 2: Fail if unable to find container",
			args: args{
				name:           "testfoo",
				imageName:      "foo",
				expectedImage:  "foobar",
				imageNamespace: "testing",
				dc:             *fakeDeploymentConfig("foo", "foo", []corev1.EnvVar{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}}),
			},
			wantErr: true,
			actions: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			// Fake "watch"
			fkWatch := watch.NewFake()
			go func() {
				fkWatch.Modify(&tt.args.dc)
			}()
			fakeClientSet.AppsClientset.PrependWatchReactor("deploymentconfigs", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			// Fake getting DC
			fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, &tt.args.dc, nil
			})

			// Fake the "update"
			fakeClientSet.AppsClientset.PrependReactor("update", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				dc := action.(ktesting.UpdateAction).GetObject().(*appsv1.DeploymentConfig)

				// Check name
				if dc.Name != tt.args.dc.Name {
					return true, nil, fmt.Errorf("got different dc")
				}

				// Check that the new patch actually has parts of supervisord in it when it's used..

				// Check that addBootstrapVolumeCopyInitContainer is the 1st initContainer and it exists
				if !tt.wantErr == (dc.Spec.Template.Spec.InitContainers[0].Name != "copy-files-to-volume") {
					return true, nil, fmt.Errorf("client.UpdateDCSupervisor() does not contain the copy-files-to-volume container within Spec.Template.Spec.InitContainers, found: %v", dc.Spec.Template.Spec.InitContainers[0].Name)
				}

				// Check that addBootstrapVolumeCopyInitContainer is the 2nd initContainer and it exists
				if !tt.wantErr == (dc.Spec.Template.Spec.InitContainers[1].Name != "copy-supervisord") {
					return true, nil, fmt.Errorf("client.UpdateDCSupervisor() does not contain the copy-supervisord container within Spec.Template.Spec.InitContainers, found: %v", dc.Spec.Template.Spec.InitContainers[1].Name)
				}

				return true, nil, nil
			})

			// Fake getting image stream
			fakeClientSet.ImageClientset.PrependReactor("get", "imagestreams", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, fakeImageStream(tt.args.expectedImage, tt.args.imageNamespace, []string{"latest"}), nil
			})

			fakeClientSet.ImageClientset.PrependReactor("get", "imagestreamimages", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, fakeImageStreamImages(tt.args.imageName), nil
			})

			// Run function UpdateDCToSupervisor
			err := fakeClient.UpdateDCToSupervisor(metav1.ObjectMeta{Name: tt.args.name}, tt.args.imageName)

			// Error checking UpdateDCToSupervisor
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.UpdateDCToSupervisor() unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			// Check to see how many actions are being ran
			if err == nil && !tt.wantErr {
				if (len(fakeClientSet.AppsClientset.Actions()) != tt.actions) && !tt.wantErr {
					t.Errorf("expected %v action(s) in UpdateDCToSupervisor got %v: %v", tt.actions, len(fakeClientSet.AppsClientset.Actions()), fakeClientSet.AppsClientset.Actions())
				}
			} else if err == nil && tt.wantErr {
				t.Error("test failed, expected: false, got true")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, expected: no error, got error: %s", err.Error())
			}

		})
	}
}

func TestIsVolumeAnEmptyDir(t *testing.T) {
	type args struct {
		VolumeName string
		dc         appsv1.DeploymentConfig
	}
	tests := []struct {
		name         string
		args         args
		wantEmptyDir bool
	}{
		{
			name: "Case 1 - Check that it is an emptyDir",
			args: args{
				VolumeName: supervisordVolumeName,
				dc:         *fakeDeploymentConfig("foo", "bar", nil),
			},
			wantEmptyDir: true,
		},
		{
			name: "Case 2 - Check a non-existent volume",
			args: args{
				VolumeName: "foobar",
				dc:         *fakeDeploymentConfig("foo", "bar", nil),
			},
			wantEmptyDir: false,
		},
		{
			name: "Case 3 - Check a volume that exists but is not emptyDir",
			args: args{
				VolumeName: "foo-s2idata",
				dc:         *fakeDeploymentConfig("foo", "bar", nil),
			},
			wantEmptyDir: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, _ := FakeNew()

			// Run function IsVolumeAnEmptyDir
			isVolumeEmpty := fakeClient.IsVolumeAnEmptyDir(tt.args.VolumeName, &tt.args.dc)

			// Error checking IsVolumeAnEmptyDir
			if tt.wantEmptyDir != isVolumeEmpty {
				t.Errorf(" client.IsVolumeAnEmptyDir() unexpected %v, wantEmptyDir %v", isVolumeEmpty, tt.wantEmptyDir)
			}

		})
	}
}

func Test_getInputEnvVarsFromStrings(t *testing.T) {
	tests := []struct {
		name          string
		envVars       []string
		wantedEnvVars []corev1.EnvVar
		wantErr       bool
	}{
		{
			name:    "Test case 1: with valid two key value pairs",
			envVars: []string{"key=value", "key1=value1"},
			wantedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key1",
					Value: "value1",
				},
			},
			wantErr: false,
		},
		{
			name:    "Test case 2: one env var with missing value",
			envVars: []string{"key=value", "key1="},
			wantedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key1",
					Value: "",
				},
			},
			wantErr: false,
		},
		{
			name:    "Test case 3: one env var with no value and no =",
			envVars: []string{"key=value", "key1"},
			wantErr: true,
		},
		{
			name:    "Test case 4: one env var with multiple values",
			envVars: []string{"key=value", "key1=value1=value2"},
			wantedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key1",
					Value: "value1=value2",
				},
			},
			wantErr: false,
		},
		{
			name:    "Test case 5: two env var with same key",
			envVars: []string{"key=value", "key=value1"},
			wantErr: true,
		},
		{
			name:    "Test case 6: one env var with base64 encoded value",
			envVars: []string{"key=value", "key1=SSd2ZSBnb3QgYSBsb3ZlbHkgYnVuY2ggb2YgY29jb251dHMhCg=="},
			wantedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key1",
					Value: "SSd2ZSBnb3QgYSBsb3ZlbHkgYnVuY2ggb2YgY29jb251dHMhCg==",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars, err := getInputEnvVarsFromStrings(tt.envVars)

			if err == nil && !tt.wantErr {
				if !reflect.DeepEqual(tt.wantedEnvVars, envVars) {
					t.Errorf("corev1.Env values are not matching with expected values, expected: %v, got %v", tt.wantedEnvVars, envVars)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestGetEnvVarsFromDC(t *testing.T) {
	tests := []struct {
		name            string
		dcName          string
		projectName     string
		returnedDC      appsv1.DeploymentConfig
		returnedEnvVars []corev1.EnvVar
		wantErr         bool
	}{
		{
			name:        "case 1: with valid existing dc and one valid env var pair",
			dcName:      "nodejs-app",
			projectName: "project",
			returnedDC: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs-app",
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "key",
											Value: "value",
										},
									},
								},
							},
						},
					},
				},
			},
			returnedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
			},
			wantErr: false,
		},
		{
			name:        "case 2: with valid existing dc and two valid env var pairs",
			dcName:      "nodejs-app",
			projectName: "project",
			returnedDC: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs-app",
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "key",
											Value: "value",
										},
										{
											Name:  "key-1",
											Value: "value-1",
										},
									},
								},
							},
						},
					},
				},
			},
			returnedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key-1",
					Value: "value-1",
				},
			},
			wantErr: false,
		},
		{
			name:        "case 3: with non valid existing dc",
			dcName:      "nodejs-app",
			projectName: "project",
			returnedDC: appsv1.DeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildfly-app",
				},
				Spec: appsv1.DeploymentConfigSpec{
					Template: &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{},
								},
							},
						},
					},
				},
			},
			returnedEnvVars: []corev1.EnvVar{},
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				dcName := action.(ktesting.GetAction).GetName()
				if dcName != tt.dcName {
					return true, nil, fmt.Errorf("get dc called with different name, expected: %s, got %s", tt.dcName, dcName)
				}
				return true, &tt.returnedDC, nil
			})

			envVars, err := fakeClient.GetEnvVarsFromDC(tt.dcName)

			if err == nil && !tt.wantErr {
				// Check for validating actions performed
				if len(fakeClientSet.AppsClientset.Actions()) != 1 {
					t.Errorf("expected 1 action in GetBuildConfigFromName got: %v", fakeClientSet.AppsClientset.Actions())
				}

				if !reflect.DeepEqual(tt.returnedEnvVars, envVars) {
					t.Errorf("env vars are not matching with expected values, expected: %s, got %s", tt.returnedEnvVars, envVars)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func Test_updateEnvVar(t *testing.T) {
	type args struct {
		dc           *appsv1.DeploymentConfig
		inputEnvVars []corev1.EnvVar
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test case 1: tests with single container in dc and no existing env vars",
			args: args{
				dc: fakeDeploymentConfig("foo", "foo", nil),
				inputEnvVars: []corev1.EnvVar{
					{
						Name:  "key",
						Value: "value",
					},
					{
						Name:  "key-1",
						Value: "value-1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test case 2: tests with single container in dc and existing env vars",
			args: args{
				dc: fakeDeploymentConfig("foo", "foo", []corev1.EnvVar{{Name: "key-1", Value: "key-1"}}),
				inputEnvVars: []corev1.EnvVar{
					{
						Name:  "key-2",
						Value: "value-2",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test case 3: tests with double container in dc",
			args: args{
				dc: &appsv1.DeploymentConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "wildfly-app",
					},
					Spec: appsv1.DeploymentConfigSpec{
						Template: &corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{},
									},
									{
										Env: []corev1.EnvVar{},
									},
								},
							},
						},
					},
				},
				inputEnvVars: []corev1.EnvVar{
					{
						Name:  "key",
						Value: "value",
					},
					{
						Name:  "key-1",
						Value: "value-1",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updateEnvVar(tt.args.dc, tt.args.inputEnvVars)

			if err == nil && !tt.wantErr {
				found := false
				for _, inputEnv := range tt.args.inputEnvVars {
					for _, foundEnv := range tt.args.dc.Spec.Template.Spec.Containers[0].Env {
						if reflect.DeepEqual(inputEnv, foundEnv) {
							found = true
						}
					}
				}
				if !found {
					t.Errorf("update env vars are not matching, expected dc to contain: %v", tt.args.inputEnvVars)
				}

			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func Test_findContainer(t *testing.T) {
	type args struct {
		name       string
		containers []corev1.Container
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Case 1 - Find the container",
			args: args{
				name: "foo",
				containers: []corev1.Container{
					{
						Name: "foo",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Case 2 - Error if container not found",
			args: args{
				name: "foo2",
				containers: []corev1.Container{
					{
						Name: "foo",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Case 3 - Error when passing in blank container name",
			args: args{
				name: "",
				containers: []corev1.Container{
					{
						Name: "foo",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Case 4 - Check against multiple containers (rather than one)",
			args: args{
				name: "foo",
				containers: []corev1.Container{
					{
						Name: "bar",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
					{
						Name: "foo",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Run function findContainer
			container, err := findContainer(tt.args.containers, tt.args.name)

			// Check that the container matches the name
			if err == nil && container.Name != tt.args.name {
				t.Errorf("Wrong container returned, wanted container %v, got %v", tt.args.name, container.Name)
			}

			if err == nil && tt.wantErr {
				t.Error("test failed, expected: false, got true")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, expected: no error, got error: %s", err.Error())
			}

		})
	}
}

func TestWaitAndGetDC(t *testing.T) {
	type args struct {
		name       string
		annotation string
		value      string
		dc         appsv1.DeploymentConfig
		timeout    time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		actions int
	}{
		{
			name: "Case 1 - Check that the function actually works",
			args: args{
				name:       "foo",
				annotation: "app.kubernetes.io/component-source-type",
				value:      "git",
				dc:         *fakeDeploymentConfig("foo", "bar", []corev1.EnvVar{}),
				timeout:    3 * time.Second,
			},
			wantErr: false,
			actions: 1,
		},
		{
			name: "Case 2 - Purposefully timeout / error",
			args: args{
				name:       "foo",
				annotation: "app.kubernetes.io/component-source-type",
				value:      "foobar",
				dc:         *fakeDeploymentConfig("foo", "bar", []corev1.EnvVar{}),
				timeout:    3 * time.Second,
			},
			wantErr: true,
			actions: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()
			fkWatch := watch.NewFake()
			go func() {
				fkWatch.Modify(&tt.args.dc)
			}()
			fakeClientSet.AppsClientset.PrependWatchReactor("deploymentconfigs", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})
			// Run function WaitAndGetDC
			_, err := fakeClient.WaitAndGetDC(tt.args.name, tt.args.annotation, tt.args.value, tt.args.timeout)
			// Error checking WaitAndGetDC
			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.WaitAndGetDC() unexpected error %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !tt.wantErr {
				// Check to see how many actions are being ran
				if (len(fakeClientSet.AppsClientset.Actions()) != tt.actions) && !tt.wantErr {
					t.Errorf("expected %v action(s) in WaitAndGetDC got %v: %v", tt.actions, len(fakeClientSet.AppsClientset.Actions()), fakeClientSet.AppsClientset.Actions())
				}
			} else if err == nil && tt.wantErr {
				t.Error("test failed, expected: false, got true")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, expected: no error, got error: %s", err.Error())
			}
		})
	}
}
