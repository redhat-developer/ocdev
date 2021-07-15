package mapping

import (
	"bytes"
	"encoding/json"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
	"text/template"
)

func Handle(ctx pipeline.Context) {
	bindingItems := ctx.BindingItems()
	templateVars := make(map[string]interface{})

	services, _ := ctx.Services()
	for _, s := range services {
		if s.Id() != nil {
			templateVars[*s.Id()] = s.Resource().Object
		}
	}
	for _, bi := range bindingItems {
		templateVars[bi.Name] = bi.Value
	}

	for name, valueTemplate := range ctx.Mappings() {
		tmpl, err := template.New("mappings").Funcs(template.FuncMap{"json": marshalToJSON}).Parse(valueTemplate)
		if err != nil {
			ctx.StopProcessing()
			ctx.Error(err)
			return
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, templateVars)
		if err != nil {
			ctx.StopProcessing()
			ctx.Error(err)
			return
		}
		ctx.AddBindingItem(&pipeline.BindingItem{Name: name, Value: buf.String()})
	}

}

func marshalToJSON(m interface{}) (string, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
