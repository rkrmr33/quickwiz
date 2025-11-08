package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/rkrmr33/quickwiz/internal/models"
	"github.com/rkrmr33/quickwiz/internal/parser"
	"github.com/rkrmr33/quickwiz/internal/quiz"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Handler manages HTTP requests
type Handler struct {
	quizManager *quiz.Manager
	templates   *template.Template
	connections map[string]map[*websocket.Conn]string // quizCode -> conn -> participantID
	connMu      sync.RWMutex
}

// NewHandler creates a new HTTP handler
func NewHandler(quizManager *quiz.Manager, templates *template.Template) *Handler {
	return &Handler{
		quizManager: quizManager,
		templates:   templates,
		connections: make(map[string]map[*websocket.Conn]string),
	}
}

// HomeHandler serves the home page
func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	h.templates.ExecuteTemplate(w, "index.html", nil)
}

// NotFoundHandler serves the 404 page
func (h *Handler) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.templates.ExecuteTemplate(w, "404.html", nil)
}

// CreateQuizHandler handles quiz creation
func (h *Handler) CreateQuizHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("CreateQuiz request received", "remote_addr", r.RemoteAddr, "content_type", r.Header.Get("Content-Type"))

	if r.Method != http.MethodPost {
		slog.Warn("CreateQuiz invalid method", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("CreateQuiz failed to decode JSON body", "error", err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	markdown := req.Markdown

	slog.Info("CreateQuiz markdown received", "length", len(markdown))

	// Parse markdown
	quiz, err := parser.ParseQuizMarkdown(markdown)
	if err != nil {
		slog.Error("CreateQuiz failed to parse markdown", "error", err)
		http.Error(w, fmt.Sprintf("Failed to parse quiz: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("CreateQuiz quiz parsed successfully",
		"title", quiz.Title,
		"questions", len(quiz.Questions),
		"time_per_question", quiz.TimePerQuestion)

	// Create session
	code, err := h.quizManager.CreateSession(*quiz)
	if err != nil {
		slog.Error("CreateQuiz failed to create session", "error", err)
		http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("CreateQuiz quiz created successfully", "code", code)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"code": code,
	})
}

// GetQuizHandler returns quiz information
func (h *Handler) GetQuizHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := cleanCode(vars["code"])

	slog.Info("GetQuiz request received", "code", code)

	session, err := h.quizManager.GetSession(code)
	if err != nil {
		slog.Warn("GetQuiz session not found", "code", code)
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	// Build participant list sorted by join time
	participantList := make([]*models.Participant, 0, len(session.Participants))
	for _, p := range session.Participants {
		participantList = append(participantList, p)
	}

	// Sort by JoinedAt (earliest first)
	for i := 0; i < len(participantList); i++ {
		for j := i + 1; j < len(participantList); j++ {
			if participantList[j].JoinedAt.Before(participantList[i].JoinedAt) {
				participantList[i], participantList[j] = participantList[j], participantList[i]
			}
		}
	}

	participants := make([]map[string]interface{}, 0, len(participantList))
	for _, p := range participantList {
		participants = append(participants, map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"isSpectator": p.IsSpectator,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":             code,
		"title":            session.Quiz.Title,
		"questionCount":    len(session.Quiz.Questions),
		"participantCount": len(session.Participants),
		"participants":     participants,
		"creatorId":        session.CreatorID,
		"state":            session.State,
	})
}

// JoinPageHandler serves the join page
func (h *Handler) JoinPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := cleanCode(vars["code"])

	// Verify session exists
	session, err := h.quizManager.GetSession(code)
	if err != nil {
		h.NotFoundHandler(w, r)
		return
	}

	h.templates.ExecuteTemplate(w, "join.html", map[string]interface{}{
		"Code":  code,
		"Title": session.Quiz.Title,
	})
}

// JoinQuizHandler handles participant joining
func (h *Handler) JoinQuizHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := cleanCode(vars["code"])

	slog.Info("JoinQuiz request received", "code", code, "remote_addr", r.RemoteAddr)

	if r.Method != http.MethodPost {
		slog.Warn("JoinQuiz invalid method", "method", r.Method, "code", code)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string `json:"name"`
		IsSpectator bool   `json:"is_spectator"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("JoinQuiz failed to decode request body", "error", err, "code", code)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		slog.Warn("JoinQuiz empty name provided", "code", code)
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	slog.Info("JoinQuiz participant attempting to join", "name", req.Name, "is_spectator", req.IsSpectator, "code", code)

	// Check if participant with this name already exists (for reconnection)
	session, err := h.quizManager.GetSession(code)
	if err != nil {
		slog.Error("JoinQuiz failed to get session", "error", err, "code", code)
		http.Error(w, fmt.Sprintf("Failed to join quiz: %v", err), http.StatusBadRequest)
		return
	}

	var participantID string
	var isRejoining bool

	// Check if a participant with this name already exists
	for id, p := range session.Participants {
		if p.Name == req.Name {
			participantID = id
			isRejoining = true
			slog.Info("JoinQuiz participant rejoining", "name", req.Name, "participant_id", participantID, "code", code)
			break
		}
	}

	// If not rejoining, create a new participant
	if !isRejoining {
		participantID = generateParticipantID()
		err = h.quizManager.AddParticipant(code, participantID, req.Name, req.IsSpectator)
		if err != nil {
			slog.Error("JoinQuiz failed to add participant", "error", err, "name", req.Name, "code", code)
			http.Error(w, fmt.Sprintf("Failed to join quiz: %v", err), http.StatusBadRequest)
			return
		}
		slog.Info("JoinQuiz participant joined successfully", "name", req.Name, "is_spectator", req.IsSpectator, "participant_id", participantID, "code", code)
	}

	// Refresh session after potential addition
	session, err = h.quizManager.GetSession(code)
	if err != nil {
		slog.Error("JoinQuiz failed to get session", "error", err, "code", code)
		http.Error(w, fmt.Sprintf("Failed to join quiz: %v", err), http.StatusBadRequest)
		return
	}

	// Get the participant to check if they're a spectator
	participant := session.Participants[participantID]

	// Only broadcast participant joined if this is a new participant (not rejoining)
	if !isRejoining {
		h.broadcast(code, models.WebSocketMessage{
			Type: "participant_joined",
			Payload: models.ParticipantJoined{
				ID:               participantID,
				Name:             req.Name,
				IsSpectator:      participant.IsSpectator,
				ParticipantCount: len(session.Participants),
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"participant_id": participantID,
	})
}

// QuizPageHandler serves the quiz page
func (h *Handler) QuizPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := cleanCode(vars["code"])
	participantID := vars["participant_id"]

	session, err := h.quizManager.GetSession(code)
	if err != nil {
		h.NotFoundHandler(w, r)
		return
	}

	h.templates.ExecuteTemplate(w, "quiz.html", map[string]interface{}{
		"Code":          code,
		"Title":         session.Quiz.Title,
		"ParticipantID": participantID,
	})
}

// StartQuizHandler starts the quiz
func (h *Handler) StartQuizHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := cleanCode(vars["code"])

	slog.Info("StartQuiz request received", "code", code, "remote_addr", r.RemoteAddr)

	if r.Method != http.MethodPost {
		slog.Warn("StartQuiz invalid method", "method", r.Method, "code", code)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get participant_id from query or body
	var req struct {
		ParticipantID string `json:"participant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("StartQuiz failed to decode request", "error", err, "code", code)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check if the participant is the creator
	session, err := h.quizManager.GetSession(code)
	if err != nil {
		slog.Error("StartQuiz failed to get session", "error", err, "code", code)
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	if session.CreatorID != req.ParticipantID {
		slog.Warn("StartQuiz unauthorized attempt", "participant_id", req.ParticipantID, "creator_id", session.CreatorID, "code", code)
		http.Error(w, "Only the quiz creator can start the quiz", http.StatusForbidden)
		return
	}

	err = h.quizManager.StartQuiz(code)
	if err != nil {
		slog.Error("StartQuiz failed to start quiz", "error", err, "code", code)
		http.Error(w, fmt.Sprintf("Failed to start quiz: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("StartQuiz quiz started successfully", "code", code, "creator_id", req.ParticipantID)

	// Start question timer
	go h.runQuestionTimer(code)

	w.WriteHeader(http.StatusOK)
}

// SubmitAnswerHandler handles answer submission
func (h *Handler) SubmitAnswerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := cleanCode(vars["code"])

	slog.Info("SubmitAnswer request received", "code", code, "remote_addr", r.RemoteAddr)

	if r.Method != http.MethodPost {
		slog.Warn("SubmitAnswer invalid method", "method", r.Method, "code", code)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ParticipantID string `json:"participant_id"`
		Answer        string `json:"answer"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("SubmitAnswer failed to decode request body", "error", err, "code", code)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("SubmitAnswer processing answer", "code", code, "participant_id", req.ParticipantID, "answer", req.Answer)

	err := h.quizManager.SubmitAnswer(code, req.ParticipantID, req.Answer)
	if err != nil {
		slog.Error("SubmitAnswer failed to submit answer", "error", err, "code", code, "participant_id", req.ParticipantID)
		http.Error(w, fmt.Sprintf("Failed to submit answer: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("SubmitAnswer answer submitted successfully", "code", code, "participant_id", req.ParticipantID)

	// Broadcast answer count update
	answeredCount, totalParticipants := h.quizManager.GetAnswerCount(code)
	h.broadcast(code, models.WebSocketMessage{
		Type: "answer_count_update",
		Payload: models.AnswerCountUpdate{
			AnsweredCount:     answeredCount,
			TotalParticipants: totalParticipants,
		},
	})

	// Check if all answered
	if h.quizManager.CheckAllAnswered(code) {
		go h.revealAnswer(code)
	}

	w.WriteHeader(http.StatusOK)
}

// WebSocketHandler handles WebSocket connections
func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := cleanCode(vars["code"])
	participantID := r.URL.Query().Get("participant_id")

	slog.Info("WebSocket connection request", "code", code, "participant_id", participantID, "remote_addr", r.RemoteAddr)

	// Verify session exists
	_, err := h.quizManager.GetSession(code)
	if err != nil {
		slog.Error("WebSocket quiz not found", "error", err, "code", code)
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err, "code", code, "participant_id", participantID)
		return
	}

	slog.Info("WebSocket connection established", "code", code, "participant_id", participantID)

	// Register connection
	h.connMu.Lock()
	if h.connections[code] == nil {
		h.connections[code] = make(map[*websocket.Conn]string)
	}
	h.connections[code][conn] = participantID
	h.connMu.Unlock()

	// Send current state
	session, _ := h.quizManager.GetSession(code)
	if session.State == models.StateQuestion {
		h.sendQuestionUpdate(conn, code)
	}

	// Handle disconnection
	defer func() {
		h.connMu.Lock()
		delete(h.connections[code], conn)
		h.connMu.Unlock()
		conn.Close()
	}()

	// Keep connection alive (read messages to detect disconnection)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Helper methods

func (h *Handler) runQuestionTimer(code string) {
	session, err := h.quizManager.GetSession(code)
	if err != nil {
		return
	}

	// Send question to all participants
	h.broadcast(code, models.WebSocketMessage{
		Type:    "question",
		Payload: h.buildQuestionUpdate(session),
	})

	// Wait for time or all answers
	duration := time.Duration(session.Quiz.TimePerQuestion) * time.Second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	for {
		<-ticker.C
		elapsed := time.Since(startTime)

		if elapsed >= duration {
			h.revealAnswer(code)
			break
		}

		// Check if all answered
		if h.quizManager.CheckAllAnswered(code) {
			h.revealAnswer(code)
			break
		}

		// Send time update
		remaining := int(duration.Seconds() - elapsed.Seconds())
		h.broadcast(code, models.WebSocketMessage{
			Type: "time_update",
			Payload: map[string]int{
				"time_remaining": remaining,
			},
		})
	}
}

func (h *Handler) revealAnswer(code string) {
	slog.Info("Revealing answer", "code", code)

	reveal, err := h.quizManager.RevealAnswer(code)
	if err != nil {
		slog.Error("Error revealing answer", "error", err, "code", code)
		return
	}

	// Get session to access time_between_questions setting
	session, err := h.quizManager.GetSession(code)
	if err != nil {
		slog.Error("Error getting session", "error", err, "code", code)
		return
	}

	// Broadcast answer reveal
	h.broadcast(code, models.WebSocketMessage{
		Type:    "answer_reveal",
		Payload: reveal,
	})

	slog.Info("Answer revealed successfully", "code", code)

	// Wait before next question based on quiz settings, sending timer updates
	duration := time.Duration(session.Quiz.TimeBetweenQuestions) * time.Second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	for {
		<-ticker.C
		elapsed := time.Since(startTime)

		if elapsed >= duration {
			break
		}

		// Send time update during answer reveal
		remaining := int(duration.Seconds() - elapsed.Seconds())
		h.broadcast(code, models.WebSocketMessage{
			Type: "time_update",
			Payload: map[string]int{
				"time_remaining": remaining,
			},
		})
	}

	hasNext, err := h.quizManager.NextQuestion(code)
	if err != nil {
		slog.Error("Error moving to next question", "error", err, "code", code)
		return
	}

	if !hasNext {
		// Quiz finished
		leaderboard, _ := h.quizManager.GetLeaderboard(code)
		h.broadcast(code, models.WebSocketMessage{
			Type: "quiz_finished",
			Payload: models.QuizFinished{
				Leaderboard: leaderboard,
			},
		})
	} else {
		// Start next question
		go h.runQuestionTimer(code)
	}
}

func (h *Handler) buildQuestionUpdate(session *models.QuizSession) models.QuestionUpdate {
	q := session.Quiz.Questions[session.CurrentQuestion]
	return models.QuestionUpdate{
		QuestionNumber: session.CurrentQuestion + 1,
		TotalQuestions: len(session.Quiz.Questions),
		Text:           q.Text,
		Options:        q.Options,
		TimeRemaining:  session.Quiz.TimePerQuestion,
	}
}

func (h *Handler) sendQuestionUpdate(conn *websocket.Conn, code string) {
	session, err := h.quizManager.GetSession(code)
	if err != nil {
		return
	}

	msg := models.WebSocketMessage{
		Type:    "question",
		Payload: h.buildQuestionUpdate(session),
	}

	conn.WriteJSON(msg)
}

func (h *Handler) broadcast(code string, msg models.WebSocketMessage) {
	h.connMu.RLock()
	defer h.connMu.RUnlock()

	conns := h.connections[code]
	for conn := range conns {
		if err := conn.WriteJSON(msg); err != nil {
			slog.Error("Error broadcasting message", "error", err, "code", code, "msg_type", msg.Type)
		}
	}
}

func generateParticipantID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func cleanCode(s string) string {
	return strings.TrimSpace(strings.ToUpper(s))
}
