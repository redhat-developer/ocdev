package catalog

import (
	"reflect"
	"testing"

	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/testingutil"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
)

func TestList(t *testing.T) {
	type args struct {
		name       string
		namespace  string
		tags       []string
		hiddenTags []string
	}
	tests := []struct {
		name              string
		args              args
		wantErr           bool
		wantAllTags       []string
		wantNonHiddenTags []string
	}{
		{
			name: "Case 1: Valid image output with one tag which is not hidden",
			args: args{
				name:       "foobar",
				namespace:  "openshift",
				tags:       []string{"latest"},
				hiddenTags: []string{},
			},
			wantErr:           false,
			wantAllTags:       []string{"latest"},
			wantNonHiddenTags: []string{"latest"},
		},
		{
			name: "Case 2: Valid image output with one tag which is hidden",
			args: args{
				name:       "foobar",
				namespace:  "openshift",
				tags:       []string{"latest"},
				hiddenTags: []string{"latest"},
			},
			wantErr:           false,
			wantAllTags:       []string{"latest"},
			wantNonHiddenTags: []string{},
		},
		{
			name: "Case 3: Valid image output with multiple tags none of which are hidden",
			args: args{
				name:       "foobar",
				namespace:  "openshift",
				tags:       []string{"1.0.0", "1.0.1", "0.0.1", "latest"},
				hiddenTags: []string{},
			},
			wantErr:           false,
			wantAllTags:       []string{"1.0.0", "1.0.1", "0.0.1", "latest"},
			wantNonHiddenTags: []string{"1.0.0", "1.0.1", "0.0.1", "latest"},
		},
		{
			name: "Case 4: Valid image output with multiple tags some of which are hidden",
			args: args{
				name:       "foobar",
				namespace:  "openshift",
				tags:       []string{"1.0.0", "1.0.1", "0.0.1", "latest"},
				hiddenTags: []string{"0.0.1", "1.0.0"},
			},
			wantErr:           false,
			wantAllTags:       []string{"1.0.0", "1.0.1", "0.0.1", "latest"},
			wantNonHiddenTags: []string{"1.0.1", "latest"},
		},
		{
			name: "Case 3: Invalid image output with no tags",
			args: args{
				name:      "foobar",
				namespace: "foo",
				tags:      []string{},
			},
			wantErr:           true,
			wantAllTags:       []string{},
			wantNonHiddenTags: []string{},
		},
		{
			name: "Case 4: Valid image with output tags from a different namespace none of which are hidden",
			args: args{
				name:       "foobar",
				namespace:  "foo",
				tags:       []string{"1", "2", "4", "latest", "10"},
				hiddenTags: []string{"1", "2"},
			},
			wantErr:           false,
			wantAllTags:       []string{"1", "2", "4", "latest", "10"},
			wantNonHiddenTags: []string{"4", "latest", "10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Fake the client with the appropriate arguments
			client, fakeClientSet := occlient.FakeNew()
			fakeClientSet.ImageClientset.PrependReactor("list", "imagestreams", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, testingutil.FakeImageStreams(tt.args.name, tt.args.namespace, tt.args.tags), nil
			})
			fakeClientSet.ImageClientset.PrependReactor("list", "imagestreamtags", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, testingutil.FakeImageStreamTags(tt.args.name, tt.args.namespace, tt.args.tags, tt.args.hiddenTags), nil
			})

			// The function we are testing
			output, err := List(client)

			//Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component List() unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			// 1 call for current project + 1 call from openshift project for each of the ImageStream and ImageStreamTag
			if len(fakeClientSet.ImageClientset.Actions()) != 4 {
				t.Errorf("expected 2 ImageClientset.Actions() in List, got: %v", fakeClientSet.ImageClientset.Actions())
			}

			// Check if the output is the same as what's expected (for all tags)
			// and only if output is more than 0 (something is actually returned)
			if len(output) > 0 && !(reflect.DeepEqual(output[0].AllTags, tt.wantAllTags)) {
				t.Errorf("expected all tags: %s, got: %s", tt.wantAllTags, output[0].AllTags)
			}

			// Check if the output is the same as what's expected (for hidden tags)
			// and only if output is more than 0 (something is actually returned)
			if len(output) > 0 && !(reflect.DeepEqual(output[0].NonHiddenTags, tt.wantNonHiddenTags)) {
				t.Errorf("expected non hidden tags: %s, got: %s", tt.wantNonHiddenTags, output[0].NonHiddenTags)
			}

		})
	}
}
