package component

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"testing"

	applabels "github.com/openshift/odo/pkg/application/labels"
	componentlabels "github.com/openshift/odo/pkg/component/labels"
	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/testingutil"

	appsv1 "github.com/openshift/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	. "github.com/openshift/odo/pkg/config"
)

func TestGetS2IPaths(t *testing.T) {

	tests := []struct {
		name    string
		podEnvs []corev1.EnvVar
		want    []string
	}{
		{
			name: "Case 1: odo expected s2i envs available",
			podEnvs: []corev1.EnvVar{
				{
					Name:  occlient.EnvS2IDeploymentDir,
					Value: "abc",
				},
				{
					Name:  occlient.EnvS2ISrcOrBinPath,
					Value: "def",
				},
				{
					Name:  occlient.EnvS2IWorkingDir,
					Value: "ghi",
				},
				{
					Name:  occlient.EnvS2ISrcBackupDir,
					Value: "ijk",
				},
			},
			want: []string{
				filepath.FromSlash("abc/src"),
				filepath.FromSlash("def/src"),
				filepath.FromSlash("ghi/src"),
				filepath.FromSlash("ijk/src"),
			},
		},
		{
			name: "Case 2: some of the odo expected s2i envs not available",
			podEnvs: []corev1.EnvVar{
				{
					Name:  occlient.EnvS2IDeploymentDir,
					Value: "abc",
				},
				{
					Name:  occlient.EnvS2ISrcOrBinPath,
					Value: "def",
				},
				{
					Name:  occlient.EnvS2ISrcBackupDir,
					Value: "ijk",
				},
			},
			want: []string{
				filepath.FromSlash("abc/src"),
				filepath.FromSlash("def/src"),
				filepath.FromSlash("ijk/src"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getS2IPaths(tt.podEnvs)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}
func TestGetComponentPorts(t *testing.T) {
	type args struct {
		componentName   string
		applicationName string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		output  []string
	}{
		{
			name: "Case 1: Invalid/Unexisting component name",
			args: args{
				componentName:   "r",
				applicationName: "app",
			},
			wantErr: true,
			output:  []string{},
		},
		{
			name: "Case 2: Valid params with multiple containers each with multiple ports",
			args: args{
				componentName:   "python",
				applicationName: "app",
			},
			output:  []string{"10080/TCP", "8080/TCP", "9090/UDP", "10090/UDP"},
			wantErr: false,
		},
		{
			name: "Case 3: Valid params with single container and single port",
			args: args{
				componentName:   "nodejs",
				applicationName: "app",
			},
			output:  []string{"8080/TCP"},
			wantErr: false,
		},
		{
			name: "Case 4: Valid params with single container and multiple port",
			args: args{
				componentName:   "wildfly",
				applicationName: "app",
			},
			output:  []string{"8090/TCP", "8080/TCP"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Fake the client with the appropriate arguments
			client, fakeClientSet := occlient.FakeNew()
			fakeClientSet.AppsClientset.PrependReactor("list", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, testingutil.FakeDeploymentConfigs(), nil
			})

			// The function we are testing
			output, err := GetComponentPorts(client, tt.args.componentName, tt.args.applicationName)

			// Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component List() unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			// Sort the output and expected o/p in-order to avoid issues due to order as its not important
			sort.Strings(output)
			sort.Strings(tt.output)

			// Check if the output is the same as what's expected (tags)
			// and only if output is more than 0 (something is actually returned)
			if len(output) > 0 && !(reflect.DeepEqual(output, tt.output)) {
				t.Errorf("expected tags: %s, got: %s", tt.output, output)
			}
		})
	}
}

func TestGetComponentLinkedSecretNames(t *testing.T) {
	type args struct {
		componentName   string
		applicationName string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		output  []string
	}{
		{
			name: "Case 1: Invalid/Unexisting component name",
			args: args{
				componentName:   "r",
				applicationName: "app",
			},
			wantErr: true,
			output:  []string{},
		},
		{
			name: "Case 2: Valid params nil env source",
			args: args{
				componentName:   "python",
				applicationName: "app",
			},
			output:  []string{},
			wantErr: false,
		},
		{
			name: "Case 3: Valid params multiple secrets",
			args: args{
				componentName:   "nodejs",
				applicationName: "app",
			},
			output:  []string{"s1", "s2"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Fake the client with the appropriate arguments
			client, fakeClientSet := occlient.FakeNew()

			fakeClientSet.AppsClientset.PrependReactor("list", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, testingutil.FakeDeploymentConfigs(), nil
			})

			// The function we are testing
			output, err := GetComponentLinkedSecretNames(client, tt.args.componentName, tt.args.applicationName)

			// Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component List() unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			// Sort the output and expected o/p in-order to avoid issues due to order as its not important
			sort.Strings(output)
			sort.Strings(tt.output)

			// Check if the output is the same as what's expected (tags)
			// and only if output is more than 0 (something is actually returned)
			if len(output) > 0 && !(reflect.DeepEqual(output, tt.output)) {
				t.Errorf("expected tags: %s, got: %s", tt.output, output)
			}
		})
	}
}

func TestList(t *testing.T) {
	componentConfig, err := GetComponentFromConfig(GetOneExistingConfigInfo("comp", "app", "test"))
	if err != nil {
		t.Errorf("error occured while calling GetComponentFromConfig, error: %v", err)
	}

	existingSampleLocalConfig := config.GetOneExistingConfigInfo("comp", "app", "test")

	tests := []struct {
		name                    string
		dcList                  appsv1.DeploymentConfigList
		projectExists           bool
		wantErr                 bool
		existingLocalConfigInfo *LocalConfigInfo
		output                  ComponentList
	}{
		{
			name: "Case 1: Components are returned",
			dcList: appsv1.DeploymentConfigList{
				Items: []appsv1.DeploymentConfig{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "frontend-app",
							Namespace: "test",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "frontend",
								componentlabels.ComponentTypeLabel: "nodejs",
							},
							Annotations: map[string]string{
								ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backend-app",
							Namespace: "test",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "backend",
								componentlabels.ComponentTypeLabel: "java",
							},
							Annotations: map[string]string{
								ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-otherApp",
							Namespace: "test",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "otherApp",
								componentlabels.ComponentLabel:     "test",
								componentlabels.ComponentTypeLabel: "python",
							},
							Annotations: map[string]string{
								ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr:       false,
			projectExists: true,
			output: ComponentList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "List",
					APIVersion: "odo.openshift.io/v1alpha1",
				},
				ListMeta: metav1.ListMeta{},
				Items: []Component{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Component",
							APIVersion: "odo.openshift.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "frontend",
							Namespace: "test",
						},
						Spec: ComponentSpec{
							Type: "nodejs",
							App:  "app",
						},
						Status: ComponentStatus{
							State:            "Pushed",
							LinkedServices:   []string{},
							LinkedComponents: map[string][]string{},
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Component",
							APIVersion: "odo.openshift.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backend",
							Namespace: "test",
						},
						Spec: ComponentSpec{
							Type: "java",
							App:  "app",
						},
						Status: ComponentStatus{
							State:            "Pushed",
							LinkedServices:   []string{},
							LinkedComponents: map[string][]string{},
						},
					},
				},
			},
		},
		{
			name:          "Case 2: projects doesn't exist",
			wantErr:       false,
			projectExists: false,
			output:        GetMachineReadableFormatForList([]Component{}),
		},
		{
			name:          "Case 3: no component and no config exists",
			wantErr:       false,
			projectExists: true,
			output:        GetMachineReadableFormatForList([]Component{}),
		},
		{
			name: "Case 4: Components are returned from the config plus and cluster",
			dcList: appsv1.DeploymentConfigList{
				Items: []appsv1.DeploymentConfig{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "frontend-app",
							Namespace: "test",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "frontend",
								componentlabels.ComponentTypeLabel: "nodejs",
							},
							Annotations: map[string]string{
								ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backend-app",
							Namespace: "project-testing",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "backend",
								componentlabels.ComponentTypeLabel: "java",
							},
							Annotations: map[string]string{
								ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-otherApp",
							Namespace: "test",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "otherApp",
								componentlabels.ComponentLabel:     "test",
								componentlabels.ComponentTypeLabel: "python",
							},
							Annotations: map[string]string{
								ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr:                 false,
			projectExists:           true,
			existingLocalConfigInfo: &existingSampleLocalConfig,
			output: ComponentList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "List",
					APIVersion: "odo.openshift.io/v1alpha1",
				},
				ListMeta: metav1.ListMeta{},
				Items: []Component{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Component",
							APIVersion: "odo.openshift.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "frontend",
							Namespace: "test",
						},
						Spec: ComponentSpec{
							Type: "nodejs",
							App:  "app",
						},
						Status: ComponentStatus{
							State:            "Pushed",
							LinkedServices:   []string{},
							LinkedComponents: map[string][]string{},
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Component",
							APIVersion: "odo.openshift.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backend",
							Namespace: "test",
						},
						Spec: ComponentSpec{
							Type: "java",
							App:  "app",
						},
						Status: ComponentStatus{
							State:            "Pushed",
							LinkedServices:   []string{},
							LinkedComponents: map[string][]string{},
						},
					},
					componentConfig,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, fakeClientSet := occlient.FakeNew()
			client.Namespace = "test"

			fakeClientSet.ProjClientset.PrependReactor("get", "projects", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				if !tt.projectExists {
					return true, nil, nil
				}
				return true, &testingutil.FakeOnlyOneExistingProjects().Items[0], nil
			})

			//fake the dcs
			fakeClientSet.AppsClientset.PrependReactor("list", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, &tt.dcList, nil
			})

			for i := range tt.dcList.Items {
				fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, &tt.dcList.Items[i], nil
				})
			}

			results, err := List(client, "app", tt.existingLocalConfigInfo)

			if (err != nil) != tt.wantErr {
				t.Errorf("expected err: %v, but err is %v", tt.wantErr, err)
			}

			if !reflect.DeepEqual(tt.output, results) {
				t.Errorf("expected output: %#v,got: %#v", tt.output, results)
			}
		})
	}
}

