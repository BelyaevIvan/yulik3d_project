package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"yulik3d/internal/mail"
)

// RegisterHandlers — связывает типы задач с обработчиками. Вызывается в main.go
// после создания Server и Mailer.
func (s *Server) RegisterHandlers(m *mail.Mailer) {
	s.mux.HandleFunc(TaskEmailOrderCreatedAdmin, func(ctx context.Context, t *asynq.Task) error {
		var p EmailOrderCreatedAdminPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		if !m.Configured() {
			s.log.Warn("smtp not configured, skipping email", "type", t.Type(), "to", p.To)
			return nil // не ретраим — конфиг не появится сам
		}
		items := make([]mail.OrderItemLine, 0, len(p.Items))
		for _, it := range p.Items {
			opts := make([]mail.OrderItemOptionLine, 0, len(it.Options))
			for _, o := range it.Options {
				opts = append(opts, mail.OrderItemOptionLine{
					TypeLabel: o.TypeLabel, Value: o.Value, Price: o.Price,
				})
			}
			items = append(items, mail.OrderItemLine{
				Name: it.Name, Quantity: it.Quantity,
				UnitTotalPrice: it.UnitTotalPrice, Subtotal: it.Subtotal,
				Options: opts,
			})
		}
		err := m.SendOrderCreatedAdmin(p.To, mail.OrderCreatedAdminData{
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
		if err != nil {
			s.log.Error("send order_created_admin", "err", err, "to", p.To, "order", p.OrderID)
			return err // asynq сделает retry
		}
		s.log.Info("email sent", "type", t.Type(), "to", p.To, "order", p.OrderID)
		return nil
	})

	s.mux.HandleFunc(TaskEmailOrderStatusChanged, func(ctx context.Context, t *asynq.Task) error {
		var p EmailOrderStatusChangedPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		if !m.Configured() {
			s.log.Warn("smtp not configured, skipping email", "type", t.Type(), "to", p.To)
			return nil
		}
		err := m.SendOrderStatusChanged(p.To, mail.OrderStatusChangedData{
			UserName:     p.UserName,
			OrderIDShort: p.OrderIDShort,
			StatusKey:    p.StatusKey,
			Total:        p.Total,
			ItemsSummary: p.ItemsSummary,
			AdminNote:    p.AdminNote,
			OrderURL:     p.OrderURL,
		})
		if err != nil {
			s.log.Error("send order_status_changed", "err", err, "to", p.To, "status", p.StatusKey)
			return err
		}
		s.log.Info("email sent", "type", t.Type(), "to", p.To, "status", p.StatusKey)
		return nil
	})

	s.mux.HandleFunc(TaskEmailPasswordReset, func(ctx context.Context, t *asynq.Task) error {
		var p EmailPasswordResetPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		if !m.Configured() {
			s.log.Warn("smtp not configured, skipping email", "type", t.Type(), "to", p.To)
			return nil
		}
		err := m.SendPasswordReset(p.To, mail.PasswordResetData{
			UserName:  p.UserName,
			ResetLink: p.ResetLink,
		})
		if err != nil {
			s.log.Error("send password_reset", "err", err, "to", p.To)
			return err
		}
		s.log.Info("email sent", "type", t.Type(), "to", p.To)
		return nil
	})

	s.mux.HandleFunc(TaskEmailVerify, func(ctx context.Context, t *asynq.Task) error {
		var p EmailVerifyPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		if !m.Configured() {
			s.log.Warn("smtp not configured, skipping email", "type", t.Type(), "to", p.To)
			return nil
		}
		err := m.SendEmailVerify(p.To, mail.EmailVerifyData{
			UserName:   p.UserName,
			VerifyLink: p.VerifyLink,
		})
		if err != nil {
			s.log.Error("send email_verify", "err", err, "to", p.To)
			return err
		}
		s.log.Info("email sent", "type", t.Type(), "to", p.To)
		return nil
	})
}
