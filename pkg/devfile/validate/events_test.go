package validate

import (
	"testing"

	"github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
)

func Test_validateEvents(t *testing.T) {

	var tests = []struct {
		name    string
		events  v1alpha2.Events
		wantErr bool
	}{
		{
			name: "just postStart event present",
			events: v1alpha2.Events{
				DevWorkspaceEvents: v1alpha2.DevWorkspaceEvents{
					PostStart: []string{"asdf"},
				},
			},
			wantErr: false,
		},
		{
			name: "preStart event present",
			events: v1alpha2.Events{
				DevWorkspaceEvents: v1alpha2.DevWorkspaceEvents{
					PostStart: []string{"asdf"},
					PreStart:  []string{"asdf"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateEvents(tt.events); (err != nil) != tt.wantErr {
				t.Errorf("validateEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
