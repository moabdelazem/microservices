package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/moabdelazem/microservices/tasks/internal/database"
	"github.com/moabdelazem/microservices/tasks/internal/handlers"
	"github.com/moabdelazem/microservices/tasks/internal/middleware"
	"github.com/moabdelazem/microservices/tasks/internal/rabbitmq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	// Connect to database
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to RabbitMQ
	consumer, err := rabbitmq.NewConsumer(db)
	if err != nil {
		log.Fatalf("❌ Failed to connect to RabbitMQ: %v", err)
	}
	defer consumer.Close()

	// Start RabbitMQ consumer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start RabbitMQ consumer: %v", err)
	}

	// Setup Gin
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())

	// Initialize handlers
	taskHandler := handlers.NewTaskHandler(db)

	// Public routes
	router.GET("/health", taskHandler.Health)

	// Protected routes
	api := router.Group("/api/tasks")
	api.Use(middleware.AuthMiddleware(db))
	{
		api.POST("", taskHandler.CreateTask)
		api.GET("", taskHandler.GetTasks)
		api.GET("/:id", taskHandler.GetTask)
		api.PUT("/:id", taskHandler.UpdateTask)
		api.DELETE("/:id", taskHandler.DeleteTask)
		api.GET("/stats/summary", taskHandler.GetStats)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("🚀 Tasks Service is running on port %s\n", port)
		log.Printf("📝 Environment: %s\n", os.Getenv("ENV"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	cancel() // Stop RabbitMQ consumer

	log.Println("✅ Server exited gracefully")
}
