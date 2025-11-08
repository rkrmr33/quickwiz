package parser

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rkrmr33/quickwiz/internal/models"
)

// ParseQuizMarkdown parses a markdown string into a Quiz struct
func ParseQuizMarkdown(markdown string) (*models.Quiz, error) {
	quiz := &models.Quiz{
		TimePerQuestion:      30, // default 30 seconds
		TimeBetweenQuestions: 5,  // default 5 seconds
		Questions:            []models.Question{},
	}

	scanner := bufio.NewScanner(strings.NewReader(markdown))
	var currentQuestion *models.Question
	inSettings := false
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Settings section (check BEFORE title to avoid "Settings" being treated as title)
		if trimmed == "# Settings" {
			inSettings = true
			continue
		}

		// Parse title (first # line)
		if strings.HasPrefix(trimmed, "# ") && quiz.Title == "" && !inSettings {
			quiz.Title = strings.TrimPrefix(trimmed, "# ")
			continue
		}

		// Parse settings
		if inSettings {
			if strings.HasPrefix(trimmed, "#") {
				inSettings = false
			} else if strings.Contains(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				if key == "time_per_question" {
					// Parse duration (e.g., "10 seconds", "1 minute")
					timeVal, err := parseDuration(value)
					if err == nil {
						quiz.TimePerQuestion = timeVal
					}
				} else if key == "time_between_questions" {
					// Parse duration (e.g., "10 seconds", "1 minute")
					timeVal, err := parseDuration(value)
					if err == nil {
						quiz.TimeBetweenQuestions = timeVal
					}
				} else if key == "streak_bonus" {
					// Parse boolean (e.g., "true", "false", "yes", "no")
					quiz.StreakBonus = parseBool(value)
				}
				continue
			}
		}

		// Parse question (### prefix)
		if strings.HasPrefix(trimmed, "###") {
			// Save previous question if exists
			if currentQuestion != nil && currentQuestion.Text != "" {
				quiz.Questions = append(quiz.Questions, *currentQuestion)
			}
			currentQuestion = &models.Question{
				Text:    strings.TrimSpace(strings.TrimPrefix(trimmed, "###")),
				Options: []string{},
			}
			continue
		}

		// Parse options (- prefix)
		if strings.HasPrefix(trimmed, "-") && currentQuestion != nil {
			option := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			currentQuestion.Options = append(currentQuestion.Options, option)
			continue
		}

		// Parse answer (* Answer: prefix)
		if strings.HasPrefix(trimmed, "*") && currentQuestion != nil {
			answerLine := strings.TrimPrefix(trimmed, "*")
			answerLine = strings.TrimSpace(answerLine)
			if strings.HasPrefix(answerLine, "Answer:") {
				answer := strings.TrimSpace(strings.TrimPrefix(answerLine, "Answer:"))
				currentQuestion.Answer = answer
			}
			continue
		}
	}

	// Add last question
	if currentQuestion != nil && currentQuestion.Text != "" {
		quiz.Questions = append(quiz.Questions, *currentQuestion)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading markdown: %w", err)
	}

	// Validate quiz
	if quiz.Title == "" {
		return nil, fmt.Errorf("quiz must have a title")
	}
	if len(quiz.Questions) == 0 {
		return nil, fmt.Errorf("quiz must have at least one question")
	}

	for i, q := range quiz.Questions {
		if q.Text == "" {
			return nil, fmt.Errorf("question %d has no text", i+1)
		}
		if len(q.Options) == 0 {
			return nil, fmt.Errorf("question %d has no options", i+1)
		}
		if q.Answer == "" {
			return nil, fmt.Errorf("question %d has no answer", i+1)
		}
		// Validate answer is in options
		found := false
		for _, opt := range q.Options {
			if opt == q.Answer {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("question %d: answer '%s' not found in options", i+1, q.Answer)
		}
	}

	return quiz, nil
}

// parseDuration parses time strings like "10 seconds", "1 minute", "30s", etc.
func parseDuration(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	// Try to match patterns like "10 seconds", "1 minute", etc.
	re := regexp.MustCompile(`(\d+)\s*(second|sec|s|minute|min|m)s?`)
	matches := re.FindStringSubmatch(s)

	if len(matches) >= 3 {
		value, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}

		unit := matches[2]
		switch unit {
		case "minute", "min", "m":
			return value * 60, nil
		case "second", "sec", "s":
			return value, nil
		}
	}

	// Try just a number (assume seconds)
	if value, err := strconv.Atoi(s); err == nil {
		return value, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s", s)
}

// parseBool parses boolean strings like "true", "false", "yes", "no", "1", "0"
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "yes" || s == "1" || s == "on"
}
