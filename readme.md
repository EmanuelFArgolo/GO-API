
# Quiz Core API

A RESTful API service built with Go for generating AI-powered quizzes and tracking user performance. The application leverages Large Language Models (LLMs) through Ollama to dynamically create quiz questions based on specific themes and topics.

## 🚀 Features

- **AI-Powered Quiz Generation**: Automatically generates 10-question quizzes using LLM (Ollama)
- **Theme-Based Learning**: Creates quizzes focused on specific themes and subject areas
- **Adaptive Learning**: Targets user's weak subjects for improvement
- **Answer Submission & Grading**: Automatically scores quiz submissions
- **User Statistics**: Comprehensive performance tracking and analytics
- **PostgreSQL Database**: Robust data persistence with transactional integrity
- **Docker Support**: Containerized deployment with Docker Compose

## 📋 Table of Contents

- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [API Endpoints](#api-endpoints)
- [Database Schema](#database-schema)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [Usage Examples](#usage-examples)

## 🏗️ Architecture

The application follows a clean architecture pattern with clear separation of concerns:

```
┌─────────────┐
│   HTTP API  │  (handlers.go)
└──────┬──────┘
       │
┌──────▼──────┐
│   Service   │  (service.go)
└──┬───────┬──┘
   │       │
┌──▼──┐ ┌─▼────┐
│Store│ │ LLM  │
│(DB) │ │Client│
└─────┘ └──────┘
```

### Layers:

1. **API Layer** (`internal/api`): HTTP handlers, request/response formatting
2. **Service Layer** (`internal/service`): Business logic, orchestration
3. **Store Layer** (`internal/store`): Database operations, repositories
4. **LLM Client** (`internal/llm`): External LLM service integration
5. **Models** (`internal/models`): Data structures and DTOs
6. **Config** (`internal/config`): Configuration management

## 🛠️ Tech Stack

- **Language**: Go 1.23+
- **Database**: PostgreSQL 16
- **ORM**: sqlx
- **Migrations**: golang-migrate
- **LLM**: Ollama (configurable model)
- **Containerization**: Docker & Docker Compose
- **Environment**: godotenv

## 📁 Project Structure

```
quizz-core/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   └── handlers.go             # HTTP request handlers
│   ├── config/
│   │   └── config.go               # Configuration loader
│   ├── llm/
│   │   └── client.go               # LLM client implementation
│   ├── models/
│   │   └── models.go               # Data models and DTOs
│   ├── service/
│   │   └── service.go              # Business logic
│   └── store/
│       ├── store.go                # Database connection
│       ├── migrate.go              # Migration runner
│       ├── quiz_repo.go            # Quiz CRUD operations
│       ├── submission_repo.go      # Submission handling
│       ├── stats_repo.go           # Statistics queries
│       └── migrations/
│           ├── 000001_initial_schema.up.sql
│           └── 000001_initial_schema.down.sql
├── docker-compose.yml              # Docker orchestration
├── Dockerfile                      # Multi-stage build
├── go.mod                          # Go dependencies
└── readme.md                       # This file
```

## 🔌 API Endpoints

### Health Check
```http
GET /health
```
Returns API health status.

**Response:**
```json
{
  "status": "ok"
}
```

### Create Quiz
```http
POST /api/v1/quiz/create
```
Generates a new quiz based on theme and weak subjects using LLM.

**Request Body:**
```json
{
  "user_id": "123",
  "theme": "Physics",
  "wrong_subjects": [
    "Newton's Laws",
    "Thermodynamics",
    "Quantum Mechanics",
    "Relativity",
    "Electromagnetism"
  ]
}
```

**Response:**
```json
{
  "quiz_id": "42",
  "subject": "Physics",
  "questions": [
    {
      "id": "1",
      "subject": "Newton's Laws",
      "question": "What is Newton's first law of motion?",
      "options": [
        "An object at rest stays at rest",
        "Force equals mass times acceleration",
        "For every action there is a reaction",
        "Energy is conserved"
      ]
    }
  ]
}
```

### Submit Answers
```http
POST /api/v1/quiz/submit
```
Submits quiz answers for grading and stores results.

**Request Body:**
```json
{
  "quiz_id": "42",
  "user_id": "123",
  "answers": [
    {
      "question_id": "1",
      "selected_option": "An object at rest stays at rest"
    },
    {
      "question_id": "2",
      "selected_option": "E=mc²"
    }
  ]
}
```

**Response:**
```json
{
  "submission_id": 15,
  "score": 85.0,
  "correct_count": 8,
  "total_count": 10,
  "message": "Submissão bem-sucedida! Acertou 8 de 10."
}
```

### Get User Statistics
```http
GET /api/v1/users/stats?user_id=123
```
Retrieves comprehensive user performance statistics.

**Response:**
```json
{
  "user_id": "123",
  "total_quizzes_realizados": 5,
  "total_perguntas_respondidas": 50,
  "total_acertos": 42,
  "total_erros": 8,
  "percentagem_acerto": 84.0,
  "pontuacao_media": 84.0
}
```

## 🗄️ Database Schema

### Core Tables

- **utilizadores**: User accounts (students, teachers, admins)
- **tema**: Quiz themes/subjects
- **quizzes**: Quiz metadata
- **perguntas**: Questions with subjects
- **respostas**: Answer options (marked as correct/incorrect)
- **submissao**: Quiz submission records with scores
- **respostas_dadas**: User's selected answers
- **dificuldades**: Tracked difficult subjects per submission

### Entity Relationships

```
utilizadores ──┐
               ├──< submissao >──┐
quizzes ───────┘                 │
   │                             │
   └──< perguntas >──< respostas │
                         │       │
                         └───< respostas_dadas
                                 │
                                 └──< dificuldades
tema ──< quizzes
```

## 🚀 Getting Started

### Prerequisites

- Docker & Docker Compose
- Git

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/quizz-core.git
   cd quizz-core
   ```

2. **Create environment file**
   ```bash
   cp .env.example .env
   ```

3. **Configure environment variables** (see [Environment Variables](#environment-variables))

4. **Start the services**
   ```bash
   docker-compose up --build
   ```

5. **Verify the API is running**
   ```bash
   curl http://localhost:8080/health
   ```

### Local Development (without Docker)

1. **Install Go 1.23+**

2. **Install PostgreSQL 16**

3. **Set environment variables**
   ```bash
   export DB_HOST=localhost
   export DB_PORT=5432
   export DB_USER=your_user
   export DB_PASSWORD=your_password
   export DB_NAME=quiz_db
   export LLM_ENDPOINT=http://localhost:11434/api/generate
   export LLM_MODEL=llama2
   ```

4. **Install dependencies**
   ```bash
   go mod download
   ```

5. **Run database migrations** (automatic on startup)

6. **Start the server**
   ```bash
   go run cmd/server/main.go
   ```

## 🔐 Environment Variables

Create a `.env` file in the project root:

```env
# Database Configuration
DB_HOST=db_quiz
DB_PORT=5432
DB_USER=quizadmin
DB_PASSWORD=securepassword123
DB_NAME=quiz_database

# LLM Configuration
LLM_ENDPOINT=http://host.docker.internal:11434/api/generate
LLM_MODEL=llama2

# API Configuration (optional)
PORT=8080
```

### Environment Variables Description

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database username | - |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | - |
| `LLM_ENDPOINT` | Ollama API endpoint | - |
| `LLM_MODEL` | LLM model to use | `llama2` |
| `PORT` | API server port | `8080` |

## 📝 Usage Examples

### Creating a Quiz Flow

1. **User completes an assessment** (external system)
2. **External system identifies weak subjects**
3. **Call Quiz Creation API** with user ID, theme, and weak subjects
4. **LLM generates 10 targeted questions**
5. **Questions are stored in database**
6. **API returns quiz with questions to present to user**

### Submitting Answers Flow

1. **User completes the quiz** (external frontend)
2. **Frontend calls Submit API** with quiz ID, user ID, and answers
3. **API retrieves correct answers from database**
4. **Compares user answers with correct answers**
5. **Calculates score and identifies difficult subjects**
6. **Stores submission, answers, and difficulties**
7. **Returns score and results to user**

### Viewing Statistics Flow

1. **User requests their statistics**
2. **API aggregates data** from all submissions
3. **Calculates metrics**: total quizzes, accuracy, average score
4. **Returns comprehensive statistics**

## 🐳 Docker Configuration

### Services

- **api**: Go application (port 8080)
- **db_quiz**: PostgreSQL 16 (port 5432)

### Volumes

- `postgres_data`: Persists database data

### Networks

- `quiz-net`: Bridge network for service communication

### Health Checks

PostgreSQL includes a health check to ensure database readiness before API startup.

## 🔄 Database Migrations

Migrations run automatically on application startup using `golang-migrate`.

**Migration files location**: `internal/store/migrations/`

### Manual Migration Commands

```bash
# Up migration
migrate -path internal/store/migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up

# Down migration
migrate -path internal/store/migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" down
```

## 🧪 Testing the API

### Using cURL

```bash
# Health check
curl http://localhost:8080/health

# Create quiz
curl -X POST http://localhost:8080/api/v1/quiz/create \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "1",
    "theme": "Mathematics",
    "wrong_subjects": ["Algebra", "Geometry", "Calculus", "Statistics", "Trigonometry"]
  }'

# Submit answers
curl -X POST http://localhost:8080/api/v1/quiz/submit \
  -H "Content-Type: application/json" \
  -d '{
    "quiz_id": "1",
    "user_id": "1",
    "answers": [
      {"question_id": "1", "selected_option": "Option text"},
      {"question_id": "2", "selected_option": "Option text"}
    ]
  }'

