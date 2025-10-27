# API - Macroserviço de Quizzes

Este serviço gere a criação e submissão de quizzes.

## Como correr
1. Tenha o Docker e o Docker Compose instalados.
2. Crie um `.env` (copie do `.env.example` - crie este ficheiro também).
3. Rode `docker-compose up --build`

## Endpoints

### 1. Criar um Novo Quiz
- **URL:** `POST /api/v1/quiz/create`
- **Request Body (JSON):**
  ```json
  {
    "user_id": "1",
    "theme": "Biologia Celular",
    "wrong_subjects": ["Mitocôndrias", "Complexo de Golgi"]
  }
  {
  "quiz_id": "1",
  "subject": "Biologia Celular",
  "questions": [
    {
      "id": "1",
      "subject": "Mitocôndrias",
      "question": "Qual é a função...",
      "options": ["A", "B", "C", "D"]
    }
  ]
}
    ```
