package quiz

import (
	"testing"

	"github.com/rkrmr33/quickwiz/internal/models"
)

func TestCreateSession(t *testing.T) {
	manager := NewManager()
	quiz := models.Quiz{
		Title:           "Test Quiz",
		TimePerQuestion: 30,
		Questions: []models.Question{
			{
				Text:    "Question 1?",
				Options: []string{"A", "B", "C"},
				Answer:  "A",
			},
		},
	}

	code, err := manager.CreateSession(quiz)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if code == "" {
		t.Error("Expected non-empty code")
	}

	// Verify session exists
	session, err := manager.GetSession(code)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session.Quiz.Title != quiz.Title {
		t.Errorf("Expected title '%s', got '%s'", quiz.Title, session.Quiz.Title)
	}
}

func TestAddParticipant(t *testing.T) {
	manager := NewManager()
	quiz := models.Quiz{
		Title:           "Test Quiz",
		TimePerQuestion: 30,
		Questions: []models.Question{
			{
				Text:    "Question 1?",
				Options: []string{"A", "B", "C"},
				Answer:  "A",
			},
		},
	}

	code, _ := manager.CreateSession(quiz)

	// Add participant
	err := manager.AddParticipant(code, "p1", "Alice", false)
	if err != nil {
		t.Fatalf("Failed to add participant: %v", err)
	}

	session, _ := manager.GetSession(code)
	if len(session.Participants) != 1 {
		t.Errorf("Expected 1 participant, got %d", len(session.Participants))
	}

	p := session.Participants["p1"]
	if p.Name != "Alice" {
		t.Errorf("Expected participant name 'Alice', got '%s'", p.Name)
	}
}

func TestStartQuiz(t *testing.T) {
	manager := NewManager()
	quiz := models.Quiz{
		Title:           "Test Quiz",
		TimePerQuestion: 30,
		Questions: []models.Question{
			{
				Text:    "Question 1?",
				Options: []string{"A", "B", "C"},
				Answer:  "A",
			},
		},
	}

	code, _ := manager.CreateSession(quiz)
	manager.AddParticipant(code, "p1", "Alice", false)

	// Start quiz
	err := manager.StartQuiz(code)
	if err != nil {
		t.Fatalf("Failed to start quiz: %v", err)
	}

	session, _ := manager.GetSession(code)
	if session.State != models.StateQuestion {
		t.Errorf("Expected state 'question', got '%s'", session.State)
	}
	if session.CurrentQuestion != 0 {
		t.Errorf("Expected current question 0, got %d", session.CurrentQuestion)
	}
}

func TestSubmitAnswer(t *testing.T) {
	manager := NewManager()
	quiz := models.Quiz{
		Title:           "Test Quiz",
		TimePerQuestion: 30,
		Questions: []models.Question{
			{
				Text:    "Question 1?",
				Options: []string{"A", "B", "C"},
				Answer:  "A",
			},
		},
	}

	code, _ := manager.CreateSession(quiz)
	manager.AddParticipant(code, "p1", "Alice", false)
	manager.StartQuiz(code)

	// Submit answer
	err := manager.SubmitAnswer(code, "p1", "A")
	if err != nil {
		t.Fatalf("Failed to submit answer: %v", err)
	}

	session, _ := manager.GetSession(code)
	p := session.Participants["p1"]
	if p.CurrentAnswer != "A" {
		t.Errorf("Expected answer 'A', got '%s'", p.CurrentAnswer)
	}
	if !p.HasAnswered {
		t.Error("Expected HasAnswered to be true")
	}
}

func TestCheckAllAnswered(t *testing.T) {
	manager := NewManager()
	quiz := models.Quiz{
		Title:           "Test Quiz",
		TimePerQuestion: 30,
		Questions: []models.Question{
			{
				Text:    "Question 1?",
				Options: []string{"A", "B", "C"},
				Answer:  "A",
			},
		},
	}

	code, _ := manager.CreateSession(quiz)
	manager.AddParticipant(code, "p1", "Alice", false)
	manager.AddParticipant(code, "p2", "Bob", false)
	manager.StartQuiz(code)

	// Check before all answered
	if manager.CheckAllAnswered(code) {
		t.Error("Expected false before all answered")
	}

	// Submit first answer
	manager.SubmitAnswer(code, "p1", "A")
	if manager.CheckAllAnswered(code) {
		t.Error("Expected false with only one answer")
	}

	// Submit second answer
	manager.SubmitAnswer(code, "p2", "B")
	if !manager.CheckAllAnswered(code) {
		t.Error("Expected true after all answered")
	}
}

