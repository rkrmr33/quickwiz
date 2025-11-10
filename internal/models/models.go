package models

import (
	"time"
)

// Quiz represents a parsed quiz from markdown
type Quiz struct {
	Title                string     `json:"title"`
	TimePerQuestion      int        `json:"time_per_question"`      // in seconds
	TimeBetweenQuestions int        `json:"time_between_questions"` // in seconds
	StreakBonus          bool       `json:"streak_bonus"`           // Enable streak bonus points
	QuickestAnswerBonus  bool       `json:"quickest_answer_bonus"`  // Give +1 point to first correct answer
	Questions            []Question `json:"questions"`
}

// Question represents a single quiz question
type Question struct {
	Text    string   `json:"text"`
	Options []string `json:"options"`
	Answer  string   `json:"answer"`
}

// QuizSession represents an active quiz session
type QuizSession struct {
	Code            string                  `json:"code"`
	Quiz            Quiz                    `json:"quiz"`
	Participants    map[string]*Participant `json:"participants"`
	CreatorID       string                  `json:"creator_id"` // ID of the participant who created the quiz (spectator)
	CurrentQuestion int                     `json:"current_question"`
	State           SessionState            `json:"state"`
	CreatedAt       time.Time               `json:"created_at"`
	QuestionStarted time.Time               `json:"question_started"`
}

// Participant represents a user in a quiz session
type Participant struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Score         int       `json:"score"`
	CurrentAnswer string    `json:"current_answer"`
	AnsweredAt    time.Time `json:"answered_at"`
	HasAnswered   bool      `json:"has_answered"`
	IsSpectator   bool      `json:"is_spectator"`   // True for the quiz creator
	CurrentStreak int       `json:"current_streak"` // Consecutive correct answers
	JoinedAt      time.Time `json:"joined_at"`      // When the participant joined
}

// SessionState represents the state of a quiz session
type SessionState string

const (
	StateWaiting    SessionState = "waiting"     // Waiting for participants
	StateInProgress SessionState = "in_progress" // Quiz in progress
	StateQuestion   SessionState = "question"    // Showing question
	StateAnswer     SessionState = "answer"      // Showing answer
	StateFinished   SessionState = "finished"    // Quiz finished
)

// WebSocketMessage represents messages sent via WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// QuestionUpdate sent to participants when a new question starts
type QuestionUpdate struct {
	QuestionNumber int      `json:"question_number"`
	TotalQuestions int      `json:"total_questions"`
	Text           string   `json:"text"`
	Options        []string `json:"options"`
	TimeRemaining  int      `json:"time_remaining"`
}

// AnswerReveal sent when answer is revealed
type AnswerReveal struct {
	CorrectAnswer string            `json:"correct_answer"`
	Participants  []ParticipantInfo `json:"participants"`
}

// ParticipantInfo for displaying participant status
type ParticipantInfo struct {
	Name                 string  `json:"name"`
	Answer               string  `json:"answer"`
	IsCorrect            bool    `json:"is_correct"`
	Score                int     `json:"score"`
	Streak               int     `json:"streak"`                 // Current streak count
	StreakBonus          int     `json:"streak_bonus"`           // Bonus points earned from streak
	QuickestAnswerFlag   bool    `json:"quickest_answer_flag"`   // True if this participant answered correctly first
	AnswerSubmissionTime float64 `json:"answer_submission_time"` // Time in seconds to submit answer (0 if not answered)
}

// ParticipantJoined sent when a new participant joins
type ParticipantJoined struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	IsSpectator      bool   `json:"is_spectator"`
	ParticipantCount int    `json:"participant_count"`
}

// QuizFinished sent when quiz is complete
type QuizFinished struct {
	Leaderboard []ParticipantInfo `json:"leaderboard"`
}

// AnswerCountUpdate sent when someone submits an answer
type AnswerCountUpdate struct {
	ParticipantID     string `json:"participant_id"`
	AnsweredCount     int    `json:"answered_count"`
	TotalParticipants int    `json:"total_participants"`
}
