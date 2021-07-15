package convert

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/devfile/library/pkg/devfile/parser"
	"github.com/devfile/library/pkg/devfile/parser/data"
	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/envinfo"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/pkg/errors"
	"k8s.io/klog"

	imagev1 "github.com/openshift/api/image/v1"

	devfilev1 "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
	devfilepkg "github.com/devfile/api/v2/pkg/devfile"
	devfileCtx "github.com/devfile/library/pkg/devfile/parser/context"
)

const (
	buildCommandID = "s2i-assemble"
	// build command to be used in s2i devfile
	buildCommandS2i = "/opt/odo/bin/s2i-setup && /opt/odo/bin/assemble-and-restart"
	// run command id to be used in devfile
	runCommandID = "s2i-run"
	// run command to be used in s2i devfile
	runCommandS2i = "/opt/odo/bin/run"
	// debug command id to be used in devfile
	debugCommandID = "s2i-debug"
	// ContainerName is component name to be used in devfile
	ContainerName = "s2i-builder"
	// DefaultSourceMappingS2i is the directory to sync s2i source code
	DefaultSourceMappingS2i = "/tmp/projects"
	// devfile version
	devfileVersion = "2.0.0"
	// environment variable set for s2i assemble and restart scripts
	// some change in script if scripts is executed for a s2i component converted to devfile
	envS2iConvertedDevfile = "ODO_S2I_CONVERTED_DEVFILE"
)

// GenerateDevfileYaml generates a devfile.yaml from s2i data.
func GenerateDevfileYaml(client *occlient.Client, co *config.LocalConfigInfo, context string) error {
	klog.V(2).Info("Generating devfile.yaml")

	// builder image to use
	componentType := co.GetType()
	// git, local, binary, none
	sourceType := co.GetSourceType()

	debugPort := co.GetDebugPort()

	imageStream, imageforDevfile, err := getImageforDevfile(client, componentType)
	if err != nil {
		return errors.Wrap(err, "Failed to get image details")
	}

	envVarList := co.GetEnvVars()

	if debugPort != 0 || debugPort == config.DefaultDebugPort {
		env := config.EnvVar{
			Name:  common.EnvDebugPort,
			Value: fmt.Sprint(debugPort),
		}
		envVarList = append(envVarList, env)
	}

	s2iEnv, err := occlient.GetS2IEnvForDevfile(string(sourceType), envVarList, *imageStream)
	if err != nil {
		return err
	}

	s2iDevfile, err := data.NewDevfileData(devfileVersion)
	if err != nil {
		return err
	}

	// set schema version
	s2iDevfile.SetSchemaVersion(devfileVersion)

	// set metadata
	s2iDevfile.SetMetadata(devfilepkg.DevfileMetadata{
		Name:     co.GetName(),
		Version:  "1.0.0",
		Language: componentType,
	})
	// set commponents
	err = setDevfileComponentsForS2I(s2iDevfile, imageforDevfile, co, s2iEnv)
	if err != nil {
		return err
	}
	// set commands
	setDevfileCommandsForS2I(s2iDevfile)

	ctx := devfileCtx.NewDevfileCtx(filepath.Join(context, "devfile.yaml"))
	err = ctx.SetAbsPath()
	if err != nil {
		return err
	}

	devObj := parser.DevfileObj{
		Ctx:  ctx,
		Data: s2iDevfile,
	}

	err = devObj.WriteYamlDevfile()
	klog.V(2).Info("Generated devfile.yaml successfully")

	if err != nil {
		return err
	}

	return nil
}

// GenerateEnvYaml generates .odo/env.yaml from s2i data.
func GenerateEnvYaml(client *occlient.Client, co *config.LocalConfigInfo, context string) (*envinfo.EnvSpecificInfo, error) {
	klog.V(2).Info("Generating env.yaml")

	debugPort := co.GetDebugPort()

	application := co.GetApplication()

	// Generate env.yaml
	envSpecificInfo, err := envinfo.NewEnvSpecificInfo(context)
	if err != nil {
		return nil, err
	}

	componentSettings := envinfo.ComponentSettings{
		Name:    co.GetName(),
		Project: co.GetProject(),
		AppName: application,
	}

	if debugPort != 0 || debugPort == config.DefaultDebugPort {
		componentSettings.DebugPort = &debugPort
	}

	err = envSpecificInfo.SetComponentSettings(componentSettings)
	if err != nil {
		return nil, err
	}
	klog.V(2).Info("Generated env.yaml successfully")

	return envSpecificInfo, nil
}

// getImageforDevfile gets image details from s2i component type.
func getImageforDevfile(client *occlient.Client, componentType string) (*imagev1.ImageStreamImage, string, error) {
	klog.V(2).Info("Getting container image details")

	imageNS, imageName, imageTag, _, err := occlient.ParseImageName(componentType)
	if err != nil {
		return nil, "", err
	}
	imageStream, err := client.GetImageStream(imageNS, imageName, imageTag)
	if err != nil {
		return nil, "", err
	}

	imageStreamImage, err := client.GetImageStreamImage(imageStream, imageTag)
	if err != nil {
		return nil, "", err
	}

	imageforDevfile := imageStream.Spec.Tags[0].From.Name

	return imageStreamImage, imageforDevfile, nil
}

