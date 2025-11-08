package parser

import (
	"testing"
)

func TestParseQuizMarkdown(t *testing.T) {
	markdown := `# My Quiz Title

# Settings
time_per_question: 10 seconds


### What is the capital of France?
- Berlin
- Madrid
- Paris
- Rome
* Answer: Paris
    
### What is 2 + 2?
- 3
- 4
- 5
- 6
* Answer: 4

### What is the largest mammal in the world?
- Elephant
- Blue Whale
- Great White Shark
- Giraffe
* Answer: Blue Whale`

	quiz, err := ParseQuizMarkdown(markdown)
	if err != nil {
		t.Fatalf("Failed to parse quiz: %v", err)
	}

	// Check title
	if quiz.Title != "My Quiz Title" {
		t.Errorf("Expected title 'My Quiz Title', got '%s'", quiz.Title)
	}

	// Check time per question
	if quiz.TimePerQuestion != 10 {
		t.Errorf("Expected time_per_question 10, got %d", quiz.TimePerQuestion)
	}

	// Check number of questions
	if len(quiz.Questions) != 3 {
		t.Errorf("Expected 3 questions, got %d", len(quiz.Questions))
	}

	// Check first question
	q1 := quiz.Questions[0]
	if q1.Text != "What is the capital of France?" {
		t.Errorf("Expected question text 'What is the capital of France?', got '%s'", q1.Text)
	}
	if len(q1.Options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(q1.Options))
	}
	if q1.Answer != "Paris" {
		t.Errorf("Expected answer 'Paris', got '%s'", q1.Answer)
	}

	// Check second question
	q2 := quiz.Questions[1]
	if q2.Text != "What is 2 + 2?" {
		t.Errorf("Expected question text 'What is 2 + 2?', got '%s'", q2.Text)
	}
	if q2.Answer != "4" {
		t.Errorf("Expected answer '4', got '%s'", q2.Answer)
	}

	// Check third question
	q3 := quiz.Questions[2]
	if q3.Text != "What is the largest mammal in the world?" {
		t.Errorf("Expected question text 'What is the largest mammal in the world?', got '%s'", q3.Text)
	}
	if q3.Answer != "Blue Whale" {
		t.Errorf("Expected answer 'Blue Whale', got '%s'", q3.Answer)
	}
}

func TestParseQuizMarkdown_NoTitle(t *testing.T) {
	markdown := `### Question 1?
- Option A
- Option B
* Answer: Option A`

	_, err := ParseQuizMarkdown(markdown)
	if err == nil {
		t.Error("Expected error for missing title, got nil")
	}
}

func TestParseQuizMarkdown_NoQuestions(t *testing.T) {
	markdown := `# My Quiz`

	_, err := ParseQuizMarkdown(markdown)
	if err == nil {
		t.Error("Expected error for missing questions, got nil")
	}
}

func TestParseQuizMarkdown_InvalidAnswer(t *testing.T) {
	markdown := `# My Quiz

### Question 1?
- Option A
- Option B
* Answer: Option C`

	_, err := ParseQuizMarkdown(markdown)
	if err == nil {
		t.Error("Expected error for invalid answer, got nil")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"10 seconds", 10, false},
		{"1 minute", 60, false},
		{"2 minutes", 120, false},
		{"30s", 30, false},
		{"1m", 60, false},
		{"45", 45, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		result, err := parseDuration(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("Expected error for input '%s', got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("For input '%s', expected %d, got %d", tt.input, tt.expected, result)
			}
		}
	}
}
