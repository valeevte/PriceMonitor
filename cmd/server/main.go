package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/valeevte/PriceMonitor/internal/database"
	"github.com/valeevte/PriceMonitor/internal/products"
	"github.com/valeevte/PriceMonitor/internal/scheduler"

	"github.com/gin-gonic/gin"
)

func main() {
	_ = godotenv.Load() // load .env if present; not fatal if missing

	// connect to DB
	database.Connect()

	// graceful shutdown coordination
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// start scheduler
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// scheduler runs until ctx is cancelled
		scheduler.Run(ctx, database.DB, scheduler.Config{IntervalSeconds: 60})
	}()

	// build router and handlers
	repo := products.NewRepository(database.DB)
	h := products.NewHandler(repo)

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(os.Getenv("GIN_MODE"))
	}
	r := gin.Default()

	api := r.Group("/api")
	{
		api.GET("/products", h.ListProducts)
		api.POST("/products", h.CreateProduct)
		api.GET("/products/:id", h.GetProduct)
		api.GET("/products/:id/history", h.GetPriceHistory)
	}

	// static files
	r.Static("/static", "./web")
	r.StaticFile("/", "./web/index.html")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// start server
	go func() {
		log.Printf("Server started on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server ListenAndServe: %v", err)
		}
	}()

	// wait for interrupt
	<-ctx.Done()
	log.Println("shutdown signal received")

	// stop accepting new requests, allow 15s to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server Shutdown: %v", err)
	}

	// wait scheduler to finish (it reacts to ctx)
	wg.Wait()

	// close DB pool (blocks until connections returned)
	database.DB.Close()

	log.Println("graceful shutdown complete")
}
