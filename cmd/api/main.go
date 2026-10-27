package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"github.com/ivancabzon/task-service/config"
	"github.com/ivancabzon/task-service/internal/handler"
	"github.com/ivancabzon/task-service/internal/middleware"
	"github.com/ivancabzon/task-service/internal/repository"
	"github.com/ivancabzon/task-service/internal/service"

	_ "github.com/ivancabzon/task-service/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Task Service API
// @version         1.0
// @description     REST API сервис для управления задачами внутри команд.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @host           localhost:8080
// @BasePath       /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите "Bearer " и ваш JWT токен. Пример: "Bearer eyJhbGciOi..."
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	db, err := sqlx.Connect("mysql", cfg.DBDSN)
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("mysql ping: %v", err)
	}
	log.Println("✅ MySQL connected")

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr, DB: cfg.RedisDB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}
	defer rdb.Close()
	log.Println("✅ Redis connected")

	userRepo := repository.NewUserRepo(db)
	teamRepo := repository.NewTeamRepo(db)
	taskRepo := repository.NewTaskRepo(db)
	cacheRepo := repository.NewCacheRepo(rdb)
	historyRepo := repository.NewHistoryRepo(db)
	commentRepo := repository.NewCommentRepo(db)
	statsRepo := repository.NewStatsRepo(db)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	teamSvc := service.NewTeamService(teamRepo, userRepo)
	taskSvc := service.NewTaskService(taskRepo, teamRepo, cacheRepo, historyRepo, commentRepo)
	statsSvc := service.NewStatsService(statsRepo, teamRepo)

	authHandler := handler.NewAuthHandler(authSvc)
	teamHandler := handler.NewTeamHandler(teamSvc)
	taskHandler := handler.NewTaskHandler(taskSvc)
	statsHandler := handler.NewStatsHandler(statsSvc)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db down"})
			return
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "redis down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	v1.POST("/register", authHandler.Register)
	v1.POST("/login", authHandler.Login)

	protected := v1.Group("")
	protected.Use(middleware.AuthRequired(cfg.JWTSecret))
	{
		protected.POST("/teams", teamHandler.Create)
		protected.GET("/teams", teamHandler.ListMine)
		protected.POST("/teams/:id/invite", teamHandler.Invite)

		protected.POST("/tasks", taskHandler.Create)
		protected.GET("/tasks", taskHandler.List)
		protected.PUT("/tasks/:id", taskHandler.Update)

		protected.GET("/tasks/:id/history", taskHandler.History)
		protected.POST("/tasks/:id/comments", taskHandler.AddComment)
		protected.GET("/tasks/:id/comments", taskHandler.ListComments)

		protected.GET("/teams/:team_id/stats", statsHandler.GetTeamStats)
	}

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{Addr: ":" + cfg.AppPort, Handler: r}
	go func() {
		log.Printf("🚀 server started on :%s (env=%s)", cfg.AppPort, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("⏳ shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("👋 server stopped")
}
