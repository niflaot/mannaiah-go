package main

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	contactdomain "mannaiah/module/contacts/domain"
	coremsgbus "mannaiah/module/core/messaging/bus"
	emailapplication "mannaiah/module/email/application"
	emaildomain "mannaiah/module/email/domain"
	ordersdomain "mannaiah/module/orders/domain"
	productdomain "mannaiah/module/products/domain/product"
	variationdomain "mannaiah/module/products/domain/variation"
	shippingdomain "mannaiah/module/shipping/domain"
	shippingport "mannaiah/module/shipping/port"
)

// shippingMarkLookupService defines mark lookup behavior required by shipping transactional email consumers.
type shippingMarkLookupService interface {
	// Get resolves one shipping mark by id.
	Get(ctx context.Context, id string) (*shippingdomain.ShippingMark, error)
}

// shippingCarrierLookupService defines carrier lookup behavior required by shipping transactional email consumers.
type shippingCarrierLookupService interface {
	// Get resolves one configured carrier by id.
	Get(ctx context.Context, id string) (*shippingdomain.Carrier, error)
}

// shippingEmailConsumerDependencies defines dependencies required by shipping mark transactional email consumers.
type shippingEmailConsumerDependencies struct {
	// marks defines shipping mark lookup dependencies.
	marks shippingMarkLookupService
	// carriers defines carrier lookup dependencies.
	carriers shippingCarrierLookupService
	// orders defines order lookup dependencies.
	orders shippingEmailOrderService
	// contacts defines contact lookup dependencies.
	contacts shippingEmailContactService
	// products defines product lookup dependencies.
	products shippingEmailProductService
	// variations defines variation lookup dependencies.
	variations shippingEmailVariationService
	// emails defines email delivery dependencies.
	emails shippingEmailSender
	// assetResolver defines asset URL resolution dependencies.
	assetResolver analyticsAssetURLResolver
	// templateRenderer defines transactional template rendering dependencies.
	templateRenderer *shippingTemplateRenderer
}

// shippingEmailOrderService defines order lookup behavior required by shipping transactional email consumers.
type shippingEmailOrderService interface {
	// Get resolves order aggregate values by identifier.
	Get(ctx context.Context, id string) (*ordersdomain.Order, error)
}

// shippingEmailContactService defines contact lookup behavior required by shipping transactional email consumers.
type shippingEmailContactService interface {
	// Get resolves one contact by identifier.
	Get(ctx context.Context, id string) (*contactdomain.Contact, error)
}

// shippingEmailProductService defines product lookup behavior required by shipping transactional email consumers.
type shippingEmailProductService interface {
	// GetBySKU resolves one product by product/variant SKU.
	GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error)
}

// shippingEmailVariationService defines variation lookup behavior required by shipping transactional email consumers.
type shippingEmailVariationService interface {
	// Get resolves one variation by identifier.
	Get(ctx context.Context, id string) (*variationdomain.Variation, error)
}

// shippingEmailSender defines email send behavior required by shipping transactional email consumers.
type shippingEmailSender interface {
	// Send dispatches one email and tracks delivery status.
	Send(ctx context.Context, command emailapplication.SendCommand) (*emaildomain.Delivery, error)
}

