package builder

import (
	"github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/handler/collect"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/handler/mapping"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/handler/naming"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/handler/project"
)

var _ pipeline.Pipeline = &impl{}

type impl struct {
	ctxProvider pipeline.ContextProvider
	handlers    []pipeline.Handler
}

func (i *impl) Process(binding *v1alpha1.ServiceBinding) (bool, error) {
	ctx, err := i.ctxProvider.Get(binding)
	if err != nil {
		return false, err
	}
	var status pipeline.FlowStatus
	for _, h := range i.handlers {
		h.Handle(ctx)
		status = ctx.FlowStatus()
		if status.Stop {
			break
		}
	}
	err = ctx.Close()
	if err != nil {
		return true, err
	}
	return status.Retry, status.Err
}

type builder struct {
	ctxProvider pipeline.ContextProvider
	handlers    []pipeline.Handler
}

func (b *builder) WithContextProvider(ctxProvider pipeline.ContextProvider) *builder {
	b.ctxProvider = ctxProvider
	return b
}

func (b *builder) WithHandlers(h ...pipeline.Handler) *builder {
	b.handlers = append(b.handlers, h...)
	return b
}

func (b *builder) Build() pipeline.Pipeline {
	return &impl{
		handlers:    b.handlers,
		ctxProvider: b.ctxProvider,
	}
}

func Builder() *builder {
	return &builder{}
}

var defaultFlow = []pipeline.Handler{
	pipeline.HandlerFunc(project.Unbind),
	pipeline.HandlerFunc(collect.PreFlight),
	pipeline.HandlerFunc(collect.ProvisionedService),
	pipeline.HandlerFunc(collect.BindingDefinitions),
	pipeline.HandlerFunc(collect.BindingItems),
	pipeline.HandlerFunc(collect.OwnedResources),
	pipeline.HandlerFunc(mapping.Handle),
	pipeline.HandlerFunc(naming.Handle),
	pipeline.HandlerFunc(project.PreFlightCheck),
	pipeline.HandlerFunc(project.InjectSecretRef),
	pipeline.HandlerFunc(project.BindingsAsEnv),
	pipeline.HandlerFunc(project.BindingsAsFiles),
	pipeline.HandlerFunc(project.PostFlightCheck),
}

var DefaultBuilder = Builder().WithHandlers(defaultFlow...)