# Get statistics
curl "http://localhost:8080/api/v1/users/stats?user_id=1"
```

## 🤝 Integration with LLM (Ollama)

The application communicates with Ollama to generate quiz questions.

### LLM Prompt Structure

The system sends a structured prompt to generate:
- 10 questions per quiz
- 4 options per question
- Subject targeting based on weak areas
- JSON-formatted response

### Response Parsing

- Cleans LLM response to extract valid JSON
- Validates structure
- Maps to internal question model

## 📊 Performance Tracking

The system tracks:
- **Per-submission metrics**: Score, correct/incorrect answers
- **Difficult subjects**: Topics where user struggled
- **Aggregate statistics**: Overall performance across all quizzes
- **Historical data**: All submissions preserved for analytics

## 🔒 Error Handling

The API implements comprehensive error handling:

- **400 Bad Request**: Invalid input data
- **404 Not Found**: Resource doesn't exist (quiz, user)
- **500 Internal Server Error**: Database or LLM failures

Custom error types:
- `ErrNotFound`: Resource not found
- `ErrInvalidInput`: Invalid input parameters

## 🌟 Key Features Explained

### Transactional Integrity

All database operations use transactions to ensure data consistency:
- Quiz creation (theme → quiz → questions → answers)
- Submission recording (submission → answers → difficulties)

### Dependency Injection

Clean architecture with dependency injection:
```go
Store → Service → Handlers
LLM Client → Service → Handlers
```

### Database Tags

Uses `sqlx` for struct-to-SQL mapping with db tags:
```go
type User struct {
    ID   int    `json:"id" db:"id"`
    Nome string `json:"nome" db:"nome"`
}
```

## 📄 License

This project is licensed under the terms specified in the [LICENSE](LICENSE) file.

## 👥 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📧 Support

For questions or support, please open an issue in the repository.

---

**Built with ❤️ for adaptive learning and educational technology**

