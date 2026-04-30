package main

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
	"time"
)

const (
	// shippingDispatchedTemplatePath defines the embedded transactional email template path.
	shippingDispatchedTemplatePath = "transactional/templates/shipping_dispatched.html.tmpl"
	// shippingMarkVoidedTemplatePath defines the embedded transactional email template path for voided marks.
	shippingMarkVoidedTemplatePath = "transactional/templates/shipping_mark_voided.html.tmpl"
)

//go:embed transactional/templates/*.html.tmpl
var shippingTemplateFiles embed.FS

// shippingTemplateRenderer renders transactional shipping HTML templates.
type shippingTemplateRenderer struct {
	// dispatchedSource defines raw dispatched template source values.
	dispatchedSource string
	// voidedSource defines raw voided template source values.
	voidedSource string
	// renderer defines go-template renderer dependencies.
	renderer *shippingTemplateEngine
}

type shippingTemplateEngine struct{}

// newShippingTemplateRenderer creates transactional template renderers.
func newShippingTemplateRenderer() (*shippingTemplateRenderer, error) {
	dispatchedRawTemplate, err := shippingTemplateFiles.ReadFile(shippingDispatchedTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("read shipping dispatched transactional template: %w", err)
	}
	voidedRawTemplate, err := shippingTemplateFiles.ReadFile(shippingMarkVoidedTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("read shipping mark voided transactional template: %w", err)
	}

	return &shippingTemplateRenderer{
		dispatchedSource: string(dispatchedRawTemplate),
		voidedSource:     string(voidedRawTemplate),
		renderer:         &shippingTemplateEngine{},
	}, nil
}

// RenderHTML renders the transactional shipping HTML template.
func (r *shippingTemplateRenderer) RenderHTML(data shippingDispatchedTemplateData) (string, error) {
	if r == nil || r.renderer == nil {
		return "", fmt.Errorf("shipping template renderer is not configured")
	}

	return r.renderer.Render("shipping_dispatched_html", r.dispatchedSource, data)
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

// RenderVoidedHTML renders the transactional shipping mark-voided HTML template.
func (r *shippingTemplateRenderer) RenderVoidedHTML(data shippingDispatchedTemplateData) (string, error) {
	if r == nil || r.renderer == nil {
		return "", fmt.Errorf("shipping template renderer is not configured")
	}

	return r.renderer.Render("shipping_mark_voided_html", r.voidedSource, data)
}

// RenderVoidedText renders a plain-text fallback body for transactional voided shipping mark emails.
func (r *shippingTemplateRenderer) RenderVoidedText(data shippingDispatchedTemplateData) string {
	builder := strings.Builder{}
	builder.WriteString("Hola ")
	builder.WriteString(strings.TrimSpace(data.FirstName))
	builder.WriteString(",\n\n")
	builder.WriteString("Queremos informarte que, por un error en el proceso de despacho, se genero una guia de envio para tu pedido que ha tenido que ser anulada.\n\n")
	builder.WriteString("Esto puede deberse a diferentes motivos logisticos, como ajustes en la preparacion del pedido o validaciones internas. Por esta razon, la guia anterior ya no es valida y no tendra movimiento.\n\n")
	builder.WriteString("No tienes que realizar ninguna accion por tu parte. Una vez se genere la nueva guia correcta, recibiras un nuevo correo con la informacion actualizada para el seguimiento de tu pedido.\n\n")
	builder.WriteString("Lamentamos cualquier inconveniente que esto pueda ocasionar y agradecemos tu comprension.")

	return builder.String()
}

func (r *shippingTemplateEngine) Render(name string, src string, data any) (string, error) {
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"formatDate": func(value *time.Time) string {
			if value == nil {
				return ""
			}
			return value.Format("2006-01-02")
		},
		"formatPrice": func(price float64) string {
			return fmt.Sprintf("%.2f", price)
		},
		"default": func(fallback string, value string) string {
			if value == "" {
				return fallback
			}
			return value
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}).Parse(src)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", name, err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", name, err)
	}

	return buffer.String(), nil
}
