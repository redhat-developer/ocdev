package generic

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	adaptersCommon "github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
)

func TestValidateComponents(t *testing.T) {

	t.Run("Duplicate volume components present", func(t *testing.T) {

		components := []common.DevfileComponent{
			{
				Name: "myvol",
				Volume: &common.Volume{
					Size: "1Gi",
				},
			},
			{
				Name: "myvol",
				Volume: &common.Volume{
					Size: "1Gi",
				},
			},
		}

		got := ValidateComponents(components)
		want := &DuplicateVolumeComponentsError{}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("TestValidateComponents error - got: '%v', want: '%v'", got, want)
		}
	})

	t.Run("Valid container and volume component", func(t *testing.T) {

		components := []common.DevfileComponent{
			{
				Name: "myvol",
				Volume: &common.Volume{
					Size: "1Gi",
				},
			},
			{
				Name: "container",
				Container: &common.Container{
					VolumeMounts: []common.VolumeMount{
						{
							Name: "myvol",
							Path: "/some/path/",
						},
					},
				},
			},
			{
				Name: "container2",
				Container: &common.Container{
					VolumeMounts: []common.VolumeMount{
						{
							Name: "myvol",
						},
					},
				},
			},
		}

		got := ValidateComponents(components)

		if got != nil {
			t.Errorf("TestValidateComponents error - got: '%v'", got)
		}
	})

	t.Run("Invalid container using reserved env", func(t *testing.T) {

		envName := []string{adaptersCommon.EnvProjectsSrc, adaptersCommon.EnvProjectsRoot}

		for _, env := range envName {
			components := []common.DevfileComponent{
				{
					Name: "container",
					Container: &common.Container{
						Env: []common.Env{
							{
								Name:  env,
								Value: "/some/path/",
							},
						},
					},
				},
			}

			got := ValidateComponents(components)
			want := fmt.Sprintf("env variable %s is reserved and cannot be customized in component container", env)

			if got != nil && !strings.Contains(got.Error(), want) {
				t.Errorf("TestValidateComponents error - got: '%v', want substring: '%v'", got.Error(), want)
			}
		}
	})

	t.Run("Invalid volume component size", func(t *testing.T) {

		components := []common.DevfileComponent{
			{
				Name: "myvol",
				Volume: &common.Volume{
					Size: "randomgarbage",
				},
			},
			{
				Name: "container",
				Container: &common.Container{
					VolumeMounts: []common.VolumeMount{
						{
							Name: "myvol",
							Path: "/some/path/",
						},
					},
				},
			},
		}

		got := ValidateComponents(components)
		want := "size randomgarbage for volume component myvol is invalid"

		if got != nil && !strings.Contains(got.Error(), want) {
			t.Errorf("TestValidateComponents error - got: '%v', want substring: '%v'", got.Error(), want)
		}
	})

	t.Run("Invalid volume mount", func(t *testing.T) {

		components := []common.DevfileComponent{
			{
				Name: "myvol",
				Volume: &common.Volume{
					Size: "2Gi",
				},
			},
			{
				Name: "container",
				Container: &common.Container{
					VolumeMounts: []common.VolumeMount{
						{
							Name: "myinvalidvol",
						},
						{
							Name: "myinvalidvol2",
						},
					},
				},
			},
		}

		got := ValidateComponents(components)
		want := "unable to find volume mount"

		if !strings.Contains(got.Error(), want) {
			t.Errorf("TestValidateComponents error - got: '%v', want substr: '%v'", got.Error(), want)
		}
	})
}