func TestRevealAnswer(t *testing.T) {
	manager := NewManager()
	quiz := models.Quiz{
		Title:           "Test Quiz",
		TimePerQuestion: 30,
		Questions: []models.Question{
			{
				Text:    "Question 1?",
				Options: []string{"A", "B", "C"},
				Answer:  "A",
			},
		},
	}

	code, _ := manager.CreateSession(quiz)
	manager.AddParticipant(code, "p1", "Alice", false)
	manager.AddParticipant(code, "p2", "Bob", false)
	manager.StartQuiz(code)
	manager.SubmitAnswer(code, "p1", "A")
	manager.SubmitAnswer(code, "p2", "B")

	// Reveal answer
	reveal, err := manager.RevealAnswer(code)
	if err != nil {
		t.Fatalf("Failed to reveal answer: %v", err)
	}

	if reveal.CorrectAnswer != "A" {
		t.Errorf("Expected correct answer 'A', got '%s'", reveal.CorrectAnswer)
	}

	// Check scores
	session, _ := manager.GetSession(code)
	if session.Participants["p1"].Score != 1 {
		t.Errorf("Expected Alice's score to be 1, got %d", session.Participants["p1"].Score)
	}
	if session.Participants["p2"].Score != 0 {
		t.Errorf("Expected Bob's score to be 0, got %d", session.Participants["p2"].Score)
	}
}

func TestNextQuestion(t *testing.T) {
	manager := NewManager()
	quiz := models.Quiz{
		Title:           "Test Quiz",
		TimePerQuestion: 30,
		Questions: []models.Question{
			{
				Text:    "Question 1?",
				Options: []string{"A", "B", "C"},
				Answer:  "A",
			},
			{
				Text:    "Question 2?",
				Options: []string{"A", "B", "C"},
				Answer:  "B",
			},
		},
	}

	code, _ := manager.CreateSession(quiz)
	manager.AddParticipant(code, "p1", "Alice", false)
	manager.StartQuiz(code)
	manager.SubmitAnswer(code, "p1", "A")
	manager.RevealAnswer(code)

	// Move to next question
	hasNext, err := manager.NextQuestion(code)
	if err != nil {
		t.Fatalf("Failed to move to next question: %v", err)
	}
	if !hasNext {
		t.Error("Expected hasNext to be true")
	}

	session, _ := manager.GetSession(code)
	if session.CurrentQuestion != 1 {
		t.Errorf("Expected current question 1, got %d", session.CurrentQuestion)
	}
	if session.State != models.StateQuestion {
		t.Errorf("Expected state 'question', got '%s'", session.State)
	}

	// Move past last question
	manager.SubmitAnswer(code, "p1", "B")
	manager.RevealAnswer(code)
	hasNext, _ = manager.NextQuestion(code)
	if hasNext {
		t.Error("Expected hasNext to be false at end")
	}

	session, _ = manager.GetSession(code)
	if session.State != models.StateFinished {
		t.Errorf("Expected state 'finished', got '%s'", session.State)
	}
}

func TestGetLeaderboard(t *testing.T) {
	manager := NewManager()
	quiz := models.Quiz{
		Title:           "Test Quiz",
		TimePerQuestion: 30,
		Questions: []models.Question{
			{
				Text:    "Question 1?",
				Options: []string{"A", "B", "C"},
				Answer:  "A",
			},
		},
	}

	code, _ := manager.CreateSession(quiz)
	manager.AddParticipant(code, "p1", "Alice", false)
	manager.AddParticipant(code, "p2", "Bob", false)
	manager.AddParticipant(code, "p3", "Charlie", false)
	manager.StartQuiz(code)

	// Set scores manually for testing
	session, _ := manager.GetSession(code)
	session.Participants["p1"].Score = 10
	session.Participants["p2"].Score = 5
	session.Participants["p3"].Score = 15

	// Get leaderboard
	leaderboard, err := manager.GetLeaderboard(code)
	if err != nil {
		t.Fatalf("Failed to get leaderboard: %v", err)
	}

	if len(leaderboard) != 3 {
		t.Errorf("Expected 3 participants, got %d", len(leaderboard))
	}

	// Check sorting (descending)
	if leaderboard[0].Name != "Charlie" || leaderboard[0].Score != 15 {
		t.Errorf("Expected Charlie with score 15 first, got %s with score %d",
			leaderboard[0].Name, leaderboard[0].Score)
	}
	if leaderboard[1].Name != "Alice" || leaderboard[1].Score != 10 {
		t.Errorf("Expected Alice with score 10 second, got %s with score %d",
			leaderboard[1].Name, leaderboard[1].Score)
	}
	if leaderboard[2].Name != "Bob" || leaderboard[2].Score != 5 {
		t.Errorf("Expected Bob with score 5 third, got %s with score %d",
			leaderboard[2].Name, leaderboard[2].Score)
	}
}
