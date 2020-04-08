package component

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/openshift/odo/pkg/util"
	"github.com/pkg/errors"

	adaptersCommon "github.com/openshift/odo/pkg/devfile/adapters/common"
	devfileParser "github.com/openshift/odo/pkg/devfile/parser"
	versionsCommon "github.com/openshift/odo/pkg/devfile/parser/data/common"
	"github.com/openshift/odo/pkg/kclient"
	"github.com/openshift/odo/pkg/testingutil"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ktesting "k8s.io/client-go/testing"
)

func TestCreateOrUpdateComponent(t *testing.T) {

	testComponentName := "test"

	tests := []struct {
		name          string
		componentType versionsCommon.DevfileComponentType
		running       bool
		wantErr       bool
	}{
		{
			name:          "Case: Invalid devfile",
			componentType: "",
			running:       false,
			wantErr:       true,
		},
		{
			name:          "Case: Valid devfile",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			running:       false,
			wantErr:       false,
		},
		{
			name:          "Case: Invalid devfile, already running component",
			componentType: "",
			running:       true,
			wantErr:       true,
		},
		{
			name:          "Case: Valid devfile, already running component",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			running:       true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: tt.componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: testComponentName,
				Devfile:       devObj,
			}

			fkclient, fkclientset := kclient.FakeNew()
			fkWatch := watch.NewFake()

			// Change the status
			go func() {
				fkWatch.Modify(kclient.FakePodStatus(corev1.PodRunning, testComponentName))
			}()
			fkclientset.Kubernetes.PrependWatchReactor("pods", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			componentAdapter := New(adapterCtx, *fkclient)
			err := componentAdapter.createOrUpdateComponent(tt.running)

			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component adapter create unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestGetFirstContainerWithSourceVolume(t *testing.T) {
	tests := []struct {
		name       string
		containers []corev1.Container
		want       string
		wantErr    bool
	}{
		{
			name: "Case: One container, no volumes",
			containers: []corev1.Container{
				{
					Name: "test",
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Case: One container, no source volume",
			containers: []corev1.Container{
				{
					Name: "test",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name: "test",
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Case: One container, source volume",
			containers: []corev1.Container{
				{
					Name: "test",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name: kclient.OdoSourceVolume,
						},
					},
				},
			},
			want:    "test",
			wantErr: false,
		},
		{
			name: "Case: One container, multiple volumes",
			containers: []corev1.Container{
				{
					Name: "test",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name: "test",
						},
						{
							Name: kclient.OdoSourceVolume,
						},
					},
				},
			},
			want:    "test",
			wantErr: false,
		},
		{
			name: "Case: Multiple containers, no source volumes",
			containers: []corev1.Container{
				{
					Name: "test",
				},
				{
					Name: "test",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name: "test",
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Case: Multiple containers, multiple volumes",
			containers: []corev1.Container{
				{
					Name: "test",
				},
				{
					Name: "container-two",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name: "test",
						},
						{
							Name: kclient.OdoSourceVolume,
						},
					},
				},
			},
			want:    "container-two",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		container, err := getFirstContainerWithSourceVolume(tt.containers)
		if container != tt.want {
			t.Errorf("expected %s, actual %s", tt.want, container)
		}

		if !tt.wantErr == (err != nil) {
			t.Errorf("expected %v, actual %v", tt.wantErr, err)
		}
	}
}

func TestGetSyncFolder(t *testing.T) {
	projectNames := []string{"some-name", "another-name"}
	projectRepos := []string{"https://github.com/some/repo.git", "https://github.com/another/repo.git"}
	projectClonePath := "src/github.com/golang/example/"
	invalidClonePaths := []string{"/var", "../var", "pkg/../../var"}

	tests := []struct {
		name     string
		projects []versionsCommon.DevfileProject
		want     string
		wantErr  bool
	}{
		{
			name:     "Case 1: No projects",
			projects: []versionsCommon.DevfileProject{},
			want:     kclient.OdoSourceVolumeMount,
			wantErr:  false,
		},
		{
			name: "Case 2: One project",
			projects: []versionsCommon.DevfileProject{
				{
					Name: projectNames[0],
					Source: versionsCommon.DevfileProjectSource{
						Type:     versionsCommon.DevfileProjectTypeGit,
						Location: projectRepos[0],
					},
				},
			},
			want:    filepath.ToSlash(filepath.Join(kclient.OdoSourceVolumeMount, projectNames[0])),
			wantErr: false,
		},
		{
			name: "Case 3: Multiple projects",
			projects: []versionsCommon.DevfileProject{
				{
					Name: projectNames[0],
					Source: versionsCommon.DevfileProjectSource{
						Type:     versionsCommon.DevfileProjectTypeGit,
						Location: projectRepos[0],
					},
				},
				{
					Name: projectNames[1],
					Source: versionsCommon.DevfileProjectSource{
						Type:     versionsCommon.DevfileProjectTypeGit,
						Location: projectRepos[1],
					},
				},
			},
			want:    kclient.OdoSourceVolumeMount,
			wantErr: false,
		},
		{
			name: "Case 4: Clone path set",
			projects: []versionsCommon.DevfileProject{
				{
					ClonePath: &projectClonePath,
					Name:      projectNames[0],
					Source: versionsCommon.DevfileProjectSource{
						Type:     versionsCommon.DevfileProjectTypeGit,
						Location: projectRepos[0],
					},
				},
			},
			want:    filepath.ToSlash(filepath.Join(kclient.OdoSourceVolumeMount, projectClonePath)),
			wantErr: false,
		},
		{
			name: "Case 5: Invalid clone path, set with absolute path",
			projects: []versionsCommon.DevfileProject{
				{
					ClonePath: &invalidClonePaths[0],
					Name:      projectNames[0],
					Source: versionsCommon.DevfileProjectSource{
						Type:     versionsCommon.DevfileProjectTypeGit,
						Location: projectRepos[0],
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Case 6: Invalid clone path, starts with ..",
			projects: []versionsCommon.DevfileProject{
				{
					ClonePath: &invalidClonePaths[1],
					Name:      projectNames[0],
					Source: versionsCommon.DevfileProjectSource{
						Type:     versionsCommon.DevfileProjectTypeGit,
						Location: projectRepos[0],
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Case 7: Invalid clone path, contains ..",
			projects: []versionsCommon.DevfileProject{
				{
					ClonePath: &invalidClonePaths[2],
					Name:      projectNames[0],
					Source: versionsCommon.DevfileProjectSource{
						Type:     versionsCommon.DevfileProjectTypeGit,
						Location: projectRepos[0],
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		syncFolder, err := getSyncFolder(tt.projects)

		if !tt.wantErr == (err != nil) {
			t.Errorf("expected %v, actual %v", tt.wantErr, err)
		}

		if syncFolder != tt.want {
			t.Errorf("expected %s, actual %s", tt.want, syncFolder)
		}
	}
}

func TestGetCmdToCreateSyncFolder(t *testing.T) {
	tests := []struct {
		name       string
		syncFolder string
		want       []string
	}{
		{
			name:       "Case 1: Sync to /projects",
			syncFolder: kclient.OdoSourceVolumeMount,
			want:       []string{"mkdir", "-p", kclient.OdoSourceVolumeMount},
		},
		{
			name:       "Case 2: Sync subdir of /projects",
			syncFolder: kclient.OdoSourceVolumeMount + "/someproject",
			want:       []string{"mkdir", "-p", kclient.OdoSourceVolumeMount + "/someproject"},
		},
	}
	for _, tt := range tests {
		cmdArr := getCmdToCreateSyncFolder(tt.syncFolder)
		if !reflect.DeepEqual(tt.want, cmdArr) {
			t.Errorf("Expected %s, got %s", tt.want, cmdArr)
		}
	}
}

func TestGetCmdToDeleteFiles(t *testing.T) {
	syncFolder := "/projects/hello-world"

	tests := []struct {
		name       string
		delFiles   []string
		syncFolder string
		want       []string
	}{
		{
			name:       "Case 1: One deleted file",
			delFiles:   []string{"test.txt"},
			syncFolder: kclient.OdoSourceVolumeMount,
			want:       []string{"rm", "-rf", kclient.OdoSourceVolumeMount + "/test.txt"},
		},
		{
			name:       "Case 2: Multiple deleted files, default sync folder",
			delFiles:   []string{"test.txt", "hello.c"},
			syncFolder: kclient.OdoSourceVolumeMount,
			want:       []string{"rm", "-rf", kclient.OdoSourceVolumeMount + "/test.txt", kclient.OdoSourceVolumeMount + "/hello.c"},
		},
		{
			name:       "Case 2: Multiple deleted files, different sync folder",
			delFiles:   []string{"test.txt", "hello.c"},
			syncFolder: syncFolder,
			want:       []string{"rm", "-rf", syncFolder + "/test.txt", syncFolder + "/hello.c"},
		},
	}
	for _, tt := range tests {
		cmdArr := getCmdToDeleteFiles(tt.delFiles, tt.syncFolder)
		if !reflect.DeepEqual(tt.want, cmdArr) {
			t.Errorf("Expected %s, got %s", tt.want, cmdArr)
		}
	}
}

func TestDoesComponentExist(t *testing.T) {

	tests := []struct {
		name             string
		componentType    versionsCommon.DevfileComponentType
		componentName    string
		getComponentName string
		want             bool
	}{
		{
			name:             "Case 1: Valid component name",
			componentType:    versionsCommon.DevfileComponentTypeDockerimage,
			componentName:    "test-name",
			getComponentName: "test-name",
			want:             true,
		},
		{
			name:             "Case 2: Non-existent component name",
			componentType:    versionsCommon.DevfileComponentTypeDockerimage,
			componentName:    "test-name",
			getComponentName: "fake-component",
			want:             false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: tt.componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: tt.componentName,
				Devfile:       devObj,
			}

			fkclient, fkclientset := kclient.FakeNew()
			fkWatch := watch.NewFake()

			fkclientset.Kubernetes.PrependWatchReactor("pods", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			// DoesComponentExist requires an already started component, so start it.
			componentAdapter := New(adapterCtx, *fkclient)
			err := componentAdapter.createOrUpdateComponent(false)

			// Checks for unexpected error cases
			if err != nil {
				t.Errorf("component adapter start unexpected error %v", err)
			}

			// Verify that a comopnent with the specified name exists
			componentExists := componentAdapter.DoesComponentExist(tt.getComponentName)
			if componentExists != tt.want {
				t.Errorf("expected %v, actual %v", tt.want, componentExists)
			}

		})
	}

}

func TestWaitAndGetComponentPod(t *testing.T) {

	testComponentName := "test"

	tests := []struct {
		name          string
		componentType versionsCommon.DevfileComponentType
		status        corev1.PodPhase
		wantErr       bool
	}{
		{
			name:          "Case 1: Running",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			status:        corev1.PodRunning,
			wantErr:       false,
		},
		{
			name:          "Case 2: Failed pod",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			status:        corev1.PodFailed,
			wantErr:       true,
		},
		{
			name:          "Case 3: Unknown pod",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			status:        corev1.PodUnknown,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: tt.componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: testComponentName,
				Devfile:       devObj,
			}

			fkclient, fkclientset := kclient.FakeNew()
			fkWatch := watch.NewFake()

			// Change the status
			go func() {
				fkWatch.Modify(kclient.FakePodStatus(tt.status, testComponentName))
			}()

			fkclientset.Kubernetes.PrependWatchReactor("pods", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			componentAdapter := New(adapterCtx, *fkclient)
			_, err := componentAdapter.waitAndGetComponentPod(false)

			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component adapter create unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestAdapterDelete(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name               string
		args               args
		existingDeployment *v1.Deployment
		componentName      string
		componentExists    bool
		wantErr            bool
	}{
		{
			name: "case 1: component exists and given labels are valid",
			args: args{labels: map[string]string{
				"component": "component",
			}},
			existingDeployment: testingutil.CreateFakeDeployment("fronted"),
			componentName:      "component",
			componentExists:    true,
			wantErr:            false,
		},
		{
			name:               "case 2: component exists and given labels are not valid",
			args:               args{labels: nil},
			existingDeployment: testingutil.CreateFakeDeployment("fronted"),
			componentName:      "component",
			componentExists:    true,
			wantErr:            true,
		},
		{
			name: "case 3: component doesn't exists",
			args: args{labels: map[string]string{
				"component": "component",
			}},
			existingDeployment: testingutil.CreateFakeDeployment("fronted"),
			componentName:      "component",
			componentExists:    false,
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: "nodejs",
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: tt.componentName,
				Devfile:       devObj,
			}

			if !tt.componentExists {
				adapterCtx.ComponentName = "doesNotExists"
			}

			fkclient, fkclientset := kclient.FakeNew()

			a := Adapter{
				Client:         *fkclient,
				AdapterContext: adapterCtx,
			}

			fkclientset.Kubernetes.PrependReactor("delete-collection", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
				if util.ConvertLabelsToSelector(tt.args.labels) != action.(ktesting.DeleteCollectionAction).GetListRestrictions().Labels.String() {
					return true, nil, errors.Errorf("collection labels are not matching, wanted: %v, got: %v", util.ConvertLabelsToSelector(tt.args.labels), action.(ktesting.DeleteCollectionAction).GetListRestrictions().Labels.String())
				}
				return true, nil, nil
			})

			fkclientset.Kubernetes.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
				if action.(ktesting.GetAction).GetName() != tt.componentName {
					return true, nil, errors.Errorf("get action called with different component name, want: %s, got: %s", action.(ktesting.GetAction).GetName(), tt.componentName)
				}
				return true, tt.existingDeployment, nil
			})

			if err := a.Delete(tt.args.labels); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
