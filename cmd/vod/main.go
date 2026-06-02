package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"test_vod/config"
	"test_vod/handler"
	"test_vod/s3"
	"test_vod/store"
)

func main() {
	cfg := config.LoadConfig()

	// Ensure data dir exists
	os.MkdirAll(cfg.DataDir, 0755)

	// Init SQLite Store
	dbPath := cfg.DataDir + "/vod.db"
	sqlStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqlStore.Close()

	// Init S3 Client
	s3Client, err := s3.NewClient(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

	// Init Handlers
	videoHandler := handler.NewVideoHandler(sqlStore, s3Client, cfg)
	folderHandler := handler.NewFolderHandler(sqlStore)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Simple CORS for test frontend
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// API Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/videos", func(r chi.Router) {
			r.Get("/", videoHandler.List)
			r.Post("/upload", videoHandler.Upload)
			r.Get("/{id}/stream", videoHandler.Stream)
			r.Delete("/{id}", videoHandler.Delete)
		})

		r.Route("/folders", func(r chi.Router) {
			r.Get("/", folderHandler.List)
			r.Post("/", folderHandler.Create)
			r.Delete("/{id}", folderHandler.Delete)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Serve the test frontend
	workDir, _ := os.Getwd()
	frontendDir := http.Dir(workDir + "/frontend")
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for root
		if r.URL.Path == "/" {
			http.ServeFile(w, r, workDir+"/frontend/index.html")
			return
		}
		http.FileServer(frontendDir).ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: r,
	}

	go func() {
		log.Printf("Starting VOD service on %s", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
