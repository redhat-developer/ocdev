package utils

import (
	"github.com/openshift/odo/pkg/storage"
	"reflect"
	"strconv"
	"strings"
	"testing"

	devfilev1 "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
	devfileParser "github.com/devfile/library/pkg/devfile/parser"
	"github.com/devfile/library/pkg/testingutil"
	adaptersCommon "github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/kclient"
	odoTestingUtil "github.com/openshift/odo/pkg/testingutil"
	"github.com/openshift/odo/pkg/util"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ktesting "k8s.io/client-go/testing"
)

func TestComponentExists(t *testing.T) {

	tests := []struct {
		name             string
		componentType    devfilev1.ComponentType
		componentName    string
		getComponentName string
		want             bool
		wantErr          bool
	}{
		{
			name:             "Case 1: Valid component name",
			componentName:    "test-name",
			getComponentName: "test-name",
			want:             true,
			wantErr:          false,
		},
		{
			name:             "Case 2: Non-existent component name",
			componentName:    "test-name",
			getComponentName: "fake-component",
			want:             false,
			wantErr:          false,
		},
		{
			name:             "Case 3: Error condition",
			componentName:    "test-name",
			getComponentName: "test-name",
			want:             false,
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fkclient, fkclientset := kclient.FakeNew()
			fkclientset.Kubernetes.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
				emptyDeployment := odoTestingUtil.CreateFakeDeployment("")
				deployment := odoTestingUtil.CreateFakeDeployment(tt.getComponentName)

				if tt.wantErr {
					return true, emptyDeployment, errors.Errorf("deployment get error")
				} else if tt.getComponentName == tt.componentName {
					return true, deployment, nil
				}

				return true, emptyDeployment, kerrors.NewNotFound(schema.GroupResource{}, "")
			})

			// Verify that a component with the specified name exists
			componentExists, err := ComponentExists(*fkclient, tt.getComponentName)
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if !tt.wantErr && componentExists != tt.want {
				t.Errorf("expected %v, actual %v", tt.want, componentExists)
			}

		})
	}

}

func TestAddOdoProjectVolume(t *testing.T) {

	tests := []struct {
		name                         string
		containers                   []corev1.Container
		containerWithProjectVolMount []string
		volMount                     map[string]string
	}{
		{
			name: "Case: Various containers with and without PROJECTS_ROOT",
			containers: []corev1.Container{
				{
					Name: "container1",
					Env: []corev1.EnvVar{
						{
							Name:  adaptersCommon.EnvProjectsRoot,
							Value: "/path1",
						},
					},
				},
				{
					Name: "container2",
					Env: []corev1.EnvVar{
						{
							Name:  adaptersCommon.EnvProjectsRoot,
							Value: "/path2",
						},
					},
				},
				{
					Name: "container3",
					Env: []corev1.EnvVar{
						{
							Name:  "RANDOM",
							Value: "/path3",
						},
					},
				},
			},
			containerWithProjectVolMount: []string{"container1", "container2"},
			volMount: map[string]string{
				"container1": "/path1",
				"container2": "/path2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			AddOdoProjectVolume(&tt.containers)

			for wantContainerName, wantMountPath := range tt.volMount {
				matched := false
				for _, container := range tt.containers {
					if container.Name == wantContainerName {
						for _, volMount := range container.VolumeMounts {
							if volMount.Name == storage.OdoSourceVolume && volMount.MountPath == wantMountPath {
								matched = true
							}
						}
					}
				}

				if !matched {
					t.Error("TestAddOdoProjectVolume error: did not match the volume mount for odo-projects")
				}
			}
		})
	}
}

