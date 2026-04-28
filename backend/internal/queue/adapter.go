package queue

import (
	"context"

	"yulik3d/internal/service"
)

// MailEnqueuer — реализация сервисных интерфейсов
// (service.OrderMailer, service.PasswordResetMailer) поверх Client.
//
// Сервисы зависят от интерфейсов, не от asynq → чистая архитектура соблюдена.
type MailEnqueuer struct {
	client *Client
}

func NewMailEnqueuer(c *Client) *MailEnqueuer {
	return &MailEnqueuer{client: c}
}

// --- service.OrderMailer ---

func (m *MailEnqueuer) EnqueueOrderCreatedAdmin(ctx context.Context, p service.OrderCreatedAdminMail) error {
	items := make([]OrderLinePayload, 0, len(p.Items))
	for _, it := range p.Items {
		opts := make([]OrderLineOptionPayload, 0, len(it.Options))
		for _, o := range it.Options {
			opts = append(opts, OrderLineOptionPayload{
				TypeLabel: o.TypeLabel, Value: o.Value, Price: o.Price,
			})
		}
		items = append(items, OrderLinePayload{
			Name: it.Name, Quantity: it.Quantity,
			UnitTotalPrice: it.UnitTotalPrice, Subtotal: it.Subtotal,
			Options: opts,
		})
	}
	return m.client.EnqueueEmail(ctx, TaskEmailOrderCreatedAdmin, EmailOrderCreatedAdminPayload{
		To:              p.To,
		OrderID:         p.OrderID,
		OrderIDShort:    p.OrderIDShort,
		CreatedAt:       p.CreatedAt,
		Total:           p.Total,
		UserEmail:       p.UserEmail,
		ContactFullName: p.ContactFullName,
		ContactPhone:    p.ContactPhone,
		CustomerComment: p.CustomerComment,
		Items:           items,
		AdminURL:        p.AdminURL,
	})
}

func (m *MailEnqueuer) EnqueueOrderStatusChanged(ctx context.Context, p service.OrderStatusChangedMail) error {
	return m.client.EnqueueEmail(ctx, TaskEmailOrderStatusChanged, EmailOrderStatusChangedPayload{
		To:           p.To,
		UserName:     p.UserName,
		OrderIDShort: p.OrderIDShort,
		StatusKey:    p.StatusKey,
		Total:        p.Total,
		ItemsSummary: p.ItemsSummary,
		AdminNote:    p.AdminNote,
		OrderURL:     p.OrderURL,
	})
}

// --- service.PasswordResetMailer ---

func (m *MailEnqueuer) EnqueuePasswordReset(ctx context.Context, to, userName, resetLink string) error {
	return m.client.EnqueueEmail(ctx, TaskEmailPasswordReset, EmailPasswordResetPayload{
		To:        to,
		UserName:  userName,
		ResetLink: resetLink,
	})
}
