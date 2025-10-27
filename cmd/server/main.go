package main

import (
	"log"
	"net/http"

	// All our application packages
	"quizz-core/internal/api"
	"quizz-core/internal/config"
	"quizz-core/internal/llm"
	"quizz-core/internal/service"
	"quizz-core/internal/store"
)

func main() {
	// 1. Load Configuration
	cfg := config.LoadConfig()

	// 2. Run Database Migrations
	store.RunMigrations(cfg.DBConnectionStringURL)

	// --- DEPENDENCY INJECTION (Building all the pieces) ---

	// 3. Build the Database Layer (Store)
	db, err := store.NewPostgresStore(cfg.DBConnectionString)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	// 4. Build the LLM Client
	llmClient := llm.NewClient(cfg.LLMEndpoint, cfg.LLMModel)
	// 5. Build the Service Layer (Injecting DB and LLM)
	quizSvc := service.NewQuizService(db, llmClient)

	// 6. Build the API/Handlers Layer (Injecting the Service)
	handlers := api.NewApiHandlers(quizSvc)

	// --- End of Dependency Injection ---

	// 7. Configure Routes (connecting URLs to the handler methods)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlers.HealthCheckHandler)
	mux.HandleFunc("/api/v1/quiz/create", handlers.CreateQuizHandler) // <-- OUR NEW ENDPOINT
	// (Future) mux.HandleFunc("/api/v1/quiz/submit", handlers.SubmitAnswersHandler)

	// 8. Start the Server
	serverAddr := ":" + cfg.Port
	log.Printf("Server running on http://localhost:%s\n", cfg.Port)

	err = http.ListenAndServe(serverAddr, mux)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
