package parser

import (
	"github.com/ghodss/yaml"
	"github.com/kylelemons/godebug/pretty"
	parser "github.com/openshift/odo/pkg/devfile/parser/context"
	v200 "github.com/openshift/odo/pkg/devfile/parser/data/2.0.0"
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const schemaV200 = "2.0.0"

func Test_parseParent(t *testing.T) {
	type args struct {
		devFileObj DevfileObj
	}
	tests := []struct {
		name          string
		args          args
		parentDevFile DevfileObj
		wantDevFile   DevfileObj
		wantErr       bool
	}{
		{
			name: "case 1: it should override the requested parent's data and add the local devfile's data",
			args: args{
				devFileObj: DevfileObj{
					Ctx: parser.NewDevfileCtx(devfileTempPath),
					Data: &v200.Devfile200{
						Parent: common.DevfileParent{
							Commands: []common.DevfileCommand{
								{
									Exec: &common.Exec{
										Id:         "devrun",
										WorkingDir: "/projects/nodejs-starter",
									},
								},
							},
							Components: []common.DevfileComponent{
								{
									Container: &common.Container{
										Image: "quay.io/nodejs-12",
										Name:  "nodejs",
									},
								},
							},
							Events: common.DevfileEvents{
								PostStart: []string{"post-start-0-override"},
							},
							Projects: []common.DevfileProject{
								{
									ClonePath: "/projects",
									Name:      "nodejs-starter",
								},
							},
						},

						Commands: []common.DevfileCommand{
							{
								Exec: &common.Exec{
									Id:         "devbuild",
									WorkingDir: "/projects/nodejs-starter",
								},
							},
						},
						Components: []common.DevfileComponent{
							{
								Container: &common.Container{
									Image: "quay.io/nodejs-12",
									Name:  "runtime",
								},
							},
						},
						Events: common.DevfileEvents{
							PostStop: []string{"post-stop"},
						},
						Projects: []common.DevfileProject{
							{
								ClonePath: "/projects",
								Name:      "nodejs-starter-build",
							},
						},
					},
				},
			},
			parentDevFile: DevfileObj{
				Data: &v200.Devfile200{
					SchemaVersion: schemaV200,
					Commands: []common.DevfileCommand{
						{
							Exec: &common.Exec{
								Id:          "devrun",
								WorkingDir:  "/projects",
								CommandLine: "npm run",
							},
						},
					},
					Components: []common.DevfileComponent{
						{
							Container: &common.Container{
								Image: "quay.io/nodejs-10",
								Name:  "nodejs",
							},
						},
					},
					Events: common.DevfileEvents{
						PostStart: []string{"post-start-0"},
					},
					Projects: []common.DevfileProject{
						{
							ClonePath: "/data",
							Github: &common.Github{
								Branch: "master",
							},
							Name: "nodejs-starter",
						},
					},
				},
			},
			wantDevFile: DevfileObj{
				Data: &v200.Devfile200{
					Commands: []common.DevfileCommand{
						{
							Exec: &common.Exec{
								Id:         "devbuild",
								WorkingDir: "/projects/nodejs-starter",
							},
						},
						{
							Exec: &common.Exec{
								Id:          "devrun",
								CommandLine: "npm run",
								WorkingDir:  "/projects/nodejs-starter",
							},
						},
					},
					Components: []common.DevfileComponent{
						{
							Container: &common.Container{
								Image: "quay.io/nodejs-12",
								Name:  "runtime",
							},
						},
						{
							Container: &common.Container{
								Image: "quay.io/nodejs-12",
								Name:  "nodejs",
							},
						},
					},
					Events: common.DevfileEvents{
						PostStart: []string{"post-start-0-override"},
						PostStop:  []string{"post-stop"},
					},
					Projects: []common.DevfileProject{
						{
							ClonePath: "/projects",
							Name:      "nodejs-starter-build",
						},
						{
							ClonePath: "/projects",
							Github: &common.Github{
								Branch: "master",
							},
							Name: "nodejs-starter",
						},
					},
				},
			},
		},
		{
			name: "case 2: handle a parent'data without any local override and add the local devfile's data",
			args: args{
				devFileObj: DevfileObj{
					Ctx: parser.NewDevfileCtx(devfileTempPath),
					Data: &v200.Devfile200{
						Commands: []common.DevfileCommand{
							{
								Exec: &common.Exec{
									Id:         "devbuild",
									WorkingDir: "/projects/nodejs-starter",
								},
							},
						},
						Components: []common.DevfileComponent{
							{
								Container: &common.Container{
									Image: "quay.io/nodejs-12",
									Name:  "runtime",
								},
							},
						},
						Events: common.DevfileEvents{
							PostStop: []string{"post-stop"},
						},
						Projects: []common.DevfileProject{
							{
								ClonePath: "/projects",
								Name:      "nodejs-starter-build",
							},
						},
					},
				},
			},
			parentDevFile: DevfileObj{
				Data: &v200.Devfile200{
					SchemaVersion: schemaV200,
					Commands: []common.DevfileCommand{
						{
							Exec: &common.Exec{
								Id:          "devrun",
								WorkingDir:  "/projects",
								CommandLine: "npm run",
							},
						},
					},
					Components: []common.DevfileComponent{
						{
							Container: &common.Container{
								Image: "quay.io/nodejs-10",
								Name:  "nodejs",
							},
						},
					},
					Events: common.DevfileEvents{
						PostStart: []string{"post-start-0"},
					},
					Projects: []common.DevfileProject{
						{
							ClonePath: "/data",
							Github: &common.Github{
								Branch: "master",
							},
							Name: "nodejs-starter",
						},
					},
				},
			},
			wantDevFile: DevfileObj{
				Data: &v200.Devfile200{
					Commands: []common.DevfileCommand{
						{
							Exec: &common.Exec{
								Id:         "devbuild",
								WorkingDir: "/projects/nodejs-starter",
							},
						},
						{
							Exec: &common.Exec{
								Id:          "devrun",
								CommandLine: "npm run",
								WorkingDir:  "/projects",
							},
						},
					},
					Components: []common.DevfileComponent{
						{
							Container: &common.Container{
								Image: "quay.io/nodejs-12",
								Name:  "runtime",
							},
						},
						{
							Container: &common.Container{
								Image: "quay.io/nodejs-10",
								Name:  "nodejs",
							},
						},
					},
					Events: common.DevfileEvents{
						PostStart: []string{"post-start-0"},
						PostStop:  []string{"post-stop"},
					},
					Projects: []common.DevfileProject{
						{
							ClonePath: "/projects",
							Name:      "nodejs-starter-build",
						},
						{
							ClonePath: "/data",
							Github: &common.Github{
								Branch: "master",
							},
							Name: "nodejs-starter",
						},
					},
				},
			},
		},
		{
			name: "case 3: it should error out when the override is invalid",
			args: args{
				devFileObj: DevfileObj{
					Ctx: parser.NewDevfileCtx(devfileTempPath),
					Data: &v200.Devfile200{
						Parent: common.DevfileParent{
							Commands: []common.DevfileCommand{
								{
									Exec: &common.Exec{
										Id:         "devrun",
										WorkingDir: "/projects/nodejs-starter",
									},
								},
							},
							Components: []common.DevfileComponent{
								{
									Container: &common.Container{
										Image: "quay.io/nodejs-12",
										Name:  "nodejs",
									},
								},
							},
							Events: common.DevfileEvents{
								PostStart: []string{"post-start-0-override"},
							},
							Projects: []common.DevfileProject{
								{
									ClonePath: "/projects",
									Name:      "nodejs-starter",
								},
							},
						},
					},
				},
			},
			parentDevFile: DevfileObj{
				Data: &v200.Devfile200{
					SchemaVersion: schemaV200,
					Commands:      []common.DevfileCommand{},
					Components:    []common.DevfileComponent{},
					Projects:      []common.DevfileProject{},
				},
			},
			wantDevFile: DevfileObj{
				Data: &v200.Devfile200{},
			},
			wantErr: true,
		},
		{
			name: "case 4: error out if the same parent command is defined again in the local devfile",
			args: args{
				devFileObj: DevfileObj{
					Ctx: parser.NewDevfileCtx(devfileTempPath),
					Data: &v200.Devfile200{
						Commands: []common.DevfileCommand{
							{
								Exec: &common.Exec{
									Id:         "devbuild",
									WorkingDir: "/projects/nodejs-starter",
								},
							},
						},
					},
				},
			},
			parentDevFile: DevfileObj{
				Data: &v200.Devfile200{
					SchemaVersion: schemaV200,
					Commands: []common.DevfileCommand{
						{
							Exec: &common.Exec{
								Id:         "devbuild",
								WorkingDir: "/projects/nodejs-starter",
							},
						},
					},
				},
			},
			wantDevFile: DevfileObj{
				Data: &v200.Devfile200{},
			},
			wantErr: true,
		},
		{
			name: "case 5: error out if the same parent component is defined again in the local devfile",
			args: args{
				devFileObj: DevfileObj{
					Ctx: parser.NewDevfileCtx(devfileTempPath),
					Data: &v200.Devfile200{
						Components: []common.DevfileComponent{
							{
								Container: &common.Container{
									Image: "quay.io/nodejs-12",
									Name:  "runtime",
								},
							},
						},
					},
				},
			},
			parentDevFile: DevfileObj{
				Data: &v200.Devfile200{
					SchemaVersion: schemaV200,
					Components: []common.DevfileComponent{
						{
							Container: &common.Container{
								Image: "quay.io/nodejs-12",
								Name:  "runtime",
							},
						},
					},
				},
			},
			wantDevFile: DevfileObj{
				Data: &v200.Devfile200{},
			},
			wantErr: true,
		},
		{
			name: "case 6: error out if the same event is defined again in the local devfile",
			args: args{
				devFileObj: DevfileObj{
					Ctx: parser.NewDevfileCtx(devfileTempPath),
					Data: &v200.Devfile200{
						Events: common.DevfileEvents{
							PostStop: []string{"post-stop"},
						},
					},
				},
			},
			parentDevFile: DevfileObj{
				Data: &v200.Devfile200{
					SchemaVersion: schemaV200,
					Events: common.DevfileEvents{
						PostStop: []string{"post-stop"},
					},
				},
			},
			wantDevFile: DevfileObj{
				Data: &v200.Devfile200{},
			},
			wantErr: true,
		},
		{
			name: "case 7: error out if the same project is defined again in the local devfile",
			args: args{
				devFileObj: DevfileObj{
					Ctx: parser.NewDevfileCtx(devfileTempPath),
					Data: &v200.Devfile200{
						Projects: []common.DevfileProject{
							{
								ClonePath: "/projects",
								Name:      "nodejs-starter-build",
							},
						},
					},
				},
			},
			parentDevFile: DevfileObj{
				Data: &v200.Devfile200{
					SchemaVersion: schemaV200,
					Projects: []common.DevfileProject{
						{
							ClonePath: "/projects",
							Name:      "nodejs-starter-build",
						},
					},
				},
			},
			wantDevFile: DevfileObj{
				Data: &v200.Devfile200{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := yaml.Marshal(tt.parentDevFile.Data)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				_, err = w.Write(data)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}))

			defer testServer.Close()

			parent := tt.args.devFileObj.Data.GetParent()
			parent.Uri = testServer.URL

			tt.args.devFileObj.Data.SetParent(parent)
			tt.wantDevFile.Data.SetParent(parent)

			err := parseParent(tt.args.devFileObj)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseParent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil {
				return
			}

			if !reflect.DeepEqual(tt.args.devFileObj.Data, tt.wantDevFile.Data) {
				t.Errorf("wanted: %v, got: %v, difference at %v", tt.wantDevFile, tt.args.devFileObj, pretty.Compare(tt.args.devFileObj.Data, tt.wantDevFile.Data))
			}
		})
	}
}