func TestGetDefaultComponentName(t *testing.T) {
	tests := []struct {
		testName           string
		componentType      string
		componentPath      string
		componentPathType  config.SrcType
		existingComponents ComponentList
		wantErr            bool
		wantRE             string
		needPrefix         bool
	}{
		{
			testName:           "Case: App prefix not configured",
			componentType:      "nodejs",
			componentPathType:  config.GIT,
			componentPath:      "https://github.com/openshift/nodejs.git",
			existingComponents: ComponentList{},
			wantErr:            false,
			wantRE:             "nodejs-*",
			needPrefix:         false,
		},
		{
			testName:           "Case: App prefix configured",
			componentType:      "nodejs",
			componentPathType:  config.LOCAL,
			componentPath:      "./testing",
			existingComponents: ComponentList{},
			wantErr:            false,
			wantRE:             "nodejs-testing-*",
			needPrefix:         true,
		},
		{
			testName:           "Case: App prefix configured",
			componentType:      "wildfly",
			componentPathType:  config.BINARY,
			componentPath:      "./testing.war",
			existingComponents: ComponentList{},
			wantErr:            false,
			wantRE:             "wildfly-testing-*",
			needPrefix:         true,
		},
	}

	for _, tt := range tests {
		t.Log("Running test: ", tt.testName)
		t.Run(tt.testName, func(t *testing.T) {
			odoConfigFile, kubeConfigFile, err := testingutil.SetUp(
				testingutil.ConfigDetails{
					FileName:      "odo-test-config",
					Config:        testingutil.FakeOdoConfig("odo-test-config", false, ""),
					ConfigPathEnv: "GLOBALODOCONFIG",
				}, testingutil.ConfigDetails{
					FileName:      "kube-test-config",
					Config:        testingutil.FakeKubeClientConfig(),
					ConfigPathEnv: "KUBECONFIG",
				},
			)
			defer testingutil.CleanupEnv([]*os.File{odoConfigFile, kubeConfigFile}, t)
			if err != nil {
				t.Errorf("failed to setup test env. Error %v", err)
			}

			name, err := GetDefaultComponentName(tt.componentPath, tt.componentPathType, tt.componentType, tt.existingComponents)
			if err != nil {
				t.Errorf("failed to setup mock environment. Error: %v", err)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("expected err: %v, but err is %v", tt.wantErr, err)
			}

			r, _ := regexp.Compile(tt.wantRE)
			match := r.MatchString(name)
			if !match {
				t.Errorf("randomly generated application name %s does not match regexp %s", name, tt.wantRE)
			}
		})
	}
}

