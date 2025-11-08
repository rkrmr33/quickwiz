package quiz

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rkrmr33/quickwiz/internal/models"
)

// Manager handles quiz sessions
type Manager struct {
	sessions map[string]*models.QuizSession
	mu       sync.RWMutex
}

// NewManager creates a new quiz manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*models.QuizSession),
	}
}

// CreateSession creates a new quiz session from a quiz
func (m *Manager) CreateSession(quiz models.Quiz) (string, error) {
	code := generateCode()

	m.mu.Lock()
	defer m.mu.Unlock()

	session := &models.QuizSession{
		Code:            code,
		Quiz:            quiz,
		Participants:    make(map[string]*models.Participant),
		CurrentQuestion: -1,
		State:           models.StateWaiting,
		CreatedAt:       time.Now(),
	}

	m.sessions[code] = session
	return code, nil
}

// GetSession retrieves a quiz session by code
func (m *Manager) GetSession(code string) (*models.QuizSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[code]
	if !exists {
		return nil, fmt.Errorf("quiz session not found")
	}

	return session, nil
}

// AddParticipant adds a participant to a session
func (m *Manager) AddParticipant(code, participantID, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[code]
	if !exists {
		return fmt.Errorf("quiz session not found")
	}

	if session.State != models.StateWaiting {
		return fmt.Errorf("quiz has already started")
	}

	// Check if this is the first participant (creator/spectator)
	isSpectator := len(session.Participants) == 0
	if isSpectator {
		session.CreatorID = participantID
	}

	participant := &models.Participant{
		ID:          participantID,
		Name:        name,
		Score:       0,
		IsSpectator: isSpectator,
		JoinedAt:    time.Now(),
	}

	session.Participants[participantID] = participant
	return nil
}

// StartQuiz starts the quiz and moves to the first question
func (m *Manager) StartQuiz(code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[code]
	if !exists {
		return fmt.Errorf("quiz session not found")
	}

	if session.State != models.StateWaiting {
		return fmt.Errorf("quiz already started")
	}

	if len(session.Participants) == 0 {
		return fmt.Errorf("no participants in quiz")
	}

	session.State = models.StateInProgress
	session.CurrentQuestion = 0
	session.QuestionStarted = time.Now()
	session.State = models.StateQuestion

	// Reset all participants' answers
	for _, p := range session.Participants {
		p.HasAnswered = false
		p.CurrentAnswer = ""
	}

	return nil
}

// SubmitAnswer submits an answer for a participant
func (m *Manager) SubmitAnswer(code, participantID, answer string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[code]
	if !exists {
		return fmt.Errorf("quiz session not found")
	}

	if session.State != models.StateQuestion {
		return fmt.Errorf("not accepting answers right now")
	}

	participant, exists := session.Participants[participantID]
	if !exists {
		return fmt.Errorf("participant not found")
	}

	if participant.HasAnswered {
		return fmt.Errorf("already answered this question")
	}

	participant.CurrentAnswer = answer
	participant.HasAnswered = true
	participant.AnsweredAt = time.Now()

	return nil
}

// CheckAllAnswered checks if all participants have answered (excluding spectators)
func (m *Manager) CheckAllAnswered(code string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[code]
	if !exists {
		return false
	}

	for _, p := range session.Participants {
		// Skip spectators
		if p.IsSpectator {
			continue
		}
		if !p.HasAnswered {
			return false
		}
	}

	return true
}

// GetAnswerCount returns the number of participants who have answered (excluding spectators)
func (m *Manager) GetAnswerCount(code string) (int, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[code]
	if !exists {
		return 0, 0
	}

	answeredCount := 0
	totalParticipants := 0
	for _, p := range session.Participants {
		// Skip spectators
		if p.IsSpectator {
			continue
		}
		totalParticipants++
		if p.HasAnswered {
			answeredCount++
		}
	}

	return answeredCount, totalParticipants
}

