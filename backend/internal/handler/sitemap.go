package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"yulik3d/internal/repository"
)

// SitemapHandler — отдаёт /sitemap.xml для поисковиков.
// Включает главную, статичные категории, и URL всех видимых товаров.
type SitemapHandler struct {
	Deps
	items     *repository.ItemRepo
	publicURL string
}

func NewSitemapHandler(d Deps, items *repository.ItemRepo, publicURL string) *SitemapHandler {
	return &SitemapHandler{Deps: d, items: items, publicURL: strings.TrimRight(publicURL, "/")}
}

// Sitemap godoc
// @Summary      Sitemap.xml для поисковых роботов
// @Tags         seo
// @Produce      xml
// @Success      200  {string}  string  "XML"
// @Router       /sitemap.xml [get]
func (h *SitemapHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	// Простой запрос: все НЕ скрытые товары, без пагинации (для MVP — до 5000 точно ок).
	// Если товаров вырастет до десятков тысяч — переделать на стрим.
	rows, err := h.items.ListAllVisible(r.Context())
	if err != nil {
		h.Log.Error("sitemap: list items", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC().Format("2006-01-02")

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/0.9">` + "\n")

	// Статичные публичные страницы
	staticPaths := []struct {
		path     string
		priority string
		changefreq string
	}{
		{"/", "1.0", "daily"},
		{"/figurines", "0.9", "daily"},
		{"/models", "0.9", "daily"},
	}
	for _, p := range staticPaths {
		fmt.Fprintf(&b,
			"  <url><loc>%s%s</loc><lastmod>%s</lastmod><changefreq>%s</changefreq><priority>%s</priority></url>\n",
			h.publicURL, p.path, now, p.changefreq, p.priority,
		)
	}

	// Товары
	for _, it := range rows {
		lastmod := it.UpdatedAt.UTC().Format("2006-01-02")
		fmt.Fprintf(&b,
			"  <url><loc>%s/product/%s</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>0.7</priority></url>\n",
			h.publicURL, it.ID.String(), lastmod,
		)
	}

	b.WriteString(`</urlset>`)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(b.String()))
}