func TestGetComponentDir(t *testing.T) {
	type args struct {
		path      string
		paramType config.SrcType
	}
	tests := []struct {
		testName string
		args     args
		want     string
		wantErr  bool
	}{
		{
			testName: "Case: Git URL",
			args: args{
				paramType: config.GIT,
				path:      "https://github.com/openshift/nodejs-ex.git",
			},
			want:    "nodejs-ex",
			wantErr: false,
		},
		{
			testName: "Case: Source Path",
			args: args{
				paramType: config.LOCAL,
				path:      "./testing",
			},
			wantErr: false,
			want:    "testing",
		},
		{
			testName: "Case: Binary path",
			args: args{
				paramType: config.BINARY,
				path:      "./testing.war",
			},
			wantErr: false,
			want:    "testing",
		},
		{
			testName: "Case: No clue of any component",
			args: args{
				paramType: config.NONE,
				path:      "",
			},
			wantErr: false,
			want:    "component",
		},
	}

	for _, tt := range tests {
		t.Log("Running test: ", tt.testName)
		t.Run(tt.testName, func(t *testing.T) {
			name, err := GetComponentDir(tt.args.path, tt.args.paramType)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected err: %v, but err is %v", tt.wantErr, err)
			}

			if name != tt.want {
				t.Errorf("received name %s which does not match %s", name, tt.want)
			}
		})
	}
}