func TestUpdateContainersWithSupervisord(t *testing.T) {

	command := "ls -la"
	component := "alias1"

	debugCommand := "nodemon --inspect={DEBUG_PORT}"
	debugComponent := "alias2"

	image := "image1"
	workDir := "/root"
	emptyString := ""
	defaultCommand := []string{"tail"}
	execRunGroup := devfilev1.CommandGroup{
		IsDefault: true,
		Kind:      devfilev1.RunCommandGroupKind,
	}
	execDebugGroup := devfilev1.CommandGroup{
		IsDefault: true,
		Kind:      devfilev1.DebugCommandGroupKind,
	}
	defaultArgs := []string{"-f", "/dev/null"}
	supervisordCommand := []string{adaptersCommon.SupervisordBinaryPath}
	supervisordArgs := []string{"-c", adaptersCommon.SupervisordConfFile}

	tests := []struct {
		name                    string
		runCommand              string
		debugCommand            string
		debugPort               int
		containers              []corev1.Container
		execCommands            []devfilev1.Command
		componentType           devfilev1.ComponentType
		expectRunCommand        string
		expectDebugCommand      string
		isSupervisordEntrypoint bool
		wantErr                 bool
	}{
		{
			name:       "Case: Container With Command and Args",
			runCommand: emptyString,
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Command:         defaultCommand,
					Args:            defaultArgs,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectRunCommand:        command,
			isSupervisordEntrypoint: false,
			wantErr:                 false,
		},
		{
			name:       "Case: Container With Command and Args but Missing Work Dir",
			runCommand: emptyString,
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Command:         defaultCommand,
					Args:            defaultArgs,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectRunCommand:        command,
			isSupervisordEntrypoint: false,
			wantErr:                 false,
		},
		{
			name:       "Case: Container With No Command and Args ",
			runCommand: emptyString,
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectRunCommand:        command,
			isSupervisordEntrypoint: true,
			wantErr:                 false,
		},
		{
			name:       "Case: Custom Command Container With No Command and Args ",
			runCommand: "customcommand",
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					Id: "customcommand",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectRunCommand:        command,
			isSupervisordEntrypoint: true,
			wantErr:                 false,
		},
		{
			name:       "Case: Wrong Custom Command Container",
			runCommand: "customcommand123",
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectRunCommand:        command,
			isSupervisordEntrypoint: true,
			wantErr:                 true,
		},

		{
			name:         "Case: empty debug command",
			runCommand:   "customruncommand",
			debugCommand: emptyString,
			debugPort:    5858,
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
				{
					Name:            debugComponent,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					Id: "customruncommand",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
				{
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execDebugGroup,
								},
							},
							CommandLine: debugCommand,
							Component:   debugComponent,
							WorkingDir:  workDir,
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectDebugCommand:      debugCommand,
			expectRunCommand:        command,
			isSupervisordEntrypoint: true,
			wantErr:                 false,
		},
		{
			name:         "Case: custom debug command",
			runCommand:   emptyString,
			debugCommand: "customdebugcommand",
			debugPort:    3000,
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
				{
					Id: "customdebugcommand",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execDebugGroup,
								},
							},
							CommandLine: debugCommand,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectDebugCommand:      debugCommand,
			expectRunCommand:        command,
			isSupervisordEntrypoint: true,
			wantErr:                 false,
		},
		{
			name:         "Case: wrong custom debug command",
			runCommand:   emptyString,
			debugCommand: "customdebugcommand123",
			debugPort:    9090,
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
				{
					Name:            debugComponent,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					Id: "run",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
				{
					Id: "debug",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &devfilev1.CommandGroup{
										IsDefault: true,
										Kind:      devfilev1.BuildCommandGroupKind,
									},
								},
							},
							CommandLine: debugCommand,
							Component:   debugComponent,
							WorkingDir:  workDir,
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectDebugCommand:      debugCommand,
			expectRunCommand:        command,
			isSupervisordEntrypoint: true,
			wantErr:                 true,
		},
		{
			name:       "Case: custom run command with single environment variable",
			runCommand: "customruncommand",
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					Id: "customruncommand",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
							Env: []devfilev1.EnvVar{
								{
									Name:  "env1",
									Value: "value1",
								},
							},
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectRunCommand:        "export " + "env1=\"value1\" && " + command,
			isSupervisordEntrypoint: true,
			wantErr:                 false,
		},
		{
			name:       "Case: custom run command with multiple environment variable",
			runCommand: "customruncommand",
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					Id: "customruncommand",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
							Env: []devfilev1.EnvVar{
								{
									Name:  "env1",
									Value: "value1",
								},
								{
									Name:  "env2",
									Value: "value2 with space",
								},
							},
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectRunCommand:        "export " + "env1=\"value1\" env2=\"value2 with space\" && " + command,
			isSupervisordEntrypoint: true,
			wantErr:                 false,
		},
		{
			name:         "Case: custom debug command with single environment variable",
			runCommand:   emptyString,
			debugCommand: "customdebugcommand",
			debugPort:    3000,
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
				{
					Id: "customdebugcommand",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execDebugGroup,
								},
							},
							CommandLine: debugCommand,
							Component:   component,
							WorkingDir:  workDir,
							Env: []devfilev1.EnvVar{
								{
									Name:  "env1",
									Value: "value1",
								},
							},
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectDebugCommand:      "export " + "env1=\"value1\" && " + debugCommand,
			expectRunCommand:        command,
			isSupervisordEntrypoint: true,
			wantErr:                 false,
		},
		{
			name:         "Case: custom debug command with multiple environment variables",
			runCommand:   emptyString,
			debugCommand: "customdebugcommand",
			debugPort:    3000,
			containers: []corev1.Container{
				{
					Name:            component,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					Env:             []corev1.EnvVar{},
				},
			},
			execCommands: []devfilev1.Command{
				{
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execRunGroup,
								},
							},
							CommandLine: command,
							Component:   component,
							WorkingDir:  workDir,
						},
					},
				},
				{
					Id: "customdebugcommand",
					CommandUnion: devfilev1.CommandUnion{
						Exec: &devfilev1.ExecCommand{
							LabeledCommand: devfilev1.LabeledCommand{
								BaseCommand: devfilev1.BaseCommand{
									Group: &execDebugGroup,
								},
							},
							CommandLine: debugCommand,
							Component:   component,
							WorkingDir:  workDir,
							Env: []devfilev1.EnvVar{
								{
									Name:  "env1",
									Value: "value1",
								},
								{
									Name:  "env2",
									Value: "value2 with space",
								},
							},
						},
					},
				},
			},
			componentType:           devfilev1.ContainerComponentType,
			expectDebugCommand:      "export " + "env1=\"value1\" env2=\"value2 with space\" && " + debugCommand,
			expectRunCommand:        command,
			isSupervisordEntrypoint: true,
			wantErr:                 false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: &testingutil.TestDevfileData{
					Components: []devfilev1.Component{
						{
							Name: component,
							ComponentUnion: devfilev1.ComponentUnion{
								Container: &devfilev1.ContainerComponent{
									Container: devfilev1.Container{
										SourceMapping: "",
									},
								},
							},
						},
						{
							Name: debugComponent,
							ComponentUnion: devfilev1.ComponentUnion{
								Container: &devfilev1.ContainerComponent{
									Container: devfilev1.Container{
										SourceMapping: "",
									},
								},
							},
						},
					},
					Commands: tt.execCommands,
				},
			}

			containers, err := UpdateContainersWithSupervisord(devObj, tt.containers, tt.runCommand, tt.debugCommand, tt.debugPort)

			if tt.wantErr {
				if err == nil {
					t.Error("wanted error but got no error")
				} else {
					// return since we dont want to test anything further
					return
				}
			} else {
				if err != nil {
					t.Errorf("TestUpdateContainersWithSupervisord: unexpected error %v", err)
				}
			}

			// Check if the supervisord volume has been mounted
			supervisordVolumeMountMatched := false
			envRunMatched := false
			envWorkDirMatched := false
			envDebugMatched := false
			envDebugWorkDirMatched := false
			envDebugPortMatched := false

			if tt.execCommands[0].Exec.WorkingDir == "" {
				// if workdir is not present, dont test for matching the env
				envWorkDirMatched = true
			}

			if len(tt.execCommands) >= 2 && tt.execCommands[1].Exec.WorkingDir == "" {
				// if workdir is not present, dont test for matching the env
				envDebugWorkDirMatched = true
			}

			for _, container := range containers {
				for _, testContainer := range tt.containers {
					if container.Name == testContainer.Name {
						for _, volumeMount := range container.VolumeMounts {
							if volumeMount.Name == adaptersCommon.SupervisordVolumeName && volumeMount.MountPath == adaptersCommon.SupervisordMountPath {
								supervisordVolumeMountMatched = true
							}
						}

						for _, envVar := range container.Env {
							if envVar.Name == adaptersCommon.EnvOdoCommandRun && envVar.Value == tt.expectRunCommand {
								envRunMatched = true
							}
							if tt.execCommands[0].Exec.WorkingDir != "" && envVar.Name == adaptersCommon.EnvOdoCommandRunWorkingDir && envVar.Value == tt.execCommands[0].Exec.WorkingDir {
								envWorkDirMatched = true
							}

							// if the debug command is also present
							if len(tt.execCommands) >= 2 {
								// check if the debug command env was set properly
								if envVar.Name == adaptersCommon.EnvOdoCommandDebug && envVar.Value == tt.expectDebugCommand {
									envDebugMatched = true
								}
								// check if the debug command's workingDir env was set properly
								if tt.execCommands[1].Exec.WorkingDir != "" && envVar.Name == adaptersCommon.EnvOdoCommandDebugWorkingDir && envVar.Value == tt.execCommands[1].Exec.WorkingDir {
									envDebugWorkDirMatched = true
								}
								// check if the debug command's debugPort env was set properly
								if envVar.Name == adaptersCommon.EnvDebugPort && envVar.Value == strconv.Itoa(tt.debugPort) {
									envDebugPortMatched = true
								}
							}
						}

						if tt.isSupervisordEntrypoint && (!reflect.DeepEqual(container.Command, supervisordCommand) || !reflect.DeepEqual(container.Args, supervisordArgs)) {
							t.Errorf("TestUpdateContainersWithSupervisord error: commands and args mismatched for container %v, expected command: %v actual command: %v, expected args: %v actual args: %v", component, supervisordCommand, container.Command, supervisordArgs, container.Args)
						} else if !tt.isSupervisordEntrypoint && (!reflect.DeepEqual(container.Command, defaultCommand) || !reflect.DeepEqual(container.Args, defaultArgs)) {
							t.Errorf("TestUpdateContainersWithSupervisord error: commands and args mismatched for container %v, expected command: %v actual command: %v, expected args: %v actual args: %v", component, defaultCommand, container.Command, defaultArgs, container.Args)
						}
					}
				}
			}
			if !supervisordVolumeMountMatched {
				t.Errorf("TestUpdateContainersWithSupervisord error: could not find supervisord volume mounts for container %v", component)
			}
			if !envRunMatched || !envWorkDirMatched {
				t.Errorf("TestUpdateContainersWithSupervisord error: could not find env vars for supervisord in container %v, found command env: %v, found work dir env: %v", component, envRunMatched, envWorkDirMatched)
			}

			if len(tt.execCommands) >= 2 && (!envDebugMatched || !envDebugWorkDirMatched || !envDebugPortMatched) {
				t.Errorf("TestUpdateContainersWithSupervisord error: could not find env vars for supervisord in container %v, found debug env: %v, found work dir env: %v, found debug port env: %v", component, envDebugMatched, envDebugWorkDirMatched, envDebugPortMatched)
			}
		})
	}
}

