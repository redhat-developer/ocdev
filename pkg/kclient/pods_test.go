package kclient

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/watch"

	ktesting "k8s.io/client-go/testing"
)

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
				fkWatch.Modify(FakePodStatus(tt.status, tt.podName))
			}()

			fkclientset.Kubernetes.PrependWatchReactor("pods", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			podSelector := fmt.Sprintf("deployment=%s", tt.podName)
			timeout := int64(10)
			watchOptions := metav1.ListOptions{
				LabelSelector:  podSelector,
				TimeoutSeconds: &timeout,
			}
			pod, err := fkclient.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for component to start", false)

			if !tt.wantErr == (err != nil) {
				t.Fatalf(" client.WaitAndGetPod(string) unexpected error %v, wantErr %v", err, tt.wantErr)
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

func TestGetOnePodFromSelector(t *testing.T) {
	fakePod := FakePodStatus(corev1.PodRunning, "nodejs")
	fakePod.Labels["component"] = "nodejs"

	type args struct {
		selector string
	}
	tests := []struct {
		name         string
		args         args
		returnedPods *corev1.PodList
		want         *corev1.Pod
		wantErr      bool
	}{
		{
			name: "valid number of pods",
			args: args{selector: fmt.Sprintf("component=%s", "nodejs")},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*fakePod,
				},
			},
			want:    fakePod,
			wantErr: false,
		},
		{
			name: "zero pods",
			args: args{selector: fmt.Sprintf("component=%s", "nodejs")},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{},
			},
			want:    &corev1.Pod{},
			wantErr: true,
		},
		{
			name: "mutiple pods",
			args: args{selector: fmt.Sprintf("component=%s", "nodejs")},
			returnedPods: &corev1.PodList{
				Items: []corev1.Pod{
					*fakePod,
					*fakePod,
				},
			},
			want:    &corev1.Pod{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fkclient, fkclientset := FakeNew()

			fkclientset.Kubernetes.PrependReactor("list", "pods", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				if action.(ktesting.ListAction).GetListRestrictions().Labels.String() != fmt.Sprintf("component=%s", "nodejs") {
					t.Errorf("list called with different selector want:%s, got:%s", fmt.Sprintf("component=%s", "nodejs"), action.(ktesting.ListAction).GetListRestrictions().Labels.String())
				}
				return true, tt.returnedPods, nil
			})

			got, err := fkclient.GetOnePodFromSelector(tt.args.selector)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOnePodFromSelector() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if tt.wantErr && err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOnePodFromSelector() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPodUsingComponentName(t *testing.T) {
	fakePod := FakePodStatus(corev1.PodRunning, "nodejs")
	fakePod.Labels["component"] = "nodejs"

	type args struct {
		componentName string
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Pod
		wantErr bool
	}{
		{
			name: "list called with same component name",
			args: args{
				componentName: "nodejs",
			},
			want:    fakePod,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()

			fkclientset.Kubernetes.PrependReactor("list", "pods", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				if action.(ktesting.ListAction).GetListRestrictions().Labels.String() != fmt.Sprintf("component=%s", tt.args.componentName) {
					t.Errorf("list called with different selector want:%s, got:%s", fmt.Sprintf("component=%s", tt.args.componentName), action.(ktesting.ListAction).GetListRestrictions().Labels.String())
				}
				return true, &corev1.PodList{
					Items: []corev1.Pod{
						*fakePod,
					},
				}, nil
			})

			got, err := fkclient.GetPodUsingComponentName(tt.args.componentName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPodUsingComponentName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPodUsingComponentName() got = %v, want %v", got, tt.want)
			}
		})
	}
}