func Test_getMachineReadableFormat(t *testing.T) {
	type args struct {
		componentName string
		componentType string
	}
	tests := []struct {
		name string
		args args
		want Component
	}{
		{
			name: "Test: Machine Readable Output",
			args: args{componentName: "frontend", componentType: "nodejs"},
			want: Component{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Component",
					APIVersion: "odo.openshift.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "frontend",
				},
				Spec: ComponentSpec{
					Type: "nodejs",
				},
				Status: ComponentStatus{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMachineReadableFormat(tt.args.componentName, tt.args.componentType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMachineReadableFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMachineReadableFormatForList(t *testing.T) {
	type args struct {
		components []Component
	}
	tests := []struct {
		name string
		args args
		want ComponentList
	}{
		{
			name: "Test: machine readable output for list",
			args: args{
				components: []Component{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Component",
							APIVersion: "odo.openshift.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "frontend",
						},
						Spec: ComponentSpec{
							Type: "nodejs",
						},
						Status: ComponentStatus{},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Component",
							APIVersion: "odo.openshift.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "backend",
						},
						Spec: ComponentSpec{
							Type: "wildfly",
						},
						Status: ComponentStatus{},
					},
				},
			},
			want: ComponentList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "List",
					APIVersion: "odo.openshift.io/v1alpha1",
				},
				ListMeta: metav1.ListMeta{},
				Items: []Component{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Component",
							APIVersion: "odo.openshift.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "frontend",
						},
						Spec: ComponentSpec{
							Type: "nodejs",
						},
						Status: ComponentStatus{},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Component",
							APIVersion: "odo.openshift.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "backend",
						},
						Spec: ComponentSpec{
							Type: "wildfly",
						},
						Status: ComponentStatus{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMachineReadableFormatForList(tt.args.components); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMachineReadableFormatForList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetComponentFromConfig(t *testing.T) {
	tempConfigFile, err := ioutil.TempFile("", "odoconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer tempConfigFile.Close()
	os.Setenv("LOCALODOCONFIG", tempConfigFile.Name())

	localExistingConfigInfoValue := config.GetOneExistingConfigInfo("comp", "app", "project")
	localNonExistingConfigInfoValue := config.GetOneNonExistingConfigInfo()

	tests := []struct {
		name            string
		isConfigExists  bool
		existingConfig  LocalConfigInfo
		wantSpec        Component
		wantPushedState string
	}{
		{
			name:           "case 1: config file exists",
			isConfigExists: true,
			existingConfig: localExistingConfigInfoValue,
			wantSpec: Component{
				Spec: ComponentSpec{
					App:    localExistingConfigInfoValue.GetApplication(),
					Type:   localExistingConfigInfoValue.GetType(),
					Source: localExistingConfigInfoValue.GetSourceLocation(),
					URL: []string{
						localExistingConfigInfoValue.LocalConfig.GetUrl()[0].Name, localExistingConfigInfoValue.LocalConfig.GetUrl()[1].Name,
					},
					Storage: []string{
						localExistingConfigInfoValue.LocalConfig.GetStorage()[0].Name, localExistingConfigInfoValue.LocalConfig.GetStorage()[1].Name,
					},
					Env: []corev1.EnvVar{
						{
							Name:  localExistingConfigInfoValue.LocalConfig.GetEnvs()[0].Name,
							Value: localExistingConfigInfoValue.LocalConfig.GetEnvs()[0].Value,
						},
						{
							Name:  localExistingConfigInfoValue.LocalConfig.GetEnvs()[1].Name,
							Value: localExistingConfigInfoValue.LocalConfig.GetEnvs()[1].Value,
						},
					},
					Ports: localExistingConfigInfoValue.LocalConfig.GetPorts(),
				},
			},
			wantPushedState: "Not Pushed",
		},
		{
			name:           "case 2: config file doesn't exists",
			isConfigExists: false,
			existingConfig: localNonExistingConfigInfoValue,
			wantSpec:       Component{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewLocalConfigInfo("")
			if err != nil {
				t.Error(err)
			}
			cfg = &tt.existingConfig

			got, _ := GetComponentFromConfig(*cfg)

			if !reflect.DeepEqual(got.Spec, tt.wantSpec.Spec) {
				t.Errorf("the component spec is different, want: %v,\n got: %v", tt.wantSpec.Spec, got.Spec)
			}

			if !reflect.DeepEqual(got.Status.State, tt.wantPushedState) {
				t.Errorf("the component status is different, want: %v,\n got: %v", tt.wantSpec.Status.State, got.Status.State)
			}

		})
	}

}