// registerShippingMarkTransactionalEmailConsumer registers shipping-mark handlers that send transactional emails.
func registerShippingMarkTransactionalEmailConsumer(
	registrar coremsgbus.Registrar,
	deps shippingEmailConsumerDependencies,
	providedLogger *zap.Logger,
) error {
	if registrar == nil || deps.marks == nil || deps.orders == nil || deps.contacts == nil || deps.emails == nil || deps.templateRenderer == nil {
		return nil
	}
	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	handleMarkGenerated := func(ctx context.Context, message coremsgbus.Message) error {
		payload, err := decodeShippingMarkGeneratedPayload(message)
		if err != nil {
			logger.Warn("decode shipping mark generated payload failed for transactional email", zap.Error(err))
			return err
		}

		command, buildErr := buildShippingDispatchedEmailCommand(ctx, deps, payload)
		if buildErr != nil {
			return buildErr
		}
		if command == nil {
			return nil
		}

		if _, sendErr := deps.emails.Send(ctx, *command); sendErr != nil {
			if isDuplicateEmailIdempotencyError(sendErr) {
				logger.Info("skip duplicate transactional shipping email", zap.String("idempotency_key", strings.TrimSpace(command.IdempotencyKey)))
				return nil
			}

			return fmt.Errorf("send transactional shipping email: %w", sendErr)
		}

		return nil
	}

	handleMarkVoided := func(ctx context.Context, message coremsgbus.Message) error {
		payload, err := decodeShippingMarkGeneratedPayload(message)
		if err != nil {
			logger.Warn("decode shipping mark voided payload failed for transactional email", zap.Error(err))
			return err
		}

		command, buildErr := buildShippingMarkVoidedEmailCommand(ctx, deps, payload)
		if buildErr != nil {
			return buildErr
		}
		if command == nil {
			return nil
		}

		if _, sendErr := deps.emails.Send(ctx, *command); sendErr != nil {
			if isDuplicateEmailIdempotencyError(sendErr) {
				logger.Info("skip duplicate transactional shipping mark voided email", zap.String("idempotency_key", strings.TrimSpace(command.IdempotencyKey)))
				return nil
			}

			return fmt.Errorf("send transactional shipping mark voided email: %w", sendErr)
		}

		return nil
	}

	if err := registrar.AddHandler(shippingport.TopicMarkGenerated, handleMarkGenerated); err != nil {
		return err
	}

	return registrar.AddHandler(shippingport.TopicMarkVoided, handleMarkVoided)
}

// buildShippingDispatchedEmailCommand builds one transactional shipping email send command.
func buildShippingDispatchedEmailCommand(
	ctx context.Context,
	deps shippingEmailConsumerDependencies,
	payload shippingMarkGeneratedPayload,
) (*emailapplication.SendCommand, error) {
	mark, err := deps.marks.Get(ctx, payload.MarkID)
	if err != nil || mark == nil {
		return nil, fmt.Errorf("load shipping mark for transactional email: %w", err)
	}
	order, err := deps.orders.Get(ctx, payload.OrderID)
	if err != nil || order == nil {
		return nil, fmt.Errorf("load order for transactional email: %w", err)
	}
	contact, err := deps.contacts.Get(ctx, strings.TrimSpace(order.ContactID))
	if err != nil || contact == nil {
		return nil, fmt.Errorf("load contact for transactional email: %w", err)
	}

	recipientEmail := firstNonEmpty(strings.TrimSpace(contact.Email), strings.TrimSpace(mark.Recipient.Email))
	if recipientEmail == "" {
		return nil, nil
	}
	trackingNumber := firstNonEmpty(strings.TrimSpace(mark.TrackingNumber), strings.TrimSpace(payload.TrackingNumber), strings.TrimSpace(mark.DocumentRef), strings.TrimSpace(mark.ID))
	shippingNumber := firstNonEmpty(strings.TrimSpace(mark.DocumentRef), strings.TrimSpace(mark.TrackingNumber), strings.TrimSpace(payload.DocumentRef), strings.TrimSpace(mark.ID))
	orderNumber := firstNonEmpty(strings.TrimSpace(order.Identifier), strings.TrimSpace(order.ID))
	carrierName := resolveCarrierName(ctx, deps.carriers, mark.CarrierID)

	templateData := buildShippingDispatchedTemplateData(ctx, deps, *order, *contact, *mark, shippingDispatchedRenderMeta{
		OrderNumber:    orderNumber,
		CarrierName:    carrierName,
		TrackingNumber: trackingNumber,
		ShippingNumber: shippingNumber,
	})
	renderedHTML, renderErr := deps.templateRenderer.RenderHTML(templateData)
	if renderErr != nil {
		return nil, fmt.Errorf("render shipping transactional html template: %w", renderErr)
	}

	return &emailapplication.SendCommand{
		ContactID:      strings.TrimSpace(contact.ID),
		Email:          strings.TrimSpace(recipientEmail),
		Subject:        "Tu pedido #" + strings.TrimSpace(orderNumber) + " fue despachado",
		HTMLBody:       renderedHTML,
		TextBody:       deps.templateRenderer.RenderText(templateData),
		IdempotencyKey: "shipping_mark_dispatched:" + strings.TrimSpace(mark.ID),
	}, nil
}

