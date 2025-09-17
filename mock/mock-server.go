package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type LanguageResponse struct {
	Lang string `json:"lang"`
}

func detectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the text from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Printf("Received text: %s\n", string(body))

	// Always return en-US for testing
	response := LanguageResponse{Lang: "en-US"}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/detect", detectHandler)
	
	fmt.Println("Mock language detection server starting on :8081")
	fmt.Println("POST /detect - accepts plaintext, returns {\"lang\": \"en-US\"}")
	
	log.Fatal(http.ListenAndServe(":8081", nil))
}