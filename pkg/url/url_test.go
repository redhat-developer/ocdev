package url

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	kappsv1 "k8s.io/api/apps/v1"

	routev1 "github.com/openshift/api/route/v1"
	applabels "github.com/openshift/odo/pkg/application/labels"
	componentlabels "github.com/openshift/odo/pkg/component/labels"
	"github.com/openshift/odo/pkg/kclient"
	"github.com/openshift/odo/pkg/kclient/fake"
	"github.com/openshift/odo/pkg/localConfigProvider"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/testingutil"
	"github.com/openshift/odo/pkg/url/labels"
	"github.com/openshift/odo/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ktesting "k8s.io/client-go/testing"
)

func TestExists(t *testing.T) {
	tests := []struct {
		name            string
		urlName         string
		componentName   string
		applicationName string
		wantBool        bool
		routes          routev1.RouteList
		labelSelector   string
		wantErr         bool
	}{
		{
			name:            "correct values and Host found",
			urlName:         "nodejs",
			componentName:   "nodejs",
			applicationName: "app",
			routes: routev1.RouteList{
				Items: []routev1.Route{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nodejs",
							Labels: map[string]string{
								applabels.ApplicationLabel:     "app",
								componentlabels.ComponentLabel: "nodejs",
								applabels.ManagedBy:            "odo",
								applabels.ManagerVersion:       version.VERSION,
								labels.URLLabel:                "nodejs",
							},
						},
						Spec: routev1.RouteSpec{
							To: routev1.RouteTargetReference{
								Kind: "Service",
								Name: "nodejs-app",
							},
							Port: &routev1.RoutePort{
								TargetPort: intstr.FromInt(8080),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "wildfly",
							Labels: map[string]string{
								applabels.ApplicationLabel:     "app",
								componentlabels.ComponentLabel: "wildfly",
								applabels.ManagedBy:            "odo",
								applabels.ManagerVersion:       version.VERSION,
								labels.URLLabel:                "wildfly",
							},
						},
						Spec: routev1.RouteSpec{
							To: routev1.RouteTargetReference{
								Kind: "Service",
								Name: "wildfly-app",
							},
							Port: &routev1.RoutePort{
								TargetPort: intstr.FromInt(9100),
							},
						},
					},
				},
			},
			wantBool:      true,
			labelSelector: "app.kubernetes.io/instance=nodejs,app.kubernetes.io/part-of=app",
			wantErr:       false,
		},
		{
			name:            "correct values and Host not found",
			urlName:         "example",
			componentName:   "nodejs",
			applicationName: "app",
			routes: routev1.RouteList{
				Items: []routev1.Route{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nodejs",
							Labels: map[string]string{
								applabels.ApplicationLabel:     "app",
								componentlabels.ComponentLabel: "nodejs",
								applabels.ManagedBy:            "odo",
								applabels.ManagerVersion:       version.VERSION,
								labels.URLLabel:                "nodejs",
							},
						},
						Spec: routev1.RouteSpec{
							To: routev1.RouteTargetReference{
								Kind: "Service",
								Name: "nodejs-app",
							},
							Port: &routev1.RoutePort{
								TargetPort: intstr.FromInt(8080),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "wildfly",
							Labels: map[string]string{
								applabels.ApplicationLabel:     "app",
								componentlabels.ComponentLabel: "wildfly",
								applabels.ManagedBy:            "odo",
								applabels.ManagerVersion:       version.VERSION,
								labels.URLLabel:                "wildfly",
							},
						},
						Spec: routev1.RouteSpec{
							To: routev1.RouteTargetReference{
								Kind: "Service",
								Name: "wildfly-app",
							},
							Port: &routev1.RoutePort{
								TargetPort: intstr.FromInt(9100),
							},
						},
					},
				},
			},
			wantBool:      false,
			labelSelector: "app.kubernetes.io/instance=nodejs,app.kubernetes.io/part-of=app",
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		client, fakeClientSet := occlient.FakeNew()

		fakeClientSet.RouteClientset.PrependReactor("list", "routes", func(action ktesting.Action) (bool, runtime.Object, error) {
			if !reflect.DeepEqual(action.(ktesting.ListAction).GetListRestrictions().Labels.String(), tt.labelSelector) {
				return true, nil, fmt.Errorf("labels not matching with expected values, expected:%s, got:%s", tt.labelSelector, action.(ktesting.ListAction).GetListRestrictions())
			}
			return true, &tt.routes, nil
		})

		exists, err := Exists(client, tt.urlName, tt.componentName, tt.applicationName)
		if err == nil && !tt.wantErr {
			if (len(fakeClientSet.RouteClientset.Actions()) != 1) && (tt.wantErr != true) {
				t.Errorf("expected 1 action in ListRoutes got: %v", fakeClientSet.RouteClientset.Actions())
			}
			if exists != tt.wantBool {
				t.Errorf("expected exists to be:%t, got :%t", tt.wantBool, exists)
			}
		} else if err == nil && tt.wantErr {
			t.Errorf("test failed, expected: %s, got %s", "false", "true")
		} else if err != nil && !tt.wantErr {
			t.Errorf("test failed, expected: %s, got %s", "no error", "error:"+err.Error())
		}
	}
}

