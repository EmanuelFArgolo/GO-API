package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"quizz-core/internal/models"
	"quizz-core/internal/service" // Import the service
)

// ApiHandlers is our struct that will hold dependencies,
// like the quiz service.
type ApiHandlers struct {
	quizService *service.QuizService
}

// NewApiHandlers is the constructor for our handlers
func NewApiHandlers(qs *service.QuizService) *ApiHandlers {
	// It receives the service (dependency injection)
	return &ApiHandlers{
		quizService: qs,
	}
}

// HealthCheckHandler (now a method of ApiHandlers)
func (h *ApiHandlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "ok"}`)
}

// CreateQuizHandler is the handler for your main endpoint
// It knows about HTTP, the service does not.
func (h *ApiHandlers) CreateQuizHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Only accept POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Decode the JSON from the request body into our struct
	var req models.CreateQuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 3. Call the Service (the "brain")
	// We pass only the clean data (req), not the whole http.Request
	quizResponse, err := h.quizService.CreateQuiz(r.Context(), req)
	if err != nil {
		// Log the detailed error for us (server-side)
		log.Printf("Error creating quiz: %v", err)
		// Send a generic error to the client
		http.Error(w, "Failed to create quiz", http.StatusInternalServerError)
		return
	}

	// 4. Send the successful JSON response back to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created is a good status here
	if err := json.NewEncoder(w).Encode(quizResponse); err != nil {
		log.Printf("Error sending JSON response: %v", err)
	}
}
