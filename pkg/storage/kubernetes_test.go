package storage

import (
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openshift/odo/pkg/kclient"
	"github.com/openshift/odo/pkg/localConfigProvider"
	"github.com/openshift/odo/pkg/occlient"
	storageLabels "github.com/openshift/odo/pkg/storage/labels"
	"github.com/openshift/odo/pkg/testingutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
)

func Test_kubernetesClient_ListFromCluster(t *testing.T) {
	type fields struct {
		generic generic
	}
	tests := []struct {
		name         string
		fields       fields
		returnedPods *corev1.PodList
		returnedPVCs *corev1.PersistentVolumeClaimList
		want         StorageList
		wantErr      bool
	}{
		{
			name: "case 1: should error out for multiple pods returned",
			fields: fields{
				generic: generic{
					appName:       "app",
					componentName: "nodejs",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePod("nodejs", "pod-0"),
					*testingutil.CreateFakePod("nodejs", "pod-1"),
				},
			},
			wantErr: true,
		},
		{
			name: "case 2: pod not found",
			fields: fields{
				generic: generic{
					appName:       "app",
					componentName: "nodejs",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{},
			},
			want:    StorageList{},
			wantErr: false,
		},
		{
			name: "case 3: no volume mounts on pod",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePod("nodejs", "pod-0"),
				},
			},
			want:    StorageList{},
			wantErr: false,
		},
		{
			name: "case 4: two volumes mounted on a single container",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{
						testingutil.CreateFakeContainerWithVolumeMounts("container-0", []corev1.VolumeMount{
							{Name: "volume-0-vol", MountPath: "/data"},
							{Name: "volume-1-vol", MountPath: "/path"},
						}),
					}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					*testingutil.FakePVC("volume-0", "5Gi", getStorageLabels("volume-0", "nodejs", "app")),
					*testingutil.FakePVC("volume-1", "10Gi", getStorageLabels("volume-1", "nodejs", "app")),
				},
			},
			want: StorageList{
				Items: []Storage{
					generateStorage(GetMachineReadableFormat("volume-0", "5Gi", "/data"), "", "container-0"),
					generateStorage(GetMachineReadableFormat("volume-1", "10Gi", "/path"), "", "container-0"),
				},
			},
			wantErr: false,
		},
		{
			name: "case 5: one volume is mounted on a single container and another on both",
			fields: fields{
				generic: generic{
					appName:       "app",
					componentName: "nodejs",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{
						testingutil.CreateFakeContainerWithVolumeMounts("container-0", []corev1.VolumeMount{
							{Name: "volume-0-vol", MountPath: "/data"},
							{Name: "volume-1-vol", MountPath: "/path"},
						}),
						testingutil.CreateFakeContainerWithVolumeMounts("container-1", []corev1.VolumeMount{
							{Name: "volume-1-vol", MountPath: "/path"},
						}),
					}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					*testingutil.FakePVC("volume-0", "5Gi", getStorageLabels("volume-0", "nodejs", "app")),
					*testingutil.FakePVC("volume-1", "10Gi", getStorageLabels("volume-1", "nodejs", "app")),
				},
			},
			want: StorageList{
				Items: []Storage{
					generateStorage(GetMachineReadableFormat("volume-0", "5Gi", "/data"), "", "container-0"),
					generateStorage(GetMachineReadableFormat("volume-1", "10Gi", "/path"), "", "container-0"),
					generateStorage(GetMachineReadableFormat("volume-1", "10Gi", "/path"), "", "container-1"),
				},
			},
			wantErr: false,
		},
		{
			name: "case 6: pvc for volumeMount not found",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{
						testingutil.CreateFakeContainerWithVolumeMounts("container-0", []corev1.VolumeMount{
							{Name: "volume-0-vol", MountPath: "/data"},
						}),
						testingutil.CreateFakeContainer("container-1"),
					}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					*testingutil.FakePVC("volume-0", "5Gi", getStorageLabels("volume-0", "nodejs", "app")),
					*testingutil.FakePVC("volume-1", "10Gi", getStorageLabels("volume-1", "nodejs", "app")),
				},
			},
			wantErr: true,
		},
		{
			name: "case 7: the storage label should be used as the name of the storage",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{
						testingutil.CreateFakeContainerWithVolumeMounts("container-0", []corev1.VolumeMount{
							{Name: "volume-0-nodejs-vol", MountPath: "/data"},
						}),
					}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					*testingutil.FakePVC("volume-0-nodejs", "5Gi", getStorageLabels("volume-0", "nodejs", "app")),
				},
			},
			want: StorageList{
				Items: []Storage{
					generateStorage(GetMachineReadableFormat("volume-0", "5Gi", "/data"), "", "container-0"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := kclient.FakeNew()

			fakeClientSet.Kubernetes.PrependReactor("list", "persistentvolumeclaims", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, tt.returnedPVCs, nil
			})

			fakeClientSet.Kubernetes.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, tt.returnedPods, nil
			})

			fkocclient, _ := occlient.FakeNew()
			fkocclient.SetKubeClient(fakeClient)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLocalConfig := localConfigProvider.NewMockLocalConfigProvider(ctrl)
			mockLocalConfig.EXPECT().GetName().Return(tt.fields.generic.componentName).AnyTimes()

			tt.fields.generic.localConfig = mockLocalConfig

			k := kubernetesClient{
				generic: tt.fields.generic,
				client:  *fkocclient,
			}
			got, err := k.ListFromCluster()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListFromCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListFromCluster() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_kubernetesClient_List(t *testing.T) {
	type fields struct {
		generic generic
	}
	tests := []struct {
		name                 string
		fields               fields
		want                 StorageList
		wantErr              bool
		returnedLocalStorage []localConfigProvider.LocalStorage
		returnedPods         *corev1.PodList
		returnedPVCs         *corev1.PersistentVolumeClaimList
	}{
		{
			name: "case 1: no volume on devfile and no pod on cluster",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedLocalStorage: []localConfigProvider.LocalStorage{},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{},
			},
			want:    GetMachineReadableFormatForList(nil),
			wantErr: false,
		},
		{
			name: "case 2: no volume on devfile and pod",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedLocalStorage: []localConfigProvider.LocalStorage{},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{testingutil.CreateFakeContainer("container-0")}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{},
			},
			want:    GetMachineReadableFormatForList(nil),
			wantErr: false,
		},
		{
			name: "case 3: same two volumes on cluster and devFile",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedLocalStorage: []localConfigProvider.LocalStorage{
				{
					Name:      "volume-0",
					Size:      "5Gi",
					Path:      "/data",
					Container: "container-0",
				},
				{
					Name:      "volume-1",
					Size:      "10Gi",
					Path:      "/path",
					Container: "container-0",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{
						testingutil.CreateFakeContainerWithVolumeMounts("container-0", []corev1.VolumeMount{
							{Name: "volume-0-vol", MountPath: "/data"},
							{Name: "volume-1-vol", MountPath: "/path"},
						}),
					}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					*testingutil.FakePVC("volume-0", "5Gi", getStorageLabels("volume-0", "nodejs", "app")),
					*testingutil.FakePVC("volume-1", "10Gi", getStorageLabels("volume-1", "nodejs", "app")),
				},
			},
			want: GetMachineReadableFormatForList([]Storage{
				generateStorage(GetMachineReadableFormat("volume-0", "5Gi", "/data"), StateTypePushed, "container-0"),
				generateStorage(GetMachineReadableFormat("volume-1", "10Gi", "/path"), StateTypePushed, "container-0"),
			}),
			wantErr: false,
		},
		{
			name: "case 4: both volumes, present on the cluster and devFile, are different",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedLocalStorage: []localConfigProvider.LocalStorage{
				{
					Name:      "volume-0",
					Size:      "5Gi",
					Path:      "/data",
					Container: "container-0",
				},
				{
					Name:      "volume-1",
					Size:      "10Gi",
					Path:      "/path",
					Container: "container-0",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{
						testingutil.CreateFakeContainerWithVolumeMounts("container-0", []corev1.VolumeMount{
							{Name: "volume-00-vol", MountPath: "/data"},
							{Name: "volume-11-vol", MountPath: "/path"},
						}),
					}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					*testingutil.FakePVC("volume-00", "5Gi", getStorageLabels("volume-00", "nodejs", "app")),
					*testingutil.FakePVC("volume-11", "10Gi", getStorageLabels("volume-11", "nodejs", "app")),
				},
			},
			want: GetMachineReadableFormatForList([]Storage{
				generateStorage(GetMachineReadableFormat("volume-0", "5Gi", "/data"), StateTypeNotPushed, "container-0"),
				generateStorage(GetMachineReadableFormat("volume-1", "10Gi", "/path"), StateTypeNotPushed, "container-0"),
				generateStorage(GetMachineReadableFormat("volume-00", "5Gi", "/data"), StateTypeLocallyDeleted, "container-0"),
				generateStorage(GetMachineReadableFormat("volume-11", "10Gi", "/path"), StateTypeLocallyDeleted, "container-0"),
			}),
			wantErr: false,
		},
		{
			name: "case 5: two containers with different volumes but one container is not pushed",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedLocalStorage: []localConfigProvider.LocalStorage{
				{
					Name:      "volume-0",
					Size:      "5Gi",
					Path:      "/data",
					Container: "container-0",
				},
				{
					Name:      "volume-1",
					Size:      "10Gi",
					Path:      "/data",
					Container: "container-1",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{
						testingutil.CreateFakeContainerWithVolumeMounts("container-0", []corev1.VolumeMount{
							{Name: "volume-0-vol", MountPath: "/data"},
						}),
					}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					*testingutil.FakePVC("volume-0", "5Gi", map[string]string{"component": "nodejs", storageLabels.DevfileStorageLabel: "volume-0"}),
					*testingutil.FakePVC("volume-0", "5Gi", getStorageLabels("volume-0", "nodejs", "app")),
				},
			},
			want: GetMachineReadableFormatForList([]Storage{
				generateStorage(GetMachineReadableFormat("volume-0", "5Gi", "/data"), StateTypePushed, "container-0"),
				generateStorage(GetMachineReadableFormat("volume-1", "10Gi", "/data"), StateTypeNotPushed, "container-1"),
			}),
			wantErr: false,
		},
		{
			name: "case 6: two containers with different volumes on the cluster but one container is deleted locally",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedLocalStorage: []localConfigProvider.LocalStorage{
				{
					Name:      "volume-0",
					Size:      "5Gi",
					Path:      "/data",
					Container: "container-0",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePodWithContainers("nodejs", "pod-0", []corev1.Container{
						testingutil.CreateFakeContainerWithVolumeMounts("container-0", []corev1.VolumeMount{
							{Name: "volume-0-vol", MountPath: "/data"},
						}),
						testingutil.CreateFakeContainerWithVolumeMounts("container-1", []corev1.VolumeMount{
							{Name: "volume-0-vol", MountPath: "/data"}},
						),
					}),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					*testingutil.FakePVC("volume-0", "5Gi", map[string]string{"component": "nodejs", storageLabels.DevfileStorageLabel: "volume-0"}),
					*testingutil.FakePVC("volume-0", "5Gi", getStorageLabels("volume-0", "nodejs", "app")),
				},
			},
			want: GetMachineReadableFormatForList([]Storage{
				generateStorage(GetMachineReadableFormat("volume-0", "5Gi", "/data"), StateTypePushed, "container-0"),
				generateStorage(GetMachineReadableFormat("volume-0", "5Gi", "/data"), StateTypeLocallyDeleted, "container-1"),
			}),
			wantErr: false,
		},
		{
			name: "case 7: multiple pods are present on the cluster",
			fields: fields{
				generic: generic{
					componentName: "nodejs",
					appName:       "app",
				},
			},
			returnedLocalStorage: []localConfigProvider.LocalStorage{
				{
					Name:      "volume-0",
					Size:      "5Gi",
					Path:      "/data",
					Container: "container-0",
				},
			},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*testingutil.CreateFakePod("nodejs", "pod-0"),
					*testingutil.CreateFakePod("nodejs", "pod-1"),
				},
			},
			returnedPVCs: &corev1.PersistentVolumeClaimList{},
			want:         StorageList{},
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := kclient.FakeNew()

			fakeClientSet.Kubernetes.PrependReactor("list", "persistentvolumeclaims", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, tt.returnedPVCs, nil
			})

			fakeClientSet.Kubernetes.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, tt.returnedPods, nil
			})

			fkocclient, _ := occlient.FakeNew()
			fkocclient.SetKubeClient(fakeClient)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLocalConfig := localConfigProvider.NewMockLocalConfigProvider(ctrl)
			mockLocalConfig.EXPECT().GetName().Return(tt.fields.generic.componentName).AnyTimes()
			mockLocalConfig.EXPECT().ListStorage().Return(tt.returnedLocalStorage, nil)

			tt.fields.generic.localConfig = mockLocalConfig

			k := kubernetesClient{
				generic: tt.fields.generic,
				client:  *fkocclient,
			}
			got, err := k.List()
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("List() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_kubernetesClient_Create(t *testing.T) {
	type fields struct {
		generic generic
		client  occlient.Client
	}
	type args struct {
		storage Storage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "case 1: valid storage",
			fields: fields{
				generic: generic{
					appName:       "app",
					componentName: "nodejs",
				},
			},
			args: args{
				storage: GetMachineFormatWithContainer("storage-0", "5Gi", "/data", "runtime"),
			},
		},
		{
			name: "case 2: invalid storage size",
			fields: fields{
				generic: generic{
					appName:       "app",
					componentName: "nodejs",
				},
			},
			args: args{
				storage: GetMachineFormatWithContainer("storage-0", "example", "/data", "runtime"),
			},
			wantErr: true,
		},
		{
			name: "case 3: valid odo project related storage",
			fields: fields{
				generic: generic{
					appName:       "app",
					componentName: "nodejs",
				},
			},
			args: args{
				storage: GetMachineFormatWithContainer("odo-projects-vol", "5Gi", "/data", "runtime"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := kclient.FakeNew()

			fakeocclient, _ := occlient.FakeNew()
			fakeocclient.SetKubeClient(fkclient)

			k := kubernetesClient{
				generic: tt.fields.generic,
				client:  *fakeocclient,
			}
			if err := k.Create(tt.args.storage); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == true {
				return
			}

			// Check for validating actions performed
			if (len(fkclientset.Kubernetes.Actions()) != 1) && (tt.wantErr != true) {
				t.Errorf("expected 1 action in CreatePVC got: %v", fkclientset.Kubernetes.Actions())
				return
			}

			createdPVC := fkclientset.Kubernetes.Actions()[0].(ktesting.CreateAction).GetObject().(*corev1.PersistentVolumeClaim)
			quantity, err := resource.ParseQuantity(tt.args.storage.Spec.Size)
			if err != nil {
				t.Errorf("failed to create quantity by calling resource.ParseQuantity(%v)", tt.args.storage.Spec.Size)
			}

			wantLabels := storageLabels.GetLabels(tt.args.storage.Name, tt.fields.generic.componentName, tt.fields.generic.appName, true)
			wantLabels[storageLabels.DevfileStorageLabel] = tt.args.storage.Name

			if strings.Contains(tt.args.storage.Name, OdoSourceVolume) {
				// Add label for source pvc
				wantLabels[storageLabels.SourcePVCLabel] = tt.args.storage.Name
			}

			// created PVC should be labeled with labels passed to CreatePVC
			if !reflect.DeepEqual(createdPVC.Labels, wantLabels) {
				t.Errorf("labels in created route is not matching expected labels, expected: %v, got: %v", wantLabels, createdPVC.Labels)
			}
			// name, size of createdPVC should be matching to size, name passed to CreatePVC
			if !reflect.DeepEqual(createdPVC.Spec.Resources.Requests["storage"], quantity) {
				t.Errorf("size of PVC is not matching to expected size, expected: %v, got %v", quantity, createdPVC.Spec.Resources.Requests["storage"])
			}
		})
	}
}

