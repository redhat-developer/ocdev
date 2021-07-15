package pipeline

import (
	"fmt"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	api "github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/pkg/binding"
	"github.com/redhat-developer/service-binding-operator/pkg/client/kubernetes"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:generate mockgen -destination=mocks/mocks_pipeline.go -package=mocks . Context,Service,CRD,Application,ContextProvider,Handler

// Reconciliation pipeline
type Pipeline interface {

	// Reconcile given service binding
	// Returns true if processing should be repeated
	// and optional error if occurred
	// important: even if error occurred it might not be needed to retry processing
	Process(binding *api.ServiceBinding) (bool, error)
}

// A pipeline stage
type Handler interface {
	Handle(ctx Context)
}

// Pipeline flow control
type FlowStatus struct {
	Retry bool
	Stop  bool
	Err   error
}

type HasResource interface {
	Resource() *unstructured.Unstructured
}

// Service to be bound
type Service interface {

	// Service resource
	HasResource

	// Return CRD for this service, otherwise nil if not backed by CRD
	// Error might be returned if occurred during the operation
	CustomResourceDefinition() (CRD, error)

	// Resources owned by the service, if any
	// Error might be returned if occurred during the operation
	OwnedResources() ([]*unstructured.Unstructured, error)

	// Attach binding definition to service
	AddBindingDef(def binding.Definition)

	// All binding definitions attached to the service
	BindingDefs() []binding.Definition

	// Optional service id
	Id() *string
}

// Application to be bound to service(s)
type Application interface {

	// Application resource
	HasResource

	// dot-separated path inside the application resource locating container resources
	// the returned value follows foo.bar.bla convention
	// it cannot be empty
	ContainersPath() string

	// optional dot-separated path inside the application resource locating field where intermediate binding secret ref should be injected
	// the returns value follows foo.bar.bla convention, but it can be empty
	SecretPath() string
}

// Custom Resource Definition
type CRD interface {

	// CRD resource
	HasResource

	// optional Descriptor attached to ClusterServiceVersion resource
	Descriptor() (*olmv1alpha1.CRDDescription, error)
}

// Pipeline context passed to each handler
type Context interface {
	BindingName() string

	// Services referred by binding
	// if reading fails, return error
	Services() ([]Service, error)

	// Applications referred by binding
	// it can be empty slice
	// if reading fails, return error
	Applications() ([]Application, error)

	// Returns true if binding is about to be removed
	UnbindRequested() bool

	BindingSecretName() string

	// Return true if bindings should be projected as files inside application containers
	BindAsFiles() bool

	// Path where bindings should be mounted inside application containers
	MountPath() string

	// Template that should be applied on collected binding names, prior projection
	NamingTemplate() string

	// Additional bindings that will be projected into application containers
	// entry key is the future binding name
	// entry value contains template that generates binding value
	Mappings() map[string]string

	// Add binding item to the context
	AddBindingItem(item *BindingItem)

	// Add bindings to the context
	AddBindings(bindings Bindings)

	// List binding items that should be projected into application containers
	BindingItems() BindingItems

	// Indicates that the binding should be retried at some later time
	// The current processing stops and context gets closed
	RetryProcessing(reason error)

	// Indicates that en error has occurred while processing the binding
	Error(err error)

	// Stops processing
	StopProcessing()

	// Closes the context, persisting changed resources
	// Returns error if occurrs
	Close() error

	// Sets context condition
	SetCondition(condition *metav1.Condition)

	kubernetes.ConfigMapReader
	kubernetes.SecretReader

	FlowStatus() FlowStatus
}

// Provides context for a given service binding
type ContextProvider interface {
	Get(binding *api.ServiceBinding) (Context, error)
}

type HandlerFunc func(ctx Context)

func (f HandlerFunc) Handle(ctx Context) {
	f(ctx)
}

type BindingItems []*BindingItem

type BindingItem struct {
	Name   string
	Value  interface{}
	Source Service
}

// a collection of bindings
type Bindings interface {
	// available bindgins
	Items() (BindingItems, error)

	// reference to resource holding the bindings, nil if not persisted in a resource
	Source() *v1.ObjectReference
}

// Returns map representation of given list of binding items
func (items *BindingItems) AsMap() map[string]string {
	result := make(map[string]string)

	for _, i := range *items {
		result[i.Name] = fmt.Sprintf("%v", i.Value)
	}
	return result
}
