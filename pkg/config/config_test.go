package config

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestGetOcDevConfigFile(t *testing.T) {
	// TODO: implement this
}

func TestNew(t *testing.T) {

	tempConfigFile, err := ioutil.TempFile("", "ocdevconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer tempConfigFile.Close()
	os.Setenv(configEnvName, tempConfigFile.Name())

	tests := []struct {
		name    string
		output  *ConfigInfo
		success bool
	}{
		{
			name: "Test filename is being set",
			output: &ConfigInfo{
				Filename: tempConfigFile.Name(),
			},
			success: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfi, err := New()
			switch test.success {
			case true:
				if err != nil {
					t.Errorf("Expected test to pass, but it failed with error: %v", err)
				}
			case false:
				if err == nil {
					t.Errorf("Expected test to fail, but it passed!")
				}
			}
			if !reflect.DeepEqual(test.output, cfi) {
				t.Errorf("Expected output: %#v", test.output)
				t.Errorf("Actual output: %#v", cfi)
			}
		})
	}
}

func TestSetActiveComponent(t *testing.T) {
	tempConfigFile, err := ioutil.TempFile("", "ocdevconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer tempConfigFile.Close()
	os.Setenv(configEnvName, tempConfigFile.Name())

	tests := []struct {
		name           string
		existingConfig Config
		component      string
		project        string
		wantErr        bool
		result         []ApplicationInfo
	}{
		{
			name:           "activeComponents nil",
			existingConfig: Config{},
			component:      "foo",
			project:        "bar",
			wantErr:        true,
			result:         nil,
		},
		{
			name: "activeComponents empty",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:    "a",
						Active:  true,
						Project: "test",
					},
				},
			},
			component: "foo",
			project:   "test",
			wantErr:   false,
			result: []ApplicationInfo{
				ApplicationInfo{
					Name:            "a",
					Active:          true,
					Project:         "test",
					ActiveComponent: "foo",
				},
			},
		},
		{
			name: "no project active",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "a",
						Active:          false,
						Project:         "test",
						ActiveComponent: "b",
					},
				},
			},
			component: "foo",
			project:   "test",
			wantErr:   true,
			result:    nil,
		},
		{
			name: "overwrite existing active component (apps with same name in different projects)",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "a",
						Active:          true,
						Project:         "test",
						ActiveComponent: "old",
					},
					ApplicationInfo{
						Name:            "a",
						Active:          false,
						Project:         "test2",
						ActiveComponent: "old2",
					},
				},
			},
			component: "new",
			project:   "test",
			wantErr:   false,
			result: []ApplicationInfo{
				ApplicationInfo{
					Name:            "a",
					Active:          true,
					Project:         "test",
					ActiveComponent: "new",
				},
				ApplicationInfo{
					Name:            "a",
					Active:          false,
					Project:         "test2",
					ActiveComponent: "old2",
				},
			},
		},
		{
			name: "overwrite existing active component (different apps in the same project)",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "a",
						Active:          true,
						Project:         "test",
						ActiveComponent: "old",
					},
					ApplicationInfo{
						Name:            "b",
						Active:          false,
						Project:         "test",
						ActiveComponent: "old2",
					},
				},
			},
			component: "new",
			project:   "test",
			wantErr:   false,
			result: []ApplicationInfo{
				ApplicationInfo{
					Name:            "a",
					Active:          true,
					Project:         "test",
					ActiveComponent: "new",
				},
				ApplicationInfo{
					Name:            "b",
					Active:          false,
					Project:         "test",
					ActiveComponent: "old2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := New()
			if err != nil {
				t.Error(err)
			}
			cfg.Config = tt.existingConfig

			err = cfg.SetActiveComponent(tt.component, tt.project)
			if tt.wantErr {
				if (err != nil) != tt.wantErr {
					t.Errorf("SetActiveComponent() unexpected error %v, wantErr %v", err, tt.wantErr)
				}
			}
			if err == nil {
				if !reflect.DeepEqual(cfg.ActiveApplications, tt.result) {
					t.Errorf("expected output doesn't match what was returned: \n expected:\n%#v\n, returned:\n%#v\n", tt.result, cfg.ActiveApplications)
				}
			}

		})
	}
}

