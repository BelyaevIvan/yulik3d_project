// Package main — точка входа HTTP-сервера yulik3d.
//
// @title           yulik3d API
// @version         1.0
// @description     Интернет-магазин 3D-печатных фигурок и макетов. Все защищённые методы используют session cookie (httpOnly). После вызова /auth/login cookie автоматически проставится браузером — Swagger UI будет слать её во все последующие запросы.
// @BasePath        /api/v1
//
// @securityDefinitions.apikey CookieAuth
// @in                          cookie
// @name                        session
// @description                 Session cookie, выставляется при успешном /auth/login или /auth/register. HttpOnly, Secure (в prod), SameSite=Lax.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx" в database/sql
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"

	"yulik3d/config"
	"yulik3d/internal/captcha"
	"yulik3d/internal/handler"
	"yulik3d/internal/mail"
	"yulik3d/internal/middleware"
	"yulik3d/internal/model"
	"yulik3d/internal/queue"
	"yulik3d/internal/repository"
	"yulik3d/internal/service"
	"yulik3d/migrations"
	"yulik3d/pkg/cookie"
	"yulik3d/pkg/logger"
	"yulik3d/pkg/passwordhash"
)

func main() {
	// Локальный dev — подхватим .env, если запускают не через docker-compose.
	_ = godotenv.Load(".env", "../.env")

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}

	log := logger.New(cfg.App.LogLevel, os.Stdout)
	log.Info("starting", "env", cfg.App.Env, "port", cfg.HTTP.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ---------- Postgres ----------
	pgCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN())
	if err != nil {
		log.Error("pgxpool parse", "err", err)
		os.Exit(2)
	}
	pgCfg.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		log.Error("pgxpool connect", "err", err)
		os.Exit(2)
	}
	defer pool.Close()
	if err := waitForPostgres(ctx, pool, log); err != nil {
		log.Error("postgres not ready", "err", err)
		os.Exit(2)
	}

	// ---------- Migrations ----------
	if err := runMigrations(cfg.Postgres.DSN(), log); err != nil {
		log.Error("migrations", "err", err)
		os.Exit(2)
	}

	// ---------- Redis ----------
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()
	if err := waitForRedis(ctx, rdb, log); err != nil {
		log.Error("redis not ready", "err", err)
		os.Exit(2)
	}

	// ---------- MinIO ----------
	mcl, err := minio.New(cfg.MinIO.Endpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.RootUser, cfg.MinIO.RootPass, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		log.Error("minio client", "err", err)
		os.Exit(2)
	}
	if err := ensureBucket(ctx, mcl, cfg.MinIO.Bucket, log); err != nil {
		log.Error("minio bucket", "err", err)
		os.Exit(2)
	}
	minioSvc := service.NewMinioClient(mcl, cfg.MinIO.Bucket, cfg.MinIO.PublicURL)

	// ---------- Repositories ----------
	db := repository.NewDB(pool)
	tx := repository.NewTxManager(pool)

	userRepo := repository.NewUserRepo(db)
	sessionRepo := repository.NewSessionRepo(rdb, cfg.Session.TTL)
	rateRepo := repository.NewRateLimitRepo(rdb)
	itemRepo := repository.NewItemRepo(db)
	pictureRepo := repository.NewPictureRepo(db)
	optionTypeRepo := repository.NewOptionTypeRepo(db)
	itemOptionRepo := repository.NewItemOptionRepo(db)
	categoryRepo := repository.NewCategoryRepo(db)
	subcategoryRepo := repository.NewSubcategoryRepo(db)
	itemSubRepo := repository.NewItemSubcategoryRepo(db)
	favoriteRepo := repository.NewFavoriteRepo(db)
	orderRepo := repository.NewOrderRepo(db)
	pwResetRepo := repository.NewPasswordResetRepo(rdb, cfg.PasswordReset.TokenTTL, cfg.PasswordReset.Throttle)
	emailVerifyRepo := repository.NewEmailVerifyRepo(rdb, cfg.EmailVerify.TokenTTL, cfg.EmailVerify.Throttle)
	mainPinRepo := repository.NewItemMainPinRepo(db)

	// ---------- Mail + Queue ----------
	smtpSender := mail.NewSender(
		mail.SMTPConfig{
			Host: cfg.SMTP.Host, Port: cfg.SMTP.Port,
			User: cfg.SMTP.User, Password: cfg.SMTP.Password,
			UseSSL: cfg.SMTP.UseSSL,
		},
		mail.FromAddress{Name: cfg.Mail.FromName, Email: cfg.Mail.FromEmail},
	)
	mailer, err := mail.NewMailer(smtpSender, cfg.Mail.SupportContact)
	if err != nil {
		log.Error("mail templates", "err", err)
		os.Exit(2)
	}
	if !mailer.Configured() {
		log.Warn("smtp not configured — emails will be skipped (set SMTP_* in .env to enable)")
	}

	asynqRedis := queue.RedisOpt{Addr: cfg.Redis.Addr(), Password: cfg.Redis.Password, DB: cfg.Redis.AsynqDB}
	queueClient := queue.NewClient(asynqRedis, log)
	defer queueClient.Close()

	queueServer := queue.NewServer(asynqRedis, log)
	queueServer.RegisterHandlers(mailer)
	if err := queueServer.Start(); err != nil {
		log.Error("asynq start", "err", err)
		os.Exit(2)
	}
	defer queueServer.Shutdown()

	mailEnq := queue.NewMailEnqueuer(queueClient)

	// ---------- Captcha ----------
	// FailClosed в production (безопасность важнее), FailOpen в development
	// (чтобы локальная разработка не страдала от лагающего интернета).
	captchaMode := captcha.FailClosed
	if cfg.App.Env != "production" {
		captchaMode = captcha.FailOpen
	}
	captchaVer := captcha.New(
		cfg.Captcha.Enabled,
		cfg.Captcha.Endpoint,
		cfg.Captcha.ServerKey,
		captchaMode,
		log,
	)

	// ---------- Services ----------
	authSvc := service.NewAuthService(userRepo, sessionRepo, rateRepo, pwResetRepo, emailVerifyRepo, mailEnq, captchaVer,
		cfg.App.PublicFrontendURL,
		passwordhash.Params{
			Memory:      cfg.Argon2.MemoryKiB,
			Iterations:  cfg.Argon2.Iterations,
			Parallelism: cfg.Argon2.Parallelism,
		},
		cfg.Session.TTL,
		cfg.RateLimit.AuthAttempts,
		cfg.RateLimit.AuthWindow,
	)
	catalogSvc := service.NewCatalogService(itemRepo, pictureRepo, itemOptionRepo, optionTypeRepo,
		itemSubRepo, categoryRepo, subcategoryRepo, mainPinRepo, minioSvc)
	favoriteSvc := service.NewFavoriteService(favoriteRepo, itemRepo, catalogSvc)
	orderSvc := service.NewOrderService(orderRepo, itemRepo, itemOptionRepo, optionTypeRepo, userRepo, tx,
		mailEnq, cfg.App.PublicFrontendURL, cfg.Mail.AdminNotify, log)
	adminItemSvc := service.NewAdminItemService(itemRepo, itemOptionRepo, optionTypeRepo, subcategoryRepo, mainPinRepo, catalogSvc, tx)
	adminPictureSvc := service.NewAdminPictureService(itemRepo, pictureRepo, minioSvc, tx, cfg.Uploads.MaxBytes)
	adminOptionSvc := service.NewAdminOptionService(optionTypeRepo, itemOptionRepo, itemRepo)
	adminCategorySvc := service.NewAdminCategoryService(categoryRepo, subcategoryRepo)
	adminMainPageSvc := service.NewAdminMainPageService(mainPinRepo, itemRepo, catalogSvc, tx, log)

	// ---------- Cookie options ----------
	cookieOpts := cookie.Options{
		Name:     cfg.Session.CookieName,
		Domain:   cfg.Session.CookieDomain,
		Secure:   cfg.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   cfg.Session.TTL,
		Path:     "/",
	}

	// ---------- Handlers ----------
	deps := handler.Deps{Log: log}
	healthH := handler.NewHealthHandler(deps, pool, rdb)
	authH := handler.NewAuthHandler(deps, authSvc, cookieOpts)
	catalogH := handler.NewCatalogHandler(deps, catalogSvc)
	favoriteH := handler.NewFavoriteHandler(deps, favoriteSvc)
	orderH := handler.NewOrderHandler(deps, orderSvc)
	adminItemH := handler.NewAdminItemHandler(deps, adminItemSvc, catalogSvc)
	adminPictureH := handler.NewAdminPictureHandler(deps, adminPictureSvc, cfg.Uploads.MaxBytes)
	adminOptionH := handler.NewAdminOptionHandler(deps, adminOptionSvc)
	adminCategoryH := handler.NewAdminCategoryHandler(deps, adminCategorySvc)
	adminOrderH := handler.NewAdminOrderHandler(deps, orderSvc)
	adminMainPageH := handler.NewAdminMainPageHandler(deps, adminMainPageSvc)
	sitemapH := handler.NewSitemapHandler(deps, itemRepo, cfg.App.PublicBackendURL)

	// ---------- Middleware ----------
	getSession := func(r *http.Request, id string) (model.Session, error) {
		return authSvc.GetSession(r.Context(), id)
	}
	requireAuth := middleware.RequireAuth(cookieOpts.Name, getSession, log)
	requireAdmin := middleware.RequireRole(model.RoleAdmin, log)
	rejectAuthed := middleware.RejectAuthed(cookieOpts.Name, getSession, log)

	// ---------- Router (net/http stdlib) ----------
	mux := http.NewServeMux()
	registerRoutes(mux, &routes{
		health:         healthH,
		auth:           authH,
		catalog:        catalogH,
		favorite:       favoriteH,
		order:          orderH,
		adminItem:      adminItemH,
		adminPicture:   adminPictureH,
		adminOption:    adminOptionH,
		adminCategory:  adminCategoryH,
		adminOrder:     adminOrderH,
		adminMainPage:  adminMainPageH,
		sitemap:        sitemapH,
		requireAuth:    requireAuth,
		requireAdmin:   requireAdmin,
		rejectAuthed:   rejectAuthed,
	})

	// Swagger UI
	mountSwagger(mux, cfg)

	// Глобальные middleware (в порядке обёртки — последний исполняется первым).
	var root http.Handler = mux
	root = middleware.CORS(cfg.HTTP.AllowedOrigins)(root)
	root = middleware.Recover(log)(root)
	root = middleware.Logging(log)(root)

	// ---------- HTTP-сервер + graceful shutdown ----------
	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Error("server", "err", err)
	case sig := <-stop:
		log.Info("shutdown", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "err", err)
	}
	log.Info("stopped")
}

// ---------- helpers ----------

func waitForPostgres(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := pool.Ping(ctx); err == nil {
			log.Info("postgres ready")
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("timeout waiting for postgres")
		}
		time.Sleep(1 * time.Second)
	}
}

func waitForRedis(ctx context.Context, rdb *redis.Client, log *slog.Logger) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := rdb.Ping(ctx).Err(); err == nil {
			log.Info("redis ready")
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("timeout waiting for redis")
		}
		time.Sleep(1 * time.Second)
	}
}

func ensureBucket(ctx context.Context, mcl *minio.Client, bucket string, log *slog.Logger) error {
	exists, err := mcl.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		if err := mcl.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
		log.Info("minio bucket created", "bucket", bucket)
	}
	return nil
}

// runMigrations — применяет миграции из embed.FS (`migrations/*.sql`).
func runMigrations(dsn string, log *slog.Logger) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql: %w", err)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	log.Info("migrations applied")
	return nil
}

