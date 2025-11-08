package main

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rkrmr33/quickwiz/internal/handlers"
	"github.com/rkrmr33/quickwiz/internal/quiz"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting QuicKwiz server")

	// Initialize quiz manager
	quizManager := quiz.NewManager()
	slog.Info("Quiz manager initialized")

	// Load templates
	templates := template.Must(template.ParseGlob("web/templates/*.html"))
	slog.Info("Templates loaded successfully")

	// Initialize handlers
	handler := handlers.NewHandler(quizManager, templates)

	// Setup router
	r := mux.NewRouter()

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Web routes
	r.HandleFunc("/", handler.HomeHandler).Methods("GET")
	r.HandleFunc("/quiz/{code}", handler.JoinPageHandler).Methods("GET")
	r.HandleFunc("/quiz/{code}/play/{participant_id}", handler.QuizPageHandler).Methods("GET")

	// API routes
	r.HandleFunc("/api/quiz", handler.CreateQuizHandler).Methods("POST")
	r.HandleFunc("/api/quiz/{code}", handler.GetQuizHandler).Methods("GET")
	r.HandleFunc("/api/quiz/{code}/join", handler.JoinQuizHandler).Methods("POST")
	r.HandleFunc("/api/quiz/{code}/start", handler.StartQuizHandler).Methods("POST")
	r.HandleFunc("/api/quiz/{code}/answer", handler.SubmitAnswerHandler).Methods("POST")

	// WebSocket route
	r.HandleFunc("/ws/{code}", handler.WebSocketHandler)

	// 404 handler
	r.NotFoundHandler = http.HandlerFunc(handler.NotFoundHandler)

	slog.Info("Routes configured")

	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			slog.Info("Running session cleanup")
			quizManager.CleanupOldSessions()
		}
	}()

	// Start server
	port := "8080"
	slog.Info("QuicKwiz server starting", "port", port, "url", "http://localhost:"+port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