func TestGetActiveComponent(t *testing.T) {
	tempConfigFile, err := ioutil.TempFile("", "ocdevconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer tempConfigFile.Close()
	os.Setenv(configEnvName, tempConfigFile.Name())

	tests := []struct {
		name              string
		existingConfig    Config
		activeApplication string
		activeProject     string
		activeComponent   string
	}{
		{
			name:              "empty config",
			existingConfig:    Config{},
			activeApplication: "test",
			activeProject:     "test",
			activeComponent:   "",
		},
		{
			name: "ActiveApplications empty",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{},
			},
			activeApplication: "test",
			activeProject:     "test",
			activeComponent:   "",
		},
		{
			name: "no active component record for given application",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:    "a",
						Active:  false,
						Project: "test",
					},
				},
			},
			activeApplication: "test",
			activeProject:     "test",
			activeComponent:   "",
		},
		{
			name: "activeComponents for one project",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "a",
						Active:          true,
						Project:         "test",
						ActiveComponent: "b",
					},
				},
			},
			activeApplication: "a",
			activeProject:     "test",
			activeComponent:   "b",
		},
		{
			name: "inactive project",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "a",
						Active:          false,
						Project:         "test",
						ActiveComponent: "b",
					},
				},
			},
			activeApplication: "a",
			activeProject:     "test",
			activeComponent:   "",
		},
		{
			name: "multiple projects",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "a",
						Active:          false,
						Project:         "test",
						ActiveComponent: "b",
					},
					ApplicationInfo{
						Name:            "a",
						Active:          true,
						Project:         "test2",
						ActiveComponent: "b2",
					},
				},
			},
			activeApplication: "a",
			activeProject:     "test2",
			activeComponent:   "b2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfigInfo{
				Config: tt.existingConfig,
			}
			output := cfg.GetActiveComponent(tt.activeApplication, tt.activeProject)

			if output != tt.activeComponent {
				t.Errorf("active component doesn't match expected \ngot: %s \nexpected: %s\n", output, tt.activeComponent)
			}

		})
	}
}

func TestSetActiveApplication(t *testing.T) {
	tempConfigFile, err := ioutil.TempFile("", "ocdevconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer tempConfigFile.Close()
	os.Setenv(configEnvName, tempConfigFile.Name())

	tests := []struct {
		name           string
		existingConfig Config
		setApplication string
		project        string
	}{
		{
			name:           "activeApplication nil",
			existingConfig: Config{},
			setApplication: "app",
			project:        "proj",
		},
		{
			name: "activeApplication empty",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{},
			},
			setApplication: "app",
			project:        "proj",
		},
		{
			name: "no Active value",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "app",
						Project:         "proj",
						ActiveComponent: "b",
					},
				},
			},
			setApplication: "app",
			project:        "proj",
		},
		{
			name: "multiple apps in the same project",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "app",
						Active:          true,
						Project:         "proj",
						ActiveComponent: "b",
					},
					ApplicationInfo{
						Name:            "app2",
						Active:          false,
						Project:         "proj",
						ActiveComponent: "b2",
					},
				},
			},
			setApplication: "app2",
			project:        "proj",
		},
		{
			name: "same app name in different projects",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "app",
						Active:          true,
						Project:         "proj",
						ActiveComponent: "b",
					},
					ApplicationInfo{
						Name:            "app",
						Active:          false,
						Project:         "proj2",
						ActiveComponent: "b2",
					},
				},
			},
			setApplication: "app",
			project:        "proj2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := New()
			if err != nil {
				t.Error(err)
			}
			err = cfg.SetActiveApplication(tt.setApplication, tt.project)
			if err != nil {
				t.Error(err)
			}

			found := false
			for _, aa := range cfg.ActiveApplications {
				if aa.Project == tt.project && aa.Name == tt.setApplication {
					found = true
				}
			}
			if !found {
				t.Errorf("application %s/%s was not set as current", tt.project, tt.setApplication)
			}

		})
	}
}

func TestGetActiveApplication(t *testing.T) {

	tempConfigFile, err := ioutil.TempFile("", "ocdevconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer tempConfigFile.Close()
	os.Setenv(configEnvName, tempConfigFile.Name())

	tests := []struct {
		name              string
		existingConfig    Config
		activeProject     string
		activeApplication string
	}{
		{
			name:              "activeApplication nil",
			existingConfig:    Config{},
			activeApplication: "",
			activeProject:     "proj",
		},
		{
			name: "activeApplication empty",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{},
			},
			activeApplication: "",
			activeProject:     "proj",
		},
		{
			name: "no Active value",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "app",
						Project:         "proj",
						ActiveComponent: "b",
					},
				},
			},
			activeApplication: "",
			activeProject:     "proj",
		},
		{
			name: "multiple apps in the same project",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "app",
						Active:          true,
						Project:         "proj",
						ActiveComponent: "b",
					},
					ApplicationInfo{
						Name:            "app2",
						Active:          false,
						Project:         "proj",
						ActiveComponent: "b2",
					},
				},
			},
			activeApplication: "app",
			activeProject:     "proj",
		},
		{
			name: "same app name in different projects",
			existingConfig: Config{
				ActiveApplications: []ApplicationInfo{
					ApplicationInfo{
						Name:            "app",
						Active:          true,
						Project:         "proj",
						ActiveComponent: "b",
					},
					ApplicationInfo{
						Name:            "app",
						Active:          false,
						Project:         "proj2",
						ActiveComponent: "b2",
					},
				},
			},
			activeApplication: "app",
			activeProject:     "proj",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfigInfo{
				Config: tt.existingConfig,
			}
			output := cfg.GetActiveApplication(tt.activeProject)

			if output != tt.activeApplication {
				t.Errorf("active application doesn't match expected \ngot: %s \nexpected: %s\n", output, tt.activeApplication)
			}

		})
	}
}

//
//func TestGet(t *testing.T) {
//
//}
//
//func TestSet(t *testing.T) {
//
//}
//
//func TestApplicationExists(t *testing.T) {
//
//}
//
//func TestAddApplication(t *testing.T) {
//
//}