// setDevfileCommandsForS2I sets command in devfile.yaml from s2i data.
func setDevfileCommandsForS2I(d data.DevfileData) {
	klog.V(2).Info("Set devfile commands from s2i data")

	buildCommand := devfilev1.Command{
		Id: buildCommandID,
		CommandUnion: devfilev1.CommandUnion{
			Exec: &devfilev1.ExecCommand{
				Component:   ContainerName,
				CommandLine: buildCommandS2i,
				LabeledCommand: devfilev1.LabeledCommand{
					BaseCommand: devfilev1.BaseCommand{
						Group: &devfilev1.CommandGroup{
							Kind:      devfilev1.BuildCommandGroupKind,
							IsDefault: true,
						},
					},
				},
			},
		},
	}

	runCommand := devfilev1.Command{
		Id: runCommandID,
		CommandUnion: devfilev1.CommandUnion{
			Exec: &devfilev1.ExecCommand{
				Component:   ContainerName,
				CommandLine: runCommandS2i,
				LabeledCommand: devfilev1.LabeledCommand{
					BaseCommand: devfilev1.BaseCommand{
						Group: &devfilev1.CommandGroup{
							Kind:      devfilev1.RunCommandGroupKind,
							IsDefault: true,
						},
					},
				},
			},
		},
	}

	debugCommand := devfilev1.Command{
		Id: debugCommandID,
		CommandUnion: devfilev1.CommandUnion{
			Exec: &devfilev1.ExecCommand{
				Component:   ContainerName,
				CommandLine: runCommandS2i,
				LabeledCommand: devfilev1.LabeledCommand{
					BaseCommand: devfilev1.BaseCommand{
						Group: &devfilev1.CommandGroup{
							Kind:      devfilev1.DebugCommandGroupKind,
							IsDefault: true,
						},
					},
				},
			},
		},
	}
	// Ignoring error as we are writing new file
	_ = d.AddCommands([]devfilev1.Command{buildCommand, runCommand, debugCommand})

}

// setDevfileComponentsForS2I sets the devfile.yaml components field from s2i data.
func setDevfileComponentsForS2I(d data.DevfileData, s2iImage string, localConfig *config.LocalConfigInfo, s2iEnv config.EnvVarList) error {
	klog.V(2).Info("Set devfile components from s2i data")

	maxMemory := localConfig.GetMaxMemory()
	volumes, err := localConfig.ListStorage()
	if err != nil {
		return err
	}
	urls, err := localConfig.ListURLs()
	if err != nil {
		return err
	}
	mountSources := true

	var endpoints []devfilev1.Endpoint
	var envs []devfilev1.EnvVar
	var volumeMounts []devfilev1.VolumeMount
	var components []devfilev1.Component

	// convert s2i storage to devfile volumes
	for _, vol := range volumes {
		volume := devfilev1.Component{
			Name: vol.Name,
			ComponentUnion: devfilev1.ComponentUnion{
				Volume: &devfilev1.VolumeComponent{
					Volume: devfilev1.Volume{
						Size: vol.Size,
					},
				},
			},
		}
		components = append(components, volume)

		volumeMount := devfilev1.VolumeMount{
			Name: vol.Name,
			Path: vol.Path,
		}

		volumeMounts = append(volumeMounts, volumeMount)
	}

	sourceMapping := DefaultSourceMappingS2i

	// Add s2i specific env variable in devfile
	for _, env := range s2iEnv {
		env := devfilev1.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		}
		envs = append(envs, env)
	}
	env := devfilev1.EnvVar{
		Name:  envS2iConvertedDevfile,
		Value: "true",
	}
	envs = append(envs, env)

	// convert s2i urls to devfile endpoints
	for _, url := range urls {

		endpoint := devfilev1.Endpoint{
			Name:       url.Name,
			TargetPort: url.Port,
			Secure:     url.Secure,
		}

		endpoints = append(endpoints, endpoint)
	}

	ports, err := localConfig.GetComponentPorts()
	if err != nil {
		return err
	}
	// convert s2i ports to devfile endpoints if there are no urls present
	for _, port := range ports {
		splitPort := strings.Split(port, "/")
		protocol := "http"
		//  protocol provided
		if len(splitPort) > 1 {
			// technically tcp when exposed is by default using http
			if strings.ToLower(splitPort[1]) == "tcp" {
				protocol = "http"
			} else {
				protocol = strings.ToLower(splitPort[1])
			}
		}
		intPort, err := strconv.Atoi(splitPort[0])
		// we dont fail if the ports are malformed
		if err != nil {
			continue
		}
		hasURL := false
		for _, url := range urls {
			if intPort == url.Port {
				hasURL = true
			}
		}

		if !hasURL {
			// every port is an exposed url for now
			endpoint := devfilev1.Endpoint{
				Name:       fmt.Sprintf("%s-%d", protocol, intPort),
				TargetPort: intPort,
				Secure:     false,
			}

			endpoints = append(endpoints, endpoint)
		}

	}

	container := devfilev1.Component{
		Name: ContainerName,
		ComponentUnion: devfilev1.ComponentUnion{
			Container: &devfilev1.ContainerComponent{
				Container: devfilev1.Container{
					Image:         s2iImage,
					MountSources:  &mountSources,
					SourceMapping: sourceMapping,
					MemoryLimit:   maxMemory,
					Env:           envs,
					VolumeMounts:  volumeMounts,
				},
				Endpoints: endpoints,
			},
		},
	}

	components = append(components, container)

	// Ignoring error here as we are writing a new file
	_ = d.AddComponents(components)

	return nil

}
