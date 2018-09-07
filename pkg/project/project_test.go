package project

import (
	"reflect"
	"testing"

	"github.com/bouk/monkey"
	"github.com/redhat-developer/odo/pkg/occlient"
	"github.com/redhat-developer/odo/pkg/testingutil"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
)

func TestDelete(t *testing.T) {
	tests := []struct {
		name        string
		wantErr     bool
		projectName string
	}{
		{
			name:        "Test project delete for multiple projects",
			wantErr:     false,
			projectName: "prj2",
		},
		{
			name:        "Test only project delete",
			wantErr:     false,
			projectName: "prj1",
		},
		{
			name:        "Test delete non existant project",
			wantErr:     true,
			projectName: "prj4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Fake the client with the appropriate arguments
			client, fakeClientSet := occlient.FakeNew()

			fakeClientSet.ProjClientset.PrependReactor("list", "projects", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, testingutil.FakeProjects(), nil
			})

			fakeClientSet.ProjClientset.PrependReactor("delete", "projects", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, nil, nil
			})

			var guard *monkey.PatchGuard
			guard = monkey.PatchInstanceMethod(reflect.TypeOf(client), "SetCurrentProject", func(c *occlient.Client, project string) error {
				guard.Unpatch()
				defer guard.Restore()

				return nil
			})

			// The function we are testing
			err := Delete(client, tt.projectName)

			// Checks for error in positive cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("project Delete() unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