// buildShippingMarkVoidedEmailCommand builds one transactional voided shipping mark email send command.
func buildShippingMarkVoidedEmailCommand(
	ctx context.Context,
	deps shippingEmailConsumerDependencies,
	payload shippingMarkGeneratedPayload,
) (*emailapplication.SendCommand, error) {
	mark, err := deps.marks.Get(ctx, payload.MarkID)
	if err != nil || mark == nil {
		return nil, fmt.Errorf("load shipping mark for transactional voided email: %w", err)
	}
	order, err := deps.orders.Get(ctx, payload.OrderID)
	if err != nil || order == nil {
		return nil, fmt.Errorf("load order for transactional voided email: %w", err)
	}
	contact, err := deps.contacts.Get(ctx, strings.TrimSpace(order.ContactID))
	if err != nil || contact == nil {
		return nil, fmt.Errorf("load contact for transactional voided email: %w", err)
	}

	recipientEmail := firstNonEmpty(strings.TrimSpace(contact.Email), strings.TrimSpace(mark.Recipient.Email))
	if recipientEmail == "" {
		return nil, nil
	}
	trackingNumber := firstNonEmpty(strings.TrimSpace(mark.TrackingNumber), strings.TrimSpace(payload.TrackingNumber), strings.TrimSpace(mark.DocumentRef), strings.TrimSpace(mark.ID))
	shippingNumber := firstNonEmpty(strings.TrimSpace(mark.DocumentRef), strings.TrimSpace(mark.TrackingNumber), strings.TrimSpace(payload.DocumentRef), strings.TrimSpace(mark.ID))
	orderNumber := firstNonEmpty(strings.TrimSpace(order.Identifier), strings.TrimSpace(order.ID))
	carrierName := resolveCarrierName(ctx, deps.carriers, mark.CarrierID)

	templateData := buildShippingDispatchedTemplateData(ctx, deps, *order, *contact, *mark, shippingDispatchedRenderMeta{
		OrderNumber:    orderNumber,
		CarrierName:    carrierName,
		TrackingNumber: trackingNumber,
		ShippingNumber: shippingNumber,
	})
	renderedHTML, renderErr := deps.templateRenderer.RenderVoidedHTML(templateData)
	if renderErr != nil {
		return nil, fmt.Errorf("render shipping mark voided transactional html template: %w", renderErr)
	}

	return &emailapplication.SendCommand{
		ContactID:      strings.TrimSpace(contact.ID),
		Email:          strings.TrimSpace(recipientEmail),
		Subject:        "Actualizacion de envio de tu pedido #" + strings.TrimSpace(orderNumber),
		HTMLBody:       renderedHTML,
		TextBody:       deps.templateRenderer.RenderVoidedText(templateData),
		IdempotencyKey: "shipping_mark_voided:" + strings.TrimSpace(mark.ID),
	}, nil
}

// resolveCarrierName resolves one carrier display name by id.
func resolveCarrierName(ctx context.Context, service shippingCarrierLookupService, carrierID string) string {
	trimmedCarrierID := strings.TrimSpace(carrierID)
	if service == nil || trimmedCarrierID == "" {
		return trimmedCarrierID
	}
	carrier, err := service.Get(ctx, trimmedCarrierID)
	if err != nil || carrier == nil {
		return trimmedCarrierID
	}

	return firstNonEmpty(strings.TrimSpace(carrier.Name), trimmedCarrierID)
}

// isDuplicateEmailIdempotencyError reports whether send errors are caused by idempotency-key uniqueness conflicts.
func isDuplicateEmailIdempotencyError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return false
	}
	if strings.Contains(message, "idempotency_key") && strings.Contains(message, "duplicate") {
		return true
	}
	if strings.Contains(message, "uq_email_deliveries_idempotency_key") {
		return true
	}

	return false
}

// firstNonEmpty resolves the first non-empty string value.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
