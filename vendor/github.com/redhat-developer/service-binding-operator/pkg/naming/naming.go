package naming

import (
	"bytes"
	"errors"
	"html/template"
	"strings"
)

var TemplateError = errors.New("please check the namingStrategy template provided")

var templateFunctions = map[string]interface{}{
	"upper": strings.ToUpper,
	"title": strings.Title,
	"lower": strings.ToLower,
}

type namingTemplate struct {
	template       *template.Template
	data           map[string]interface{}
	namingTemplate string
}

// NewTemplate creates template instance which handles how binding names should be prepared
// templateStr is being used to format binding name.
func NewTemplate(templateStr string, data map[string]interface{}) (*namingTemplate, error) {
	t, err := template.New("template").Funcs(templateFunctions).Parse(templateStr)
	if err != nil {
		return nil, err
	}
	return &namingTemplate{
		template:       t,
		namingTemplate: templateStr,
		data:           data,
	}, nil
}

// GetBindingName prepares binding name which accepts binding name from the OLM descriptor/annotation
// namingTemplate uses string template provided to build final binding name.
func (n *namingTemplate) GetBindingName(bindingName string) (string, error) {
	d := map[string]interface{}{
		"service": n.data,
		"name":    bindingName,
	}

	var tpl bytes.Buffer
	err := n.template.Execute(&tpl, d)
	if err != nil {
		return "", TemplateError
	}
	return tpl.String(), nil
}
