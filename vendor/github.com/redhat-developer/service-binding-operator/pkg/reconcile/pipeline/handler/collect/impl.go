package collect

import (
	"encoding/base64"
	"errors"
	"fmt"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/pkg/binding"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
	"github.com/redhat-developer/service-binding-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"strings"
)

var DataNotMap = errors.New("Returned data are not a map, skip collecting")

const (
	ErrorReadingServicesReason   = "ErrorReadingServices"
	ErrorReadingCRD              = "ErrorReadingCRD"
	ErrorReadingDescriptorReason = "ErrorReadingDescriptor"
	ErrorReadingBindingReason    = "ErrorReadingBinding"
	ErrorReadingSecret           = "ErrorReadingSecret"
)

func PreFlight(ctx pipeline.Context) {
	_, err := ctx.Services()
	if err != nil {
		requestRetry(ctx, ErrorReadingServicesReason, err)
		return
	}
}

func BindingDefinitions(ctx pipeline.Context) {
	services, _ := ctx.Services()

	for _, service := range services {
		anns := make(map[string]string)
		crd, err := service.CustomResourceDefinition()
		if err != nil {
			requestRetry(ctx, ErrorReadingCRD, err)
			return
		}
		if crd != nil {
			descr, err := crd.Descriptor()
			if err != nil {
				requestRetry(ctx, ErrorReadingDescriptorReason, err)
				return
			}
			if descr != nil {
				util.MergeMaps(anns, bindingAnnotations(descr))
			}
			util.MergeMaps(anns, crd.Resource().GetAnnotations())
		}

		util.MergeMaps(anns, service.Resource().GetAnnotations())

		for k, v := range anns {
			definition, err := makeBindingDefinition(k, v, ctx)
			if err != nil {
				continue
			}
			service.AddBindingDef(definition)
		}
	}
}

func BindingItems(ctx pipeline.Context) {
	services, _ := ctx.Services()

	for _, service := range services {
		serviceResource := service.Resource()
		for _, bd := range service.BindingDefs() {
			bindingValue, err := bd.Apply(serviceResource)
			if err != nil {
				requestRetry(ctx, ErrorReadingBindingReason, err)
				return
			}
			val := bindingValue.Get()
			v := reflect.ValueOf(val)
			if v.Kind() != reflect.Map {
				requestRetry(ctx, "DataNotMap", DataNotMap)
				return
			}
			for _, n := range v.MapKeys() {
				collectItems("", ctx, service, n, v.MapIndex(n).Interface())
			}
		}
	}
}

const ProvisionedServiceAnnotationKey = "service.binding/provisioned-service"

func ProvisionedService(ctx pipeline.Context) {
	services, _ := ctx.Services()

	for _, service := range services {
		res := service.Resource()
		secretName, found, err := unstructured.NestedString(res.Object, "status", "binding", "name")
		if err != nil {
			requestRetry(ctx, ErrorReadingBindingReason, err)
			return
		}
		if found {
			if secretName != "" {
				secret, err := ctx.ReadSecret(res.GetNamespace(), secretName)
				if err != nil {
					requestRetry(ctx, ErrorReadingSecret, err)
					return
				}
				ctx.AddBindings(&pipeline.SecretBackedBindings{Service: service, Secret: secret})
			}
		} else {
			crd, err := service.CustomResourceDefinition()
			if err != nil {
				requestRetry(ctx, ErrorReadingCRD, err)
				return
			}
			if crd == nil {
				continue
			}
			v, ok := crd.Resource().GetAnnotations()[ProvisionedServiceAnnotationKey]
			if ok && v == "true" {
				requestRetry(ctx, ErrorReadingBindingReason, fmt.Errorf("CRD of service %v/%v indicates provisioned service, but no secret name provided under .status.binding.name", res.GetNamespace(), res.GetName()))
				return
			}
		}
	}
}

type pathMapping struct {
	input     string
	transform func(interface{}) (interface{}, error)
	output    string
}

var bindableResources = map[schema.GroupVersionKind]pathMapping{
	schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}: {
		input:  "data",
		output: "",
	},
	schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}: {
		input: "data",
		transform: func(i interface{}) (interface{}, error) {
			v := reflect.ValueOf(i)
			if v.Kind() != reflect.Map {
				return nil, errors.New("data is not map")
			}
			result := map[string]string{}

			for _, n := range v.MapKeys() {
				b, err := base64.StdEncoding.DecodeString(fmt.Sprintf("%v", v.MapIndex(n).Interface()))
				if err != nil {
					return nil, err
				}
				key := fmt.Sprintf("%v", n.Interface())
				result[key] = string(b)
			}
			return result, nil
		},
		output: "",
	},
	schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}: {
		input:  "spec.clusterIP",
		output: "clusterIP",
	},
	schema.GroupVersionKind{
		Group:   "route.openshift.io",
		Version: "v1",
		Kind:    "Route",
	}: {
		input:  "spec.host",
		output: "host",
	},
}