func TestGetPreStartInitContainers(t *testing.T) {

	containers := []corev1.Container{
		testingutil.CreateFakeContainer("container1"),
		testingutil.CreateFakeContainer("container2"),
	}

	execCommands := []devfilev1.Command{
		{
			Id: "exec1",
			CommandUnion: devfilev1.CommandUnion{
				Exec: &devfilev1.ExecCommand{
					CommandLine: "execcommand1",
					WorkingDir:  "execworkdir1",
					Component:   "container1",
				},
			},
		},
		{
			Id: "exec2",
			CommandUnion: devfilev1.CommandUnion{
				Exec: &devfilev1.ExecCommand{
					CommandLine: "execcommand2",
					WorkingDir:  "",
					Component:   "container1",
				},
			},
		},
		{
			Id: "exec3",
			CommandUnion: devfilev1.CommandUnion{
				Exec: &devfilev1.ExecCommand{
					CommandLine: "execcommand3",
					WorkingDir:  "execworkdir3",
					Component:   "container2",
				},
			},
		},
	}

	compCommands := []devfilev1.Command{
		{
			Id: "comp1",
			CommandUnion: devfilev1.CommandUnion{
				Composite: &devfilev1.CompositeCommand{
					Commands: []string{
						"exec1",
						"exec3",
					},
				},
			},
		},
	}

	longContainerName := "thisisaverylongcontainerandkuberneteshasalimitforanamesize-exec2"
	trimmedLongContainerName := util.TruncateString(longContainerName, containerNameMaxLen)

	tests := []struct {
		name              string
		eventCommands     []string
		wantInitContainer map[string]corev1.Container
		longName          bool
		wantErr           bool
	}{
		{
			name: "Case 1: Composite and Exec events",
			eventCommands: []string{
				"exec1",
				"exec3",
				"exec2",
			},
			wantInitContainer: map[string]corev1.Container{
				"container1-exec1": {
					Command: []string{adaptersCommon.ShellExecutable, "-c", "cd execworkdir1 && execcommand1"},
				},
				"container1-exec2": {
					Command: []string{adaptersCommon.ShellExecutable, "-c", "execcommand2"},
				},
				"container2-exec3": {
					Command: []string{adaptersCommon.ShellExecutable, "-c", "cd execworkdir3 && execcommand3"},
				},
			},
		},
		{
			name: "Case 2: Long Container Name",
			eventCommands: []string{
				"exec2",
			},
			wantInitContainer: map[string]corev1.Container{
				trimmedLongContainerName: {
					Command: []string{adaptersCommon.ShellExecutable, "-c", "execcommand2"},
				},
			},
			longName: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.longName {
				containers[0].Name = longContainerName
				execCommands[1].Exec.Component = longContainerName
			}

			devObj := devfileParser.DevfileObj{
				Data: &testingutil.TestDevfileData{
					Commands: append(execCommands, compCommands...),
					Events: devfilev1.Events{
						WorkspaceEvents: devfilev1.WorkspaceEvents{
							PreStart: tt.eventCommands,
						},
					},
				},
			}

			initContainers, err := GetPreStartInitContainers(devObj, containers)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPreStartInitContainers() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(tt.wantInitContainer) != len(initContainers) {
				t.Errorf("TestGetPreStartInitContainers error: init container length mismatch, wanted %v got %v", len(tt.wantInitContainer), len(initContainers))
			}

			for _, initContainer := range initContainers {
				nameMatched := false
				commandMatched := false
				for containerName, container := range tt.wantInitContainer {
					if strings.Contains(initContainer.Name, containerName) {
						nameMatched = true
					}

					if reflect.DeepEqual(initContainer.Command, container.Command) {
						commandMatched = true
					}

					if !reflect.DeepEqual(initContainer.Args, []string{}) {
						t.Errorf("TestGetPreStartInitContainers error: init container args not empty, got %v", initContainer.Args)
					}
				}

				if !nameMatched {
					t.Errorf("TestGetPreStartInitContainers error: init container name mismatch, container name not present in %v", initContainer.Name)
				}

				if !commandMatched {
					t.Errorf("TestGetPreStartInitContainers error: init container command mismatch, command not found in %v", initContainer.Command)
				}
			}
		})
	}

}