func Test_kubernetesClient_Delete(t *testing.T) {
	pvcName := "pvc-0"
	returnedPVCs := corev1.PersistentVolumeClaimList{
		Items: []corev1.PersistentVolumeClaim{
			*testingutil.FakePVC(pvcName, "5Gi", getStorageLabels("storage-0", "nodejs", "app")),
		},
	}

	type fields struct {
		generic generic
		client  occlient.Client
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "case 1: delete successful",
			fields: fields{
				generic: generic{
					appName:       "app",
					componentName: "nodejs",
				},
			},
			args: args{
				"storage-0",
			},
			wantErr: false,
		},
		{
			name: "case 2: pvc not found",
			fields: fields{
				generic: generic{
					appName:       "app",
					componentName: "nodejs",
				},
			},
			args: args{
				"storage-example",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := kclient.FakeNew()

			fakeocclient, _ := occlient.FakeNew()
			fakeocclient.SetKubeClient(fkclient)

			fkclientset.Kubernetes.PrependReactor("list", "persistentvolumeclaims", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, &returnedPVCs, nil
			})

			fkclientset.Kubernetes.PrependReactor("delete", "persistentvolumeclaims", func(action ktesting.Action) (bool, runtime.Object, error) {
				if action.(ktesting.DeleteAction).GetName() != pvcName {
					t.Errorf("delete called with = %v, want %v", action.(ktesting.DeleteAction).GetName(), pvcName)
				}
				return true, nil, nil
			})

			k := kubernetesClient{
				generic: tt.fields.generic,
				client:  *fakeocclient,
			}
			if err := k.Delete(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(fkclientset.Kubernetes.Actions()) != 2 {
				t.Errorf("expected 2 action, got %v", len(fkclientset.Kubernetes.Actions()))
			}
		})
	}
}