func OwnedResources(ctx pipeline.Context) {
	services, err := ctx.Services()
	if err != nil {
		requestRetry(ctx, ErrorReadingServicesReason, err)
		return
	}
	for _, service := range services {
		ownedResources, err := service.OwnedResources()
		if err != nil {
			requestRetry(ctx, ErrorReadingServicesReason, err)
			return
		}
		for _, res := range ownedResources {
			pathMapping, ok := bindableResources[res.GroupVersionKind()]
			if !ok {
				continue
			}
			val, found, err := unstructured.NestedFieldNoCopy(res.Object, strings.Split(pathMapping.input, ".")...)
			if !found {
				err = errors.New("Not found")
			}
			if err != nil {
				requestRetry(ctx, ErrorReadingServicesReason, err)
				return
			}
			if pathMapping.transform != nil {
				val, err = pathMapping.transform(val)
				if err != nil {
					requestRetry(ctx, ErrorReadingServicesReason, err)
					return
				}
			}
			collectItems("", ctx, service, reflect.ValueOf(pathMapping.output), val)
		}
	}
}

func collectItems(prefix string, ctx pipeline.Context, service pipeline.Service, k reflect.Value, val interface{}) {
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Map:
		p := prefix + k.String() + "_"
		if p == "_" {
			p = ""
		}
		for _, n := range v.MapKeys() {
			collectItems(p, ctx, service, n, v.MapIndex(n).Interface())
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			ctx.AddBindingItem(&pipeline.BindingItem{Name: fmt.Sprintf("%v_%v", prefix+k.String(), i), Value: v.Index(i).Interface(), Source: service})
		}
	default:
		ctx.AddBindingItem(&pipeline.BindingItem{Name: prefix + k.String(), Value: v.Interface(), Source: service})
	}
}

func requestRetry(ctx pipeline.Context, reason string, err error) {
	ctx.RetryProcessing(err)
	ctx.SetCondition(notCollectionReadyCond(reason, err))
}

func notCollectionReadyCond(reason string, err error) *metav1.Condition {
	return v1alpha1.Conditions().NotCollectionReady().Reason(reason).Msg(err.Error()).Build()
}

func makeBindingDefinition(key string, value string, ctx pipeline.Context) (binding.Definition, error) {
	return binding.NewDefinitionBuilder(key,
		value,
		func(namespace string, name string) (*unstructured.Unstructured, error) {
			return ctx.ReadConfigMap(namespace, name)
		},
		func(namespace string, name string) (*unstructured.Unstructured, error) {
			return ctx.ReadSecret(namespace, name)
		}).Build()
}

func bindingAnnotations(crdDescription *olmv1alpha1.CRDDescription) map[string]string {
	anns := make(map[string]string)
	for _, sd := range crdDescription.StatusDescriptors {
		objectType := getObjectType(sd.XDescriptors)
		for _, xd := range sd.XDescriptors {
			loadDescriptor(anns, sd.Path, xd, "status", objectType)
		}
	}

	for _, sd := range crdDescription.SpecDescriptors {
		objectType := getObjectType(sd.XDescriptors)
		for _, xd := range sd.XDescriptors {
			loadDescriptor(anns, sd.Path, xd, "spec", objectType)
		}
	}
	return anns
}

func getObjectType(descriptors []string) string {
	typeAnno := "urn:alm:descriptor:io.kubernetes:"
	for _, desc := range descriptors {
		if strings.HasPrefix(desc, typeAnno) {
			return strings.TrimPrefix(desc, typeAnno)
		}
	}
	return ""
}

func loadDescriptor(anns map[string]string, path string, descriptor string, root string, objectType string) {
	if !strings.HasPrefix(descriptor, binding.AnnotationPrefix) {
		return
	}

	keys := strings.Split(descriptor, ":")
	key := binding.AnnotationPrefix
	value := ""

	if len(keys) > 1 {
		key += "/" + keys[1]
	} else {
		key += "/" + path
	}

	p := []string{fmt.Sprintf("path={.%s.%s}", root, path)}
	if len(keys) > 1 {
		p = append(p, keys[2:]...)
	}
	if objectType != "" {
		p = append(p, []string{fmt.Sprintf("objectType=%s", objectType)}...)
	}

	value += strings.Join(p, ",")
	anns[key] = value
}
