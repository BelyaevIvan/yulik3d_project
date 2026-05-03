package mail

import (
	"bytes"
	"embed"
	"fmt"
	htmltpl "html/template"
	texttpl "text/template"
)

//go:embed templates/*.html templates/*.txt
var templatesFS embed.FS

// templates — лениво проинициализированные коллекции html/text шаблонов.
type templates struct {
	html *htmltpl.Template
	text *texttpl.Template
}

func loadTemplates() (*templates, error) {
	htmlTpl, err := htmltpl.New("").Funcs(htmlFuncs()).ParseFS(templatesFS,
		"templates/layout.html",
		"templates/order_created_admin.html",
		"templates/order_status_changed_user.html",
		"templates/password_reset.html",
		"templates/email_verify.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse html templates: %w", err)
	}
	textTpl, err := texttpl.New("").Funcs(textFuncs()).ParseFS(templatesFS,
		"templates/layout.txt",
		"templates/order_created_admin.txt",
		"templates/order_status_changed_user.txt",
		"templates/password_reset.txt",
		"templates/email_verify.txt",
	)
	if err != nil {
		return nil, fmt.Errorf("parse text templates: %w", err)
	}
	return &templates{html: htmlTpl, text: textTpl}, nil
}

func htmlFuncs() htmltpl.FuncMap {
	return htmltpl.FuncMap{
		"formatPrice": formatPrice,
	}
}

func textFuncs() texttpl.FuncMap {
	return texttpl.FuncMap{
		"formatPrice": formatPrice,
	}
}

func formatPrice(v int) string {
	// Простой формат с пробелами как разделителем тысяч.
	s := fmt.Sprintf("%d", v)
	n := len(s)
	if n <= 3 {
		return s
	}
	out := make([]byte, 0, n+n/3)
	for i, c := range []byte(s) {
		if i > 0 && (n-i)%3 == 0 {
			out = append(out, ' ')
		}
		out = append(out, c)
	}
	return string(out)
}

// renderHTML рендерит шаблон (верхнего уровня) в строку.
func (t *templates) renderHTML(name string, data any) (string, error) {
	var b bytes.Buffer
	if err := t.html.ExecuteTemplate(&b, name, data); err != nil {
		return "", fmt.Errorf("render html %s: %w", name, err)
	}
	return b.String(), nil
}

func (t *templates) renderText(name string, data any) (string, error) {
	var b bytes.Buffer
	if err := t.text.ExecuteTemplate(&b, name, data); err != nil {
		return "", fmt.Errorf("render text %s: %w", name, err)
	}
	return b.String(), nil
}
