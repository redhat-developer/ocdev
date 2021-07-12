package project

import (
	"errors"
	"fmt"
	"github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"path"
	"reflect"
	"strings"
)

func PreFlightCheck(ctx pipeline.Context) {
	ctx.SetCondition(v1alpha1.Conditions().CollectionReady().DataCollected().Build())
	applications, err := ctx.Applications()
	if err != nil {
		ctx.RetryProcessing(err)
		ctx.SetCondition(v1alpha1.Conditions().NotInjectionReady().ApplicationNotFound().Msg(err.Error()).Build())
		return
	}
	if len(applications) == 0 {
		ctx.SetCondition(v1alpha1.Conditions().NotInjectionReady().Reason(v1alpha1.EmptyApplicationReason).Build())
		ctx.StopProcessing()
	}
}

func PostFlightCheck(ctx pipeline.Context) {
	ctx.SetCondition(v1alpha1.Conditions().InjectionReady().Reason("ApplicationUpdated").Build())
}

func InjectSecretRef(ctx pipeline.Context) {
	applications, _ := ctx.Applications()
	for _, app := range applications {
		secretPath := app.SecretPath()
		if secretPath == "" {
			continue
		}
		err := unstructured.SetNestedField(app.Resource().Object, ctx.BindingSecretName(), strings.Split(secretPath, ".")...)
		if err != nil {
			stop(ctx, err)
			return
		}

	}
}

func BindingsAsEnv(ctx pipeline.Context) {
	if ctx.BindAsFiles() {
		return
	}
	applications, _ := ctx.Applications()
	envFromSecret := corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: ctx.BindingSecretName(),
			},
		},
	}
	for _, app := range applications {
		if app.SecretPath() != "" {
			continue
		}
		appResource := app.Resource()
		containerResources, found, err := resources(&corev1.Container{}, appResource.Object, strings.Split(app.ContainersPath(), ".")...)
		if !found {
			err = errors.New("Containers not found in app resource")
		}
		if err != nil {
			stop(ctx, err)
			return
		}
		for _, container := range containerResources {
			envFrom, found := container["envFrom"]
			if !found {

				u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&envFromSecret)
				if err != nil {
					stop(ctx, err)
					return
				}
				container["envFrom"] = []interface{}{u}
				continue
			}
			envFromSlice, ok := envFrom.([]interface{})
			if !ok {
				stop(ctx, errors.New("envFrom not a slice"))
				return
			}
			u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&envFromSecret)
			if err != nil {
				stop(ctx, err)
				return
			}
			container["envFrom"] = append(envFromSlice, u)
		}
	}
}

var volumesPath = []string{"spec", "template", "spec", "volumes"}

func BindingsAsFiles(ctx pipeline.Context) {
	if !ctx.BindAsFiles() {
		return
	}
	secretName := ctx.BindingSecretName()
	bindingName := ctx.BindingName()
	applications, _ := ctx.Applications()
	for _, app := range applications {
		if app.SecretPath() != "" {
			continue
		}
		appResource := app.Resource()
		volumerResources, found, err := resources(&corev1.Volume{}, appResource.Object, volumesPath...)
		if err != nil {
			stop(ctx, err)
			return
		}

		volume, err := runtime.DefaultUnstructuredConverter.ToUnstructured(
			&corev1.Volume{
				Name: bindingName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretName,
					},
				},
			})
		if err != nil {
			stop(ctx, err)
			return
		}
		if found {
			var newVolumes []interface{}
			exist := false
			for _, v := range volumerResources {
				if v["name"] == volume["name"] {
					exist = true
					if !reflect.DeepEqual(v["secret"], volume["secret"]) {
						newVolumes = append(newVolumes, volume)
					} else {
						newVolumes = append(newVolumes, v)
					}
				} else {
					newVolumes = append(newVolumes, v)
				}
			}
			if !exist {
				newVolumes = append(newVolumes, volume)
			}
			if err = unstructured.SetNestedSlice(appResource.Object, newVolumes, volumesPath...); err != nil {
				stop(ctx, err)
				return
			}
		} else {
			if err = unstructured.SetNestedSlice(appResource.Object, []interface{}{volume}, volumesPath...); err != nil {
				stop(ctx, err)
				return
			}
		}

		containerResources, found, err := resources(&corev1.Container{}, appResource.Object, strings.Split(app.ContainersPath(), ".")...)
		if !found {
			err = errors.New("Containers not found in app resource")
		}
		if err != nil {
			stop(ctx, err)
			return
		}
		for _, container := range containerResources {
			mountPath, err := mountPath(container, ctx)
			if err != nil {
				stop(ctx, err)
				return
			}
			volumeMounts, found, err := resources(&corev1.VolumeMount{}, container, "volumeMounts")
			if err != nil {
				stop(ctx, err)
				return
			}
			u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&corev1.VolumeMount{
				Name:      bindingName,
				MountPath: mountPath,
			})
			if err != nil {
				stop(ctx, err)
				return
			}
			if found {
				var newVolumeMounts []interface{}
				exist := false
				for _, vm := range volumeMounts {
					if vm["name"] == u["name"] {
						exist = true
						if !reflect.DeepEqual(vm, u) {
							newVolumeMounts = append(newVolumeMounts, u)
						} else {
							newVolumeMounts = append(newVolumeMounts, vm)
						}
					} else {
						newVolumeMounts = append(newVolumeMounts, vm)
					}
				}
				if !exist {
					newVolumeMounts = append(newVolumeMounts, u)
				}
				container["volumeMounts"] = newVolumeMounts
			} else {
				if err := unstructured.SetNestedField(container, []interface{}{u}, "volumeMounts"); err != nil {
					stop(ctx, err)
					return
				}
			}
		}

	}
}

