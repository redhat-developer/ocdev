package storage

import (
	"testing"

	devfilev1 "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
	"github.com/devfile/api/v2/pkg/attributes"
	"github.com/devfile/library/pkg/devfile/generator"
	devfileParser "github.com/devfile/library/pkg/devfile/parser"
	"github.com/devfile/library/pkg/devfile/parser/data"
	parsercommon "github.com/devfile/library/pkg/devfile/parser/data/v2/common"
	"github.com/openshift/odo/pkg/testingutil"
	v1 "k8s.io/api/core/v1"
)

func TestGetPVC(t *testing.T) {

	tests := []struct {
		pvc        string
		volumeName string
	}{
		{
			pvc:        "mypvc",
			volumeName: "myvolume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.volumeName, func(t *testing.T) {
			volume := getPVC(tt.volumeName, tt.pvc)

			if volume.Name != tt.volumeName {
				t.Errorf("TestGetPVC error: volume name does not match; expected %s got %s", tt.volumeName, volume.Name)
			}

			if volume.PersistentVolumeClaim.ClaimName != tt.pvc {
				t.Errorf("TestGetPVC error: pvc name does not match; expected %s got %s", tt.pvc, volume.PersistentVolumeClaim.ClaimName)
			}
		})
	}
}

func TestAddVolumeMountToContainers(t *testing.T) {

	tests := []struct {
		podName                string
		namespace              string
		serviceAccount         string
		pvc                    string
		volumeName             string
		containerMountPathsMap map[string][]string
		container              v1.Container
		labels                 map[string]string
		wantErr                bool
	}{
		{
			podName:        "podSpecTest",
			namespace:      "default",
			serviceAccount: "default",
			pvc:            "mypvc",
			volumeName:     "myvolume",
			containerMountPathsMap: map[string][]string{
				"container1": {"/tmp/path1", "/tmp/path2"},
			},
			container: v1.Container{
				Name:            "container1",
				Image:           "image1",
				ImagePullPolicy: v1.PullAlways,

				Command: []string{"tail"},
				Args:    []string{"-f", "/dev/null"},
				Env:     []v1.EnvVar{},
			},
			labels: map[string]string{
				"app":       "app",
				"component": "frontend",
			},
			wantErr: false,
		},
		{
			podName:        "podSpecTest",
			namespace:      "default",
			serviceAccount: "default",
			pvc:            "mypvc",
			volumeName:     "myvolume",
			containerMountPathsMap: map[string][]string{
				"container1": {"/tmp/path1", "/tmp/path2"},
			},
			container: v1.Container{},
			labels: map[string]string{
				"app":       "app",
				"component": "frontend",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.podName, func(t *testing.T) {
			containers := []v1.Container{tt.container}
			initContainers := []v1.Container{}
			addVolumeMountToContainers(containers, initContainers, tt.volumeName, tt.containerMountPathsMap)

			mountPathCount := 0
			for _, container := range containers {
				if container.Name == tt.container.Name {
					for _, volumeMount := range container.VolumeMounts {
						if volumeMount.Name == tt.volumeName {
							for _, mountPath := range tt.containerMountPathsMap[tt.container.Name] {
								if volumeMount.MountPath == mountPath {
									mountPathCount++
								}
							}
						}
					}
				}
			}

			if mountPathCount != len(tt.containerMountPathsMap[tt.container.Name]) {
				t.Errorf("Volume Mounts for %s have not been properly mounted to the container", tt.volumeName)
			}
		})
	}
}

func TestGetVolumesAndVolumeMounts(t *testing.T) {

	type testVolumeMountInfo struct {
		mountPath  string
		volumeName string
	}

	tests := []struct {
		name                string
		components          []devfilev1.Component
		volumeNameToVolInfo map[string]VolumeInfo
		wantContainerToVol  map[string][]testVolumeMountInfo
		wantErr             bool
	}{
		{
			name:       "One volume mounted",
			components: []devfilev1.Component{testingutil.GetFakeContainerComponent("comp1"), testingutil.GetFakeContainerComponent("comp2")},
			volumeNameToVolInfo: map[string]VolumeInfo{
				"myvolume1": {
					PVCName:    "volume1-pvc",
					VolumeName: "volume1-pvc-vol",
				},
			},
			wantContainerToVol: map[string][]testVolumeMountInfo{
				"comp1": {
					{
						mountPath:  "/my/volume/mount/path1",
						volumeName: "volume1-pvc-vol",
					},
				},
				"comp2": {
					{
						mountPath:  "/my/volume/mount/path1",
						volumeName: "volume1-pvc-vol",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "One volume mounted at diff locations",
			components: []devfilev1.Component{
				{
					Name: "container1",
					ComponentUnion: devfilev1.ComponentUnion{
						Container: &devfilev1.ContainerComponent{
							Container: devfilev1.Container{
								VolumeMounts: []devfilev1.VolumeMount{
									{
										Name: "volume1",
										Path: "/path1",
									},
									{
										Name: "volume1",
										Path: "/path2",
									},
								},
							},
						},
					},
				},
			},
			volumeNameToVolInfo: map[string]VolumeInfo{
				"volume1": {
					PVCName:    "volume1-pvc",
					VolumeName: "volume1-pvc-vol",
				},
			},
			wantContainerToVol: map[string][]testVolumeMountInfo{
				"container1": {
					{
						mountPath:  "/path1",
						volumeName: "volume1-pvc-vol",
					},
					{
						mountPath:  "/path2",
						volumeName: "volume1-pvc-vol",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "One volume mounted at diff container components",
			components: []devfilev1.Component{
				{
					Name: "container1",
					ComponentUnion: devfilev1.ComponentUnion{
						Container: &devfilev1.ContainerComponent{
							Container: devfilev1.Container{
								VolumeMounts: []devfilev1.VolumeMount{
									{
										Name: "volume1",
										Path: "/path1",
									},
								},
							},
						},
					},
				},
				{
					Name: "container2",
					ComponentUnion: devfilev1.ComponentUnion{
						Container: &devfilev1.ContainerComponent{
							Container: devfilev1.Container{
								VolumeMounts: []devfilev1.VolumeMount{
									{
										Name: "volume1",
										Path: "/path2",
									},
								},
							},
						},
					},
				},
			},
			volumeNameToVolInfo: map[string]VolumeInfo{
				"volume1": {
					PVCName:    "volume1-pvc",
					VolumeName: "volume1-pvc-vol",
				},
			},
			wantContainerToVol: map[string][]testVolumeMountInfo{
				"container1": {
					{
						mountPath:  "/path1",
						volumeName: "volume1-pvc-vol",
					},
				},
				"container2": {
					{
						mountPath:  "/path2",
						volumeName: "volume1-pvc-vol",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid case",
			components: []devfilev1.Component{
				{
					Name: "container1",
					Attributes: attributes.Attributes{}.FromStringMap(map[string]string{
						"firstString": "firstStringValue",
					}),
					ComponentUnion: devfilev1.ComponentUnion{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			devObj := devfileParser.DevfileObj{
				Data: func() data.DevfileData {
					devfileData, err := data.NewDevfileData(string(data.APISchemaVersion200))
					if err != nil {
						t.Error(err)
					}
					err = devfileData.AddComponents(tt.components)
					if err != nil {
						t.Error(err)
					}
					return devfileData
				}(),
			}

			containers, err := generator.GetContainers(devObj, parsercommon.DevfileOptions{})
			if err != nil {
				t.Errorf("TestGetVolumesAndVolumeMounts error - %v", err)
				return
			}

			var options parsercommon.DevfileOptions
			if tt.wantErr {
				options = parsercommon.DevfileOptions{
					Filter: map[string]interface{}{
						"firstString": "firstStringValue",
					},
				}
			}

			initContainers := []v1.Container{}
			pvcVols, err := GetVolumesAndVolumeMounts(devObj, containers, initContainers, tt.volumeNameToVolInfo, options)
			if !tt.wantErr && err != nil {
				t.Errorf("TestGetVolumesAndVolumeMounts unexpected error: %v", err)
				return
			} else if tt.wantErr && err != nil {
				return
			} else if tt.wantErr && err == nil {
				t.Error("TestGetVolumesAndVolumeMounts expected error but got nil")
				return
			}

			// check if the pvc volumes returned are correct
			for _, volInfo := range tt.volumeNameToVolInfo {
				matched := false
				for _, pvcVol := range pvcVols {
					if volInfo.VolumeName == pvcVol.Name && pvcVol.PersistentVolumeClaim != nil && volInfo.PVCName == pvcVol.PersistentVolumeClaim.ClaimName {
						matched = true
					}
				}

				if !matched {
					t.Errorf("TestGetVolumesAndVolumeMounts error - could not find volume details %s in the actual result", volInfo.VolumeName)
				}
			}

			// check the volume mounts of the containers
			for _, container := range containers {
				if volMounts, ok := tt.wantContainerToVol[container.Name]; !ok {
					t.Errorf("TestGetVolumesAndVolumeMounts error - did not find the expected container %s", container.Name)
					return
				} else {
					for _, expectedVolMount := range volMounts {
						matched := false
						for _, actualVolMount := range container.VolumeMounts {
							if expectedVolMount.volumeName == actualVolMount.Name && expectedVolMount.mountPath == actualVolMount.MountPath {
								matched = true
							}
						}

						if !matched {
							t.Errorf("TestGetVolumesAndVolumeMounts error - could not find volume mount details for path %s in the actual result for container %s", expectedVolMount.mountPath, container.Name)
						}
					}
				}
			}
		})
	}
}
