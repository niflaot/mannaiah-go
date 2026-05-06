package http

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	corehttp "mannaiah/module/core/http"
	shopifyport "mannaiah/module/shopify/port"
)

// appLaunch handles Shopify App URL launches for backend-only installations.
func (h *Handler) appLaunch(ctx corehttp.Context) error {
	shopDomain := shopifyport.NormalizeShopDomain(ctx.Query("shop", ""))
	if !isValidShopDomain(shopDomain) {
		shopDomain = ""
	}
	installed := strings.TrimSpace(ctx.Query("installed", "")) == "1"
	if shopDomain != "" && !installed && h != nil && h.installations != nil {
		installation, err := h.installations.FindByShopDomain(ctx.Context(), shopDomain)
		if err != nil {
			return h.mapError(err)
		}
		if installation == nil || installation.UninstalledAt != nil {
			ctx.SetHeader("Location", buildInstallLaunchPath(shopDomain))
			return ctx.Status(302).SendString("")
		}
		installed = true
	}

	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")

	return ctx.Status(200).SendString(renderAppLaunchPage(shopDomain, installed))
}

func buildInstallLaunchPath(shopDomain string) string {
	endpoint := url.URL{Path: "/shopify/oauth/install"}
	query := endpoint.Query()
	if normalizedShopDomain := shopifyport.NormalizeShopDomain(shopDomain); isValidShopDomain(normalizedShopDomain) {
		query.Set("shop", normalizedShopDomain)
	}
	endpoint.RawQuery = query.Encode()

	return endpoint.String()
}

// renderAppLaunchPage builds the HTML landing page returned from the Shopify App URL.
func renderAppLaunchPage(shopDomain string, installed bool) string {
	headline := "Mannaiah Shopify backend"
	lead := "Shopify abrio la landing tecnica de la app. Este servicio actua como backend de integracion y no expone una interfaz comercial completa en esta ruta."
	if installed {
		headline = "Instalacion de Shopify completada"
		lead = "La instalacion OAuth termino correctamente, el token offline ya fue persistido y el backend intento registrar los webhooks requeridos."
	}

	shopDetails := ""
	if strings.TrimSpace(shopDomain) != "" {
		shopDetails = fmt.Sprintf("<p><strong>Tienda:</strong> %s</p>", html.EscapeString(shopDomain))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Mannaiah Shopify backend</title>
    <style>
      body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #111827; color: #f9fafb; }
      main { max-width: 720px; margin: 48px auto; padding: 32px; background: #1f2937; border-radius: 16px; box-shadow: 0 24px 80px rgba(0, 0, 0, 0.35); }
      h1 { margin: 0 0 16px 0; font-size: 28px; line-height: 1.2; }
      p { margin: 0 0 12px 0; font-size: 16px; line-height: 1.6; color: #d1d5db; }
      a { color: #f59e0b; font-weight: 700; text-decoration: none; }
    </style>
  </head>
  <body>
    <main>
      <h1>%s</h1>
      <p>%s</p>
      %s
      <p>Ruta recomendada para Dev Dashboard: <strong>/shopify/app</strong>.</p>
      <p>Siguiente paso: desplegar la Admin UI extension si quieres tarjetas dentro del Shopify Admin, o validar la API directamente en <a href="/docs">/docs</a>.</p>
    </main>
  </body>
</html>`, headline, lead, shopDetails)
}
