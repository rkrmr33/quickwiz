# 🎯 QuicKwiz

A real-time quiz platform built with Go and HTMX. Create quizzes using simple markdown, share a code, and engage participants with timed questions and live results.

## ✨ Features

- **📝 Markdown-based Quiz Creation**: Define quizzes using a simple, readable markdown format
- **🔗 Easy Sharing**: Generate a unique code to share with participants
- **⚡ Real-time Synchronization**: WebSocket-powered live updates for all participants
- **⏱️ Timed Questions**: Configurable time limits per question
- **🏆 Live Leaderboard**: See results and scores update in real-time
- **🎨 Modern UI**: Clean, responsive interface using HTMX
- **🐳 Docker Ready**: Easy deployment with Docker and Docker Compose

## 🚀 Quick Start

### Prerequisites

- Go 1.24 or higher
- Docker and Docker Compose (optional, for containerized deployment)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/rkrmr33/quickwiz.git
   cd quickwiz
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Run the server**
   ```bash
   go run cmd/server/main.go
   ```

4. **Open your browser**
   ```
   http://localhost:8080
   ```

### Using Docker

1. **Build and run with Docker Compose**
   ```bash
   docker-compose up --build
   ```

2. **Access the application**
   ```
   http://localhost:8080
   ```

## 📖 Quiz Markdown Format

Create quizzes using this simple format:

```markdown
# My Awesome Quiz

# Settings
time_per_question: 30 seconds

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
```

### Format Rules

1. **Title**: First `#` heading becomes the quiz title
2. **Settings**: Optional `# Settings` section
   - `time_per_question`: Duration in seconds, minutes, or as "X seconds/minutes"
   - `time_between_questions`: Time between questions
   - `streak_bonus`: true/false to enable/disable streak scoring
3. **Questions**: Use `###` for question text
4. **Options**: Use `-` for each answer option
5. **Answer**: Use `* Answer:` followed by the correct answer (must match one of the options exactly)

## 🎮 How to Use

### Creating a Quiz

1. Navigate to the home page
2. Paste your quiz markdown in the text area
3. Click "Create Quiz 🚀"
4. Share the generated code with participants

### Joining a Quiz

1. Receive the quiz code from the creator
2. Navigate to `/quiz/{code}`
3. Enter your name
4. Click "Join Quiz 🚀"
5. Wait for the quiz to start

### Playing the Quiz

1. Once the quiz starts, questions appear one at a time
2. Select your answer before time runs out
3. See results after each question
4. View the final leaderboard at the end

### Key Technologies

- **Backend**: Go with Gorilla Mux and WebSocket
- **Frontend**: HTML, HTMX for dynamic updates
- **Real-time**: WebSockets for live synchronization
- **Testing**: Go testing package

## 📝 License

MIT License - feel free to use this project for any purpose.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 🐛 Known Issues

- Session cleanup happens hourly; very old sessions will persist until cleanup

## 🚀 Future Enhancements

- [ ] Persistent storage (database)
- [ ] Question categories and difficulty levels
- [ ] Image support in questions
- [ ] Mobile app

## 📧 Contact

For questions or feedback, please open an issue on GitHub.

---

Made with ❤️ using Go and HTMX