func Unbind(ctx pipeline.Context) {
	if !ctx.UnbindRequested() {
		return
	}
	applications, err := ctx.Applications()
	if err != nil || len(applications) == 0 {
		ctx.StopProcessing()
		return
	}
	secretName := ctx.BindingSecretName()
	bindingName := ctx.BindingName()
	if secretName == "" {
		ctx.StopProcessing()
		return
	}
	for _, app := range applications {
		appResource := app.Resource()
		podSpec, found, err := unstructured.NestedFieldNoCopy(appResource.Object, volumesPath[:len(volumesPath)-1]...)
		if !found || err != nil {
			continue
		}
		podSpecMap, ok := podSpec.(map[string]interface{})
		if !ok {
			continue
		}
		volumeResources, found, _ := resources(&corev1.Volume{}, podSpecMap, "volumes")
		if found {
			for i, vol := range volumeResources {
				if val, found, err := unstructured.NestedString(vol, "name"); found && err == nil && val == bindingName {
					s := append(volumeResources[:i], volumeResources[i+1:]...)
					if len(s) == 0 {
						delete(podSpecMap, "volumes")
					} else {
						podSpecMap["volumes"] = s
					}
					break
				}
			}
		}
		containerResources, found, _ := resources(&corev1.Container{}, appResource.Object, strings.Split(app.ContainersPath(), ".")...)
		if !found {
			ctx.StopProcessing()
			return
		}
		for _, container := range containerResources {
			envFrom, found, _ := resources(&corev1.EnvFromSource{}, container, "envFrom")
			if found {
				for i, envSource := range envFrom {
					if val, found, err := unstructured.NestedString(envSource, "secretRef", "name"); found && err == nil && val == secretName {
						s := append(envFrom[:i], envFrom[i+1:]...)
						if len(s) == 0 {
							delete(container, "envFrom")
						} else {
							container["envFrom"] = s
						}
						break
					}
				}
			}
			volumeMounts, found, _ := resources(&corev1.VolumeMount{}, container, "volumeMounts")
			if found {
				for i, vm := range volumeMounts {
					if val, found, err := unstructured.NestedString(vm, "name"); found && err == nil && val == bindingName {
						s := append(volumeMounts[:i], volumeMounts[i+1:]...)
						if len(s) == 0 {
							delete(container, "volumeMounts")
						} else {
							container["volumeMounts"] = s
						}
						break
					}
				}
			}
		}
	}
	ctx.StopProcessing()
}

const bindingRootEnvVar = "SERVICE_BINDING_ROOT"

func mountPath(container map[string]interface{}, ctx pipeline.Context) (string, error) {
	envs, found, err := resources(&corev1.EnvVar{}, container, "env")
	if err != nil {
		return "", err
	}
	bindingRoot := ""
	if found {
		for _, e := range envs {
			if e["name"] == bindingRootEnvVar {
				bindingRoot = fmt.Sprintf("%v", e["value"])
				return path.Join(bindingRoot, ctx.BindingName()), nil
			}
		}
	}

	mp := ctx.MountPath()
	if mp == "" {
		bindingRoot = "/bindings"
		mp = path.Join(bindingRoot, ctx.BindingName())
	} else {
		return mp, nil
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&corev1.EnvVar{
		Name:  bindingRootEnvVar,
		Value: bindingRoot,
	})
	if err != nil {
		return "", err
	}
	envs = append(envs, u)
	if found {
		container["env"] = append(envs, u)
	} else {
		if err := unstructured.SetNestedField(container, []interface{}{u}, "env"); err != nil {
			return "", err
		}
	}
	return mp, nil

}

func stop(ctx pipeline.Context, err error) {
	ctx.StopProcessing()
	ctx.Error(err)
	ctx.SetCondition(v1alpha1.Conditions().NotInjectionReady().Reason("Error").Msg(err.Error()).Build())
}

func resources(obj interface{}, resource map[string]interface{}, path ...string) ([]map[string]interface{}, bool, error) {
	val, found, err := unstructured.NestedFieldNoCopy(resource, path...)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, found, nil
	}
	valSlice, ok := val.([]interface{})
	if !ok {
		return nil, true, errors.New("not a slice")
	}
	var containers []map[string]interface{}
	for _, item := range valSlice {
		u, ok := item.(map[string]interface{})
		if !ok {
			return nil, true, errors.New("not a map")
		}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, obj)
		if err != nil {
			return nil, true, err
		}
		containers = append(containers, u)
	}
	return containers, true, nil
}