// RevealAnswer reveals the answer and updates scores
func (m *Manager) RevealAnswer(code string) (*models.AnswerReveal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[code]
	if !exists {
		return nil, fmt.Errorf("quiz session not found")
	}

	if session.State != models.StateQuestion {
		return nil, fmt.Errorf("not in question state")
	}

	currentQ := session.Quiz.Questions[session.CurrentQuestion]
	session.State = models.StateAnswer

	participants := make([]models.ParticipantInfo, 0, len(session.Participants))
	for _, p := range session.Participants {
		// Skip spectators in results
		if p.IsSpectator {
			continue
		}

		isCorrect := p.CurrentAnswer == currentQ.Answer
		streakBonus := 0

		if isCorrect {
			// Increment streak
			p.CurrentStreak++

			// Base point for correct answer
			p.Score++

			// Calculate streak bonus if enabled
			if session.Quiz.StreakBonus {
				streakBonus = calculateStreakBonus(p.CurrentStreak)
				p.Score += streakBonus
			}
		} else {
			// Reset streak on wrong answer
			p.CurrentStreak = 0
		}

		participants = append(participants, models.ParticipantInfo{
			Name:        p.Name,
			Answer:      p.CurrentAnswer,
			IsCorrect:   isCorrect,
			Score:       p.Score,
			Streak:      p.CurrentStreak,
			StreakBonus: streakBonus,
		})
	}

	return &models.AnswerReveal{
		CorrectAnswer: currentQ.Answer,
		Participants:  participants,
	}, nil
}

// NextQuestion moves to the next question or finishes the quiz
func (m *Manager) NextQuestion(code string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[code]
	if !exists {
		return false, fmt.Errorf("quiz session not found")
	}

	if session.State != models.StateAnswer {
		return false, fmt.Errorf("not in answer state")
	}

	// Check if there are more questions
	if session.CurrentQuestion+1 >= len(session.Quiz.Questions) {
		session.State = models.StateFinished
		return false, nil
	}

	// Move to next question
	session.CurrentQuestion++
	session.State = models.StateQuestion
	session.QuestionStarted = time.Now()

	// Reset all participants' answers
	for _, p := range session.Participants {
		p.HasAnswered = false
		p.CurrentAnswer = ""
	}

	return true, nil
}

// GetLeaderboard returns the final leaderboard (excluding spectators)
func (m *Manager) GetLeaderboard(code string) ([]models.ParticipantInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[code]
	if !exists {
		return nil, fmt.Errorf("quiz session not found")
	}

	participants := make([]models.ParticipantInfo, 0, len(session.Participants))
	for _, p := range session.Participants {
		// Skip spectators in leaderboard
		if p.IsSpectator {
			continue
		}
		participants = append(participants, models.ParticipantInfo{
			Name:  p.Name,
			Score: p.Score,
		})
	}

	// Sort by score (descending)
	for i := 0; i < len(participants)-1; i++ {
		for j := i + 1; j < len(participants); j++ {
			if participants[j].Score > participants[i].Score {
				participants[i], participants[j] = participants[j], participants[i]
			}
		}
	}

	return participants, nil
}

// CleanupOldSessions removes sessions older than 24 hours
func (m *Manager) CleanupOldSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for code, session := range m.sessions {
		if session.CreatedAt.Before(cutoff) {
			delete(m.sessions, code)
		}
	}
}

// generateCode generates a random 8-character
func generateCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return strings.ToUpper(hex.EncodeToString(bytes))
}

// calculateStreakBonus calculates bonus points based on current streak
// 3+ correct: +1 point per answer
// 5+ correct: +2 points per answer
// 10+ correct: +5 points per answer
func calculateStreakBonus(streak int) int {
	if streak >= 10 {
		return 5
	} else if streak >= 5 {
		return 2
	} else if streak >= 3 {
		return 1
	}
	return 0
}
