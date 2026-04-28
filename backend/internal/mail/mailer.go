package mail

import (
	"fmt"
	"strings"
)

// FooterCtx — общий контекст футера (вшивается во все письма через layout).
type FooterCtx struct {
	SupportContact string
}

// Mailer — высокоуровневый отправитель доменных писем. Внутри использует
// Sender + готовые шаблоны. Имеет три семантичных метода под три типа писем.
type Mailer struct {
	sender *Sender
	tpls   *templates
	footer FooterCtx
}

// NewMailer возвращает Mailer. Если шаблоны не парсятся — ошибка фатальная,
// возвращается из конструктора (баг в коде, а не в рантайме).
func NewMailer(sender *Sender, supportContact string) (*Mailer, error) {
	tpls, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	return &Mailer{
		sender: sender,
		tpls:   tpls,
		footer: FooterCtx{SupportContact: supportContact},
	}, nil
}

// Configured — true, если SMTP настроен и можно реально отправлять.
// В dev-окружении без SMTP_PASS — false, тогда вызовы Send* возвращают ошибку
// (а воркер логирует и идёт в retry — не блокирует основной флоу).
func (m *Mailer) Configured() bool { return m.sender.Configured() }

// ---------- Восстановление пароля ----------

type PasswordResetData struct {
	UserName  string
	ResetLink string
	Footer    FooterCtx
}

func (m *Mailer) SendPasswordReset(to string, d PasswordResetData) error {
	d.Footer = m.footer
	html, err := m.tpls.renderHTML("password_reset.html", d)
	if err != nil {
		return err
	}
	text, err := m.tpls.renderText("password_reset.txt", d)
	if err != nil {
		return err
	}
	return m.sender.Send(Message{
		To:       to,
		Subject:  "Восстановление пароля Yulik3D",
		HTMLBody: html,
		TextBody: text,
	})
}

// ---------- Новый заказ (админу) ----------

type OrderItemLine struct {
	Name           string
	Quantity       int
	UnitTotalPrice int
	Subtotal       int
	Options        []OrderItemOptionLine
}

type OrderItemOptionLine struct {
	TypeLabel string
	Value     string
	Price     int // 0 если без доплаты
}

type OrderCreatedAdminData struct {
	OrderID         string // полный UUID
	OrderIDShort    string // первые 8 символов для удобочитаемости
	CreatedAt       string // отформатированная дата
	Total           int
	UserName        string
	UserEmail       string
	UserPhone       string
	ContactFullName string
	ContactPhone    string
	CustomerComment string // может быть пустым → секция не показывается
	Items           []OrderItemLine
	AdminURL        string
	Footer          FooterCtx
}

func (m *Mailer) SendOrderCreatedAdmin(to string, d OrderCreatedAdminData) error {
	d.Footer = m.footer
	html, err := m.tpls.renderHTML("order_created_admin.html", d)
	if err != nil {
		return err
	}
	text, err := m.tpls.renderText("order_created_admin.txt", d)
	if err != nil {
		return err
	}
	return m.sender.Send(Message{
		To:       to,
		Subject:  fmt.Sprintf("Новый заказ %s на сумму %s ₽", d.OrderID, formatPrice(d.Total)),
		HTMLBody: html,
		TextBody: text,
	})
}

// ---------- Смена статуса (пользователю) ----------

type OrderStatusChangedData struct {
	UserName        string
	OrderIDShort    string
	StatusKey       string // "created" | "confirmed" | "manufacturing" | "delivering" | "completed" | "cancelled"
	StatusHuman     string // «Передан в производство» и т.п.
	Total           int
	ItemsSummary    []string // для статуса «Создан» показываем краткий состав
	AdminNote       string   // при «Отменён» — может быть пустым
	OrderURL        string
	Footer          FooterCtx
}

func (m *Mailer) SendOrderStatusChanged(to string, d OrderStatusChangedData) error {
	d.Footer = m.footer
	d.StatusHuman = StatusHuman(d.StatusKey)
	html, err := m.tpls.renderHTML("order_status_changed_user.html", d)
	if err != nil {
		return err
	}
	text, err := m.tpls.renderText("order_status_changed_user.txt", d)
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("Заказ #%s — %s", d.OrderIDShort, strings.ToLower(d.StatusHuman))
	return m.sender.Send(Message{
		To:       to,
		Subject:  subject,
		HTMLBody: html,
		TextBody: text,
	})
}

// StatusHuman — человекочитаемое название статуса для писем и UI.
func StatusHuman(key string) string {
	switch key {
	case "created":
		return "Создан"
	case "confirmed":
		return "Подтверждён"
	case "manufacturing":
		return "В производстве"
	case "delivering":
		return "В доставке"
	case "completed":
		return "Завершён"
	case "cancelled":
		return "Отменён"
	}
	return key
}
