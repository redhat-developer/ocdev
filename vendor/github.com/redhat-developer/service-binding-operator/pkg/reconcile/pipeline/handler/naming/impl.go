package naming

import (
	"github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/pkg/naming"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
)

const StrategyError = "NamingStrategyError"

func Handle(ctx pipeline.Context) {
	for _, item := range ctx.BindingItems() {
		if item.Source != nil {
			template, err := naming.NewTemplate(ctx.NamingTemplate(), templateData(item.Source))
			if err != nil {
				stop(ctx, err)
				return
			}
			item.Name, err = template.GetBindingName(item.Name)
			if err != nil {
				stop(ctx, err)
				return
			}
		}
	}
}

func templateData(service pipeline.Service) map[string]interface{} {
	res := service.Resource()
	return map[string]interface{}{
		"kind": res.GetKind(),
		"name": res.GetName(),
	}
}

func stop(ctx pipeline.Context, err error) {
	ctx.Error(err)
	ctx.StopProcessing()
	ctx.SetCondition(v1alpha1.Conditions().NotCollectionReady().Reason(StrategyError).Msg(err.Error()).Build())
}