func TestPush(t *testing.T) {
	type deleteParameters struct {
		string
		localConfigProvider.URLKind
	}
	type args struct {
		isRouteSupported bool
	}
	tests := []struct {
		name                string
		args                args
		componentName       string
		applicationName     string
		existingLocalURLs   []localConfigProvider.LocalURL
		existingClusterURLs URLList
		deletedItems        []deleteParameters
		createdURLs         []URL
		wantErr             bool
	}{
		{
			name: "no urls on local config and cluster",
			args: args{
				isRouteSupported: true,
			},
			componentName:   "nodejs",
			applicationName: "app",
		},
		{
			name:            "2 urls on local config and 0 on openshift cluster",
			componentName:   "nodejs",
			applicationName: "app",
			args: args{
				isRouteSupported: true,
			},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example",
					Port:   8080,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				},
				{
					Name:   "example-1",
					Port:   9090,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				},
			},
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example",
					Port:   8080,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				}),
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example-1",
					Port:   9090,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				}),
			},
		},
		{
			name:            "0 url on local config and 2 on openshift cluster",
			componentName:   "wildfly",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormat(testingutil.GetSingleRoute("example", 8080, "wildfly", "app")),
				getMachineReadableFormat(testingutil.GetSingleRoute("example-1", 9100, "wildfly", "app")),
			}),
			deletedItems: []deleteParameters{
				{"example", localConfigProvider.ROUTE},
				{"example-1", localConfigProvider.ROUTE},
			},
		},
		{
			name:            "2 url on local config and 2 on openshift cluster, but they are different",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example-local-0",
					Port:   8080,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				},
				{
					Name:   "example-local-1",
					Port:   9090,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				},
			},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormat(testingutil.GetSingleRoute("example", 8080, "wildfly", "app")),
				getMachineReadableFormat(testingutil.GetSingleRoute("example-1", 9100, "wildfly", "app")),
			}),
			deletedItems: []deleteParameters{
				{"example", localConfigProvider.ROUTE},
				{"example-1", localConfigProvider.ROUTE},
			},
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example-local-0",
					Port:   8080,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				}),
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example-local-1",
					Port:   9090,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				}),
			},
		},
		{
			name:            "2 url on local config and openshift cluster are in sync",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example",
					Port:   8080,
					Secure: false,
					Path:   "/",
					Kind:   localConfigProvider.ROUTE,
				},
				{
					Name:   "example-1",
					Port:   9100,
					Secure: false,
					Path:   "/",
					Kind:   localConfigProvider.ROUTE,
				},
			},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormat(testingutil.GetSingleRoute("example", 8080, "wildfly", "app")),
				getMachineReadableFormat(testingutil.GetSingleRoute("example-1", 9100, "wildfly", "app")),
			}),
			createdURLs: []URL{},
		},
		{
			name:              "0 urls on env file and cluster",
			componentName:     "nodejs",
			applicationName:   "app",
			args:              args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{},
		},
		{
			name:            "2 urls on env file and 0 on openshift cluster",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name: "example",
					Host: "com",
					Port: 8080,
					Kind: localConfigProvider.INGRESS,
				},
				{
					Name: "example-1",
					Host: "com",
					Port: 9090,
					Kind: localConfigProvider.INGRESS,
				},
			},
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name: "example",
					Host: "com",
					Port: 8080,
					Kind: localConfigProvider.INGRESS,
				}),
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name: "example-1",
					Host: "com",
					Port: 9090,
					Kind: localConfigProvider.INGRESS,
				}),
			},
		},
		{
			name:              "0 urls on env file and 2 on openshift cluster",
			componentName:     "nodejs",
			applicationName:   "app",
			args:              args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormatIngress(*fake.GetSingleIngress("example-0", "nodejs", "app")),
				getMachineReadableFormatIngress(*fake.GetSingleIngress("example-1", "nodejs", "app")),
			}),
			deletedItems: []deleteParameters{
				{"example-0", localConfigProvider.INGRESS},
				{"example-1", localConfigProvider.INGRESS},
			},
		},
		{
			name:            "2 urls on env file and 2 on openshift cluster, but they are different",
			componentName:   "wildfly",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name: "example-local-0",
					Host: "com",
					Port: 8080,
					Kind: localConfigProvider.INGRESS,
				},
				{
					Name: "example-local-1",
					Host: "com",
					Port: 9090,
					Kind: localConfigProvider.INGRESS,
				},
			},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormatIngress(*fake.GetSingleIngress("example-0", "nodejs", "app")),
				getMachineReadableFormatIngress(*fake.GetSingleIngress("example-1", "nodejs", "app")),
			}),
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name: "example-local-0",
					Host: "com",
					Port: 8080,
					Kind: localConfigProvider.INGRESS,
				}),
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name: "example-local-1",
					Host: "com",
					Port: 9090,
					Kind: localConfigProvider.INGRESS,
				}),
			},
			deletedItems: []deleteParameters{
				{"example-0", localConfigProvider.INGRESS},
				{"example-1", localConfigProvider.INGRESS},
			},
		},
		{
			name:            "2 urls on env file and openshift cluster are in sync",
			componentName:   "wildfly",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name: "example-0",
					Host: "com",
					Port: 8080,
					Kind: localConfigProvider.INGRESS,
					Path: "/",
				},
				{
					Name: "example-1",
					Host: "com",
					Port: 9090,
					Kind: localConfigProvider.INGRESS,
					Path: "/",
				},
			},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormatIngress((*fake.GetIngressListWithMultiple("wildfly", "app")).Items[0]),
				getMachineReadableFormatIngress((*fake.GetIngressListWithMultiple("wildfly", "app")).Items[1]),
			}),
			createdURLs: []URL{},
		},
		{
			name:            "2 (1 ingress,1 route) urls on env file and 2 on openshift cluster (1 ingress,1 route), but they are different",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name: "example-local-0",
					Port: 8080,
					Kind: localConfigProvider.ROUTE,
				},
				{
					Name: "example-local-1",
					Host: "com",
					Port: 9090,
					Kind: localConfigProvider.INGRESS,
				},
			},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormatIngress(*fake.GetSingleIngress("example-0", "nodejs", "app")),
				getMachineReadableFormat(testingutil.GetSingleRoute("example-1", 9090, "nodejs", "app")),
			}),
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name: "example-local-0",
					Port: 8080,
					Kind: localConfigProvider.ROUTE,
				}),
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name: "example-local-1",
					Host: "com",
					Port: 9090,
					Kind: localConfigProvider.INGRESS,
				}),
			},
			deletedItems: []deleteParameters{
				{"example-0", localConfigProvider.INGRESS},
				{"example-1", localConfigProvider.ROUTE},
			},
		},
		{
			name:            "create a ingress on a kubernetes cluster",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: false},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:      "example",
					Host:      "com",
					TLSSecret: "secret",
					Port:      8080,
					Secure:    true,
					Kind:      localConfigProvider.INGRESS,
				},
			},
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:      "example",
					Host:      "com",
					TLSSecret: "secret",
					Port:      8080,
					Secure:    true,
					Kind:      localConfigProvider.INGRESS,
				}),
			},
		},
		{
			name:            "url with same name exists on env and cluster but with different specs",
			componentName:   "nodejs",
			applicationName: "app",
			args: args{
				isRouteSupported: true,
			},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name: "example-local-0",
					Port: 8080,
					Kind: localConfigProvider.ROUTE,
				},
			},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormatIngress(*fake.GetSingleIngress("example-local-0", "nodejs", "app")),
			}),
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name: "example-local-0",
					Port: 8080,
					Kind: localConfigProvider.ROUTE,
				}),
			},
			deletedItems: []deleteParameters{
				{"example-local-0", localConfigProvider.INGRESS},
			},
			wantErr: false,
		},
		{
			name:            "url with same name exists on config and cluster but with different specs",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example-local-0",
					Port:   8080,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				},
			},
			existingClusterURLs: getMachineReadableFormatForList([]URL{
				getMachineReadableFormat(testingutil.GetSingleRoute("example-local-0-app", 9090, "nodejs", "app")),
			}),
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example-local-0",
					Port:   8080,
					Secure: false,
					Kind:   localConfigProvider.ROUTE,
				}),
			},
			deletedItems: []deleteParameters{
				{"example-local-0-app", localConfigProvider.ROUTE},
			},
			wantErr: false,
		},
		{
			name:            "create a secure route url",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example",
					Port:   8080,
					Secure: true,
					Kind:   localConfigProvider.ROUTE,
				},
			},
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example",
					Port:   8080,
					Secure: true,
					Kind:   localConfigProvider.ROUTE,
				}),
			},
		},
		{
			name:            "create a secure ingress url with empty user given tls secret",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example",
					Host:   "com",
					Secure: true,
					Port:   8080,
					Kind:   localConfigProvider.INGRESS,
				},
			},
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example",
					Host:   "com",
					Secure: true,
					Port:   8080,
					Kind:   localConfigProvider.INGRESS,
				}),
			},
		},
		{
			name:            "create a secure ingress url with user given tls secret",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:      "example",
					Host:      "com",
					TLSSecret: "secret",
					Port:      8080,
					Secure:    true,
					Kind:      localConfigProvider.INGRESS,
				},
			},
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:      "example",
					Host:      "com",
					TLSSecret: "secret",
					Port:      8080,
					Secure:    true,
					Kind:      localConfigProvider.INGRESS,
				}),
			},
		},
		{
			name:          "no host defined for ingress should not create any URL",
			componentName: "nodejs",
			args:          args{isRouteSupported: false},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name: "example",
					Port: 8080,
					Kind: localConfigProvider.ROUTE,
				},
			},
			wantErr:     false,
			createdURLs: []URL{},
		},
		{
			name:            "should create route in openshift cluster if endpoint is defined in devfile",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example",
					Port:   8080,
					Kind:   localConfigProvider.ROUTE,
					Secure: false,
				},
			},
			wantErr: false,
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example",
					Port:   8080,
					Kind:   localConfigProvider.ROUTE,
					Secure: false,
				}),
			},
		},
		{
			name:            "should create ingress if endpoint is defined in devfile",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name: "example",
					Host: "com",
					Port: 8080,
					Kind: localConfigProvider.INGRESS,
				},
			},
			wantErr: false,
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name: "example",
					Host: "com",
					Port: 8080,
					Kind: localConfigProvider.INGRESS,
				}),
			},
		},
		{
			name:            "should create route in openshift cluster with path defined in devfile",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example",
					Port:   8080,
					Secure: false,
					Path:   "/testpath",
					Kind:   localConfigProvider.ROUTE,
				},
			},
			wantErr: false,
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example",
					Port:   8080,
					Secure: false,
					Path:   "/testpath",
					Kind:   localConfigProvider.ROUTE,
				}),
			},
		},
		{
			name:            "should create ingress with path defined in devfile",
			componentName:   "nodejs",
			applicationName: "app",
			args:            args{isRouteSupported: true},
			existingLocalURLs: []localConfigProvider.LocalURL{
				{
					Name:   "example",
					Host:   "com",
					Port:   8080,
					Secure: false,
					Path:   "/testpath",
					Kind:   localConfigProvider.INGRESS,
				},
			},
			wantErr: false,
			createdURLs: []URL{
				ConvertLocalURL(localConfigProvider.LocalURL{
					Name:   "example",
					Host:   "com",
					Port:   8080,
					Secure: false,
					Path:   "/testpath",
					Kind:   localConfigProvider.INGRESS,
				}),
			},
		},
	}
	for _, tt := range tests {
		//tt.name = fmt.Sprintf("case %d: ", testNum+1) + tt.name
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLocalConfigProvider := localConfigProvider.NewMockLocalConfigProvider(ctrl)
			mockLocalConfigProvider.EXPECT().GetName().Return(tt.componentName).AnyTimes()
			mockLocalConfigProvider.EXPECT().GetApplication().Return(tt.applicationName).AnyTimes()
			mockLocalConfigProvider.EXPECT().ListURLs().Return(tt.existingLocalURLs, nil)

			mockURLClient := NewMockClient(ctrl)
			mockURLClient.EXPECT().ListFromCluster().Return(tt.existingClusterURLs, nil)

			for i := range tt.createdURLs {
				mockURLClient.EXPECT().Create(tt.createdURLs[i]).Times(1)
			}

			for i := range tt.deletedItems {
				mockURLClient.EXPECT().Delete(gomock.Eq(tt.deletedItems[i].string), gomock.Eq(tt.deletedItems[i].URLKind)).Times(1)
			}

			fakeClient, _ := occlient.FakeNew()
			fakeKClient, fakeKClientSet := kclient.FakeNew()

			fakeKClientSet.Kubernetes.PrependReactor("list", "deployments", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &kappsv1.DeploymentList{
					Items: []kappsv1.Deployment{
						*testingutil.CreateFakeDeployment(tt.componentName),
					},
				}, nil
			})

			fakeClient.SetKubeClient(fakeKClient)

			if err := Push(PushParameters{
				LocalConfig:      mockLocalConfigProvider,
				URLClient:        mockURLClient,
				IsRouteSupported: tt.args.isRouteSupported,
			}); (err != nil) != tt.wantErr {
				t.Errorf("Push() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConvertEnvinfoURL(t *testing.T) {
	serviceName := "testService"
	urlName := "testURL"
	host := "com"
	secretName := "test-tls-secret"
	tests := []struct {
		name       string
		envInfoURL localConfigProvider.LocalURL
		wantURL    URL
	}{
		{
			name: "Case 1: insecure URL",
			envInfoURL: localConfigProvider.LocalURL{
				Name:   urlName,
				Host:   host,
				Port:   8080,
				Secure: false,
				Kind:   localConfigProvider.INGRESS,
			},
			wantURL: URL{
				TypeMeta:   metav1.TypeMeta{Kind: "url", APIVersion: "odo.dev/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: urlName},
				Spec:       URLSpec{Host: fmt.Sprintf("%s.%s", urlName, host), Port: 8080, Secure: false, Kind: localConfigProvider.INGRESS},
			},
		},
		{
			name: "Case 2: secure Ingress URL without tls secret defined",
			envInfoURL: localConfigProvider.LocalURL{
				Name:   urlName,
				Host:   host,
				Port:   8080,
				Secure: true,
				Kind:   localConfigProvider.INGRESS,
			},
			wantURL: URL{
				TypeMeta:   metav1.TypeMeta{Kind: "url", APIVersion: "odo.dev/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: urlName},
				Spec:       URLSpec{Host: fmt.Sprintf("%s.%s", urlName, host), Port: 8080, Secure: true, TLSSecret: fmt.Sprintf("%s-tlssecret", serviceName), Kind: localConfigProvider.INGRESS},
			},
		},
		{
			name: "Case 3: secure Ingress URL with tls secret defined",
			envInfoURL: localConfigProvider.LocalURL{
				Name:      urlName,
				Host:      host,
				Port:      8080,
				Secure:    true,
				TLSSecret: secretName,
				Kind:      localConfigProvider.INGRESS,
			},
			wantURL: URL{
				TypeMeta:   metav1.TypeMeta{Kind: "url", APIVersion: "odo.dev/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: urlName},
				Spec:       URLSpec{Host: fmt.Sprintf("%s.%s", urlName, host), Port: 8080, Secure: true, TLSSecret: secretName, Kind: localConfigProvider.INGRESS},
			},
		},
		{
			name: "Case 4: Insecure route URL",
			envInfoURL: localConfigProvider.LocalURL{
				Name: urlName,
				Port: 8080,
				Kind: localConfigProvider.ROUTE,
			},
			wantURL: URL{
				TypeMeta:   metav1.TypeMeta{Kind: "url", APIVersion: "odo.dev/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: urlName},
				Spec:       URLSpec{Port: 8080, Secure: false, Kind: localConfigProvider.ROUTE},
			},
		},
		{
			name: "Case 4: Secure route URL",
			envInfoURL: localConfigProvider.LocalURL{
				Name:   urlName,
				Port:   8080,
				Secure: true,
				Kind:   localConfigProvider.ROUTE,
			},
			wantURL: URL{
				TypeMeta:   metav1.TypeMeta{Kind: "url", APIVersion: "odo.dev/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: urlName},
				Spec:       URLSpec{Port: 8080, Secure: true, Kind: localConfigProvider.ROUTE},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := ConvertEnvinfoURL(tt.envInfoURL, serviceName)
			if !reflect.DeepEqual(url, tt.wantURL) {
				t.Errorf("Expected %v, got %v", tt.wantURL, url)
			}
		})
	}
}

func TestGetURLString(t *testing.T) {
	cases := []struct {
		name          string
		protocol      string
		URL           string
		ingressDomain string
		isS2I         bool
		expected      string
	}{
		{
			name:          "simple s2i case",
			protocol:      "http",
			URL:           "example.com",
			ingressDomain: "",
			isS2I:         true,
			expected:      "http://example.com",
		},
		{
			name:          "all blank with s2i",
			protocol:      "",
			URL:           "",
			ingressDomain: "",
			isS2I:         true,
			expected:      "",
		},
		{
			name:          "all blank without s2i",
			protocol:      "",
			URL:           "",
			ingressDomain: "",
			isS2I:         false,
			expected:      "",
		},
		{
			name:          "devfile case",
			protocol:      "http",
			URL:           "",
			ingressDomain: "spring-8080.192.168.39.247.nip.io",
			isS2I:         false,
			expected:      "http://spring-8080.192.168.39.247.nip.io",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			output := GetURLString(testCase.protocol, testCase.URL, testCase.ingressDomain, testCase.isS2I)
			if output != testCase.expected {
				t.Errorf("Expected: %v, got %v", testCase.expected, output)

			}
		})
	}
}
