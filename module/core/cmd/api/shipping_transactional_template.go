package main

import (
	"embed"
	"fmt"
	"strings"

	campaigntemplate "mannaiah/module/campaign/application/template"
)

const (
	// shippingDispatchedTemplatePath defines the embedded transactional email template path.
	shippingDispatchedTemplatePath = "transactional/templates/shipping_dispatched.html.tmpl"
)

//go:embed transactional/templates/*.html.tmpl
var shippingTemplateFiles embed.FS

// shippingTemplateRenderer renders transactional shipping HTML templates.
type shippingTemplateRenderer struct {
	// source defines raw template source values.
	source string
	// renderer defines go-template renderer dependencies.
	renderer *campaigntemplate.Renderer
}

// newShippingTemplateRenderer creates transactional template renderers.
func newShippingTemplateRenderer() (*shippingTemplateRenderer, error) {
	rawTemplate, err := shippingTemplateFiles.ReadFile(shippingDispatchedTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("read shipping transactional template: %w", err)
	}

	return &shippingTemplateRenderer{
		source:   string(rawTemplate),
		renderer: campaigntemplate.NewRenderer(),
	}, nil
}

// RenderHTML renders the transactional shipping HTML template.
func (r *shippingTemplateRenderer) RenderHTML(data shippingDispatchedTemplateData) (string, error) {
	if r == nil || r.renderer == nil {
		return "", fmt.Errorf("shipping template renderer is not configured")
	}

	return r.renderer.Render("shipping_dispatched_html", r.source, data)
}

// RenderText renders a plain-text fallback body for transactional shipping emails.
func (r *shippingTemplateRenderer) RenderText(data shippingDispatchedTemplateData) string {
	builder := strings.Builder{}
	builder.WriteString("Hola ")
	builder.WriteString(strings.TrimSpace(data.FirstName))
	builder.WriteString(",\n\n")
	builder.WriteString("Tu pedido #")
	builder.WriteString(strings.TrimSpace(data.OrderNumber))
	builder.WriteString(" fue despachado con ")
	builder.WriteString(strings.TrimSpace(data.CarrierName))
	builder.WriteString(".\n")
	builder.WriteString("Numero de guia: ")
	builder.WriteString(strings.TrimSpace(data.TrackingNumber))
	builder.WriteString("\n")
	builder.WriteString("Rastreo: ")
	builder.WriteString(strings.TrimSpace(data.TrackingURL))
	builder.WriteString("\n\n")
	builder.WriteString("Si necesitas ayuda: ")
	builder.WriteString(strings.TrimSpace(data.HelpURL))

	return builder.String()
}
