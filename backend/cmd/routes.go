package main

import (
	"net/http"

	"yulik3d/internal/handler"
)

type mw = func(http.Handler) http.Handler

type routes struct {
	health         *handler.HealthHandler
	auth           *handler.AuthHandler
	catalog        *handler.CatalogHandler
	favorite       *handler.FavoriteHandler
	order          *handler.OrderHandler
	adminItem      *handler.AdminItemHandler
	adminPicture   *handler.AdminPictureHandler
	adminOption    *handler.AdminOptionHandler
	adminCategory  *handler.AdminCategoryHandler
	adminOrder     *handler.AdminOrderHandler
	adminMainPage  *handler.AdminMainPageHandler
	sitemap        *handler.SitemapHandler

	requireAuth  mw
	requireAdmin mw
	rejectAuthed mw
}

// registerRoutes — маршруты на ServeMux (Go 1.22+ с поддержкой паттернов).
func registerRoutes(mux *http.ServeMux, r *routes) {
	base := "/api/v1"

	// ---- Public ----
	mux.HandleFunc("GET "+base+"/health", r.health.Health)

	// SEO: sitemap.xml — корневой путь, не под /api/v1/
	mux.HandleFunc("GET /sitemap.xml", r.sitemap.Sitemap)

	// Auth (guest)
	mux.Handle("POST "+base+"/auth/register", r.rejectAuthed(http.HandlerFunc(r.auth.Register)))
	mux.Handle("POST "+base+"/auth/login", r.rejectAuthed(http.HandlerFunc(r.auth.Login)))

	// Восстановление пароля — без auth, доступно гостям
	mux.HandleFunc("POST "+base+"/auth/password/reset-request", r.auth.PasswordResetRequest)
	mux.HandleFunc("POST "+base+"/auth/password/reset-confirm", r.auth.PasswordResetConfirm)

	// Подтверждение email — публичные эндпоинты (юзер мог быть разлогинен
	// в момент клика по ссылке из письма; resend тоже работает по email из тела)
	mux.HandleFunc("POST "+base+"/auth/email/verify", r.auth.EmailVerifyConfirm)
	mux.HandleFunc("POST "+base+"/auth/email/verify/resend", r.auth.EmailVerifyResend)

	// Catalog (public)
	mux.HandleFunc("GET "+base+"/items", r.catalog.ListItems)
	mux.HandleFunc("GET "+base+"/items/main", r.catalog.MainPage)
	mux.HandleFunc("GET "+base+"/items/{id}", r.catalog.GetItem)
	mux.HandleFunc("GET "+base+"/categories", r.catalog.ListCategories)
	mux.HandleFunc("GET "+base+"/categories/{id}/subcategories", r.catalog.ListSubcategories)

	// ---- User (RequireAuth) ----
	mux.Handle("POST "+base+"/auth/logout", r.requireAuth(http.HandlerFunc(r.auth.Logout)))
	mux.Handle("GET "+base+"/me", r.requireAuth(http.HandlerFunc(r.auth.Me)))
	mux.Handle("PATCH "+base+"/me", r.requireAuth(http.HandlerFunc(r.auth.UpdateMe)))

	mux.Handle("GET "+base+"/favorites", r.requireAuth(http.HandlerFunc(r.favorite.List)))
	mux.Handle("POST "+base+"/favorites/{item_id}", r.requireAuth(http.HandlerFunc(r.favorite.Add)))
	mux.Handle("DELETE "+base+"/favorites/{item_id}", r.requireAuth(http.HandlerFunc(r.favorite.Remove)))

	mux.Handle("POST "+base+"/orders", r.requireAuth(http.HandlerFunc(r.order.Create)))
	mux.Handle("GET "+base+"/orders", r.requireAuth(http.HandlerFunc(r.order.ListMy)))
	mux.Handle("GET "+base+"/orders/{id}", r.requireAuth(http.HandlerFunc(r.order.GetMy)))

	// ---- Admin (RequireAuth + RequireRole(admin)) ----
	admin := func(h http.HandlerFunc) http.Handler {
		return r.requireAuth(r.requireAdmin(http.HandlerFunc(h)))
	}

	// items
	mux.Handle("GET "+base+"/admin/items", admin(r.adminItem.List))
	mux.Handle("POST "+base+"/admin/items", admin(r.adminItem.Create))
	mux.Handle("GET "+base+"/admin/items/{id}", admin(r.adminItem.Get))
	mux.Handle("PUT "+base+"/admin/items/{id}", admin(r.adminItem.Update))
	mux.Handle("PATCH "+base+"/admin/items/{id}", admin(r.adminItem.Patch))

	// pictures
	mux.Handle("POST "+base+"/admin/items/{id}/pictures", admin(r.adminPicture.Upload))
	mux.Handle("DELETE "+base+"/admin/items/{item_id}/pictures/{picture_id}", admin(r.adminPicture.Delete))
	mux.Handle("PATCH "+base+"/admin/items/{id}/pictures/reorder", admin(r.adminPicture.Reorder))

	// option types
	mux.Handle("GET "+base+"/admin/option-types", admin(r.adminOption.ListTypes))
	mux.Handle("POST "+base+"/admin/option-types", admin(r.adminOption.CreateType))
	mux.Handle("PATCH "+base+"/admin/option-types/{id}", admin(r.adminOption.PatchType))
	mux.Handle("DELETE "+base+"/admin/option-types/{id}", admin(r.adminOption.DeleteType))

	// item options
	mux.Handle("POST "+base+"/admin/items/{id}/options", admin(r.adminOption.CreateItemOption))
	mux.Handle("PATCH "+base+"/admin/item-options/{id}", admin(r.adminOption.PatchItemOption))
	mux.Handle("DELETE "+base+"/admin/item-options/{id}", admin(r.adminOption.DeleteItemOption))

	// categories
	mux.Handle("POST "+base+"/admin/categories", admin(r.adminCategory.CreateCategory))
	mux.Handle("PATCH "+base+"/admin/categories/{id}", admin(r.adminCategory.PatchCategory))
	mux.Handle("DELETE "+base+"/admin/categories/{id}", admin(r.adminCategory.DeleteCategory))
	mux.Handle("POST "+base+"/admin/categories/{id}/subcategories", admin(r.adminCategory.CreateSubcategory))
	mux.Handle("PATCH "+base+"/admin/subcategories/{id}", admin(r.adminCategory.PatchSubcategory))
	mux.Handle("DELETE "+base+"/admin/subcategories/{id}", admin(r.adminCategory.DeleteSubcategory))

	// admin main-page (закрепления)
	mux.Handle("GET "+base+"/admin/main", admin(r.adminMainPage.List))
	mux.Handle("POST "+base+"/admin/main", admin(r.adminMainPage.Pin))
	mux.Handle("DELETE "+base+"/admin/main/{type}/{item_id}", admin(r.adminMainPage.Unpin))
	mux.Handle("PATCH "+base+"/admin/main/{type}/reorder", admin(r.adminMainPage.Reorder))

	// admin orders
	mux.Handle("GET "+base+"/admin/orders", admin(r.adminOrder.List))
	mux.Handle("GET "+base+"/admin/orders/{id}", admin(r.adminOrder.Get))
	mux.Handle("PATCH "+base+"/admin/orders/{id}/status", admin(r.adminOrder.PatchStatus))
	mux.Handle("PATCH "+base+"/admin/orders/{id}", admin(r.adminOrder.Patch))
}
