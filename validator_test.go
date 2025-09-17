package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	cv := NewCaptionValidator("http://test.com")

	tests := []struct {
		name        string
		content     string
		expected    string
		expectError bool
	}{
		{
			name:        "WebVTT format",
			content:     "WEBVTT\n\n00:00:01.000 --> 00:00:05.000\nHello world",
			expected:    "webvtt",
			expectError: false,
		},
		{
			name:        "SRT format",
			content:     "1\n00:00:01,000 --> 00:00:05,000\nHello world",
			expected:    "srt",
			expectError: false,
		},
		{
			name:        "unsupported format",
			content:     "This is just plain text without any caption format",
			expected:    "unknown",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test_caption_*.txt")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			if err != nil {
				t.Fatal(err)
			}
			tmpFile.Close()

			format, err := cv.detectFormat(tmpFile.Name())
			if tt.expectError {
				if err == nil {
					t.Error("expected error for unsupported format, got none")
				}
				if format != tt.expected {
					t.Errorf("expected format %s, got %s", tt.expected, format)
				}
			} else {
				if err != nil {
					t.Fatalf("detectFormat failed: %v", err)
				}
				if format != tt.expected {
					t.Errorf("expected format %s, got %s", tt.expected, format)
				}
			}
		})
	}
}

func TestParseWebVTT(t *testing.T) {
	cv := NewCaptionValidator("http://test.com")

	content := `WEBVTT

00:00:01.000 --> 00:00:05.000
Hello world

00:00:06.000 --> 00:00:10.000
This is a test`

	captions, err := cv.parseWebVTT(content)
	if err != nil {
		t.Fatal(err)
	}

	if len(captions) != 2 {
		t.Errorf("expected 2 captions, got %d", len(captions))
	}

	if captions[0].StartTime != 1.0 {
		t.Errorf("expected start time 1.0, got %f", captions[0].StartTime)
	}

	if captions[0].EndTime != 5.0 {
		t.Errorf("expected end time 5.0, got %f", captions[0].EndTime)
	}

	if captions[0].Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got '%s'", captions[0].Text)
	}
}

func TestParseSRT(t *testing.T) {
	cv := NewCaptionValidator("http://test.com")

	content := `1
00:00:01,000 --> 00:00:05,000
Hello world

2
00:00:06,000 --> 00:00:10,000
This is a test`

	captions, err := cv.parseSRT(content)
	if err != nil {
		t.Fatal(err)
	}

	if len(captions) != 2 {
		t.Errorf("expected 2 captions, got %d", len(captions))
	}

	if captions[0].StartTime != 1.0 {
		t.Errorf("expected start time 1.0, got %f", captions[0].StartTime)
	}

	if captions[0].EndTime != 5.0 {
		t.Errorf("expected end time 5.0, got %f", captions[0].EndTime)
	}

	if captions[0].Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got '%s'", captions[0].Text)
	}
}

func TestValidateCoverage(t *testing.T) {
	cv := NewCaptionValidator("http://test.com")

	captions := []Caption{
		{StartTime: 1.0, EndTime: 3.0, Text: "Hello"},
		{StartTime: 5.0, EndTime: 7.0, Text: "World"},
	}

	tests := []struct {
		name             string
		tStart           float64
		tEnd             float64
		requiredCoverage float64
		expectError      bool
	}{
		{
			name:             "sufficient coverage",
			tStart:           0.0,
			tEnd:             10.0,
			requiredCoverage: 40.0,
			expectError:      false,
		},
		{
			name:             "insufficient coverage",
			tStart:           0.0,
			tEnd:             10.0,
			requiredCoverage: 50.0,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cv.validateCoverage(captions, tt.tStart, tt.tEnd, tt.requiredCoverage)
			if tt.expectError && err == nil {
				t.Error("expected coverage error, got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected coverage error: %v", err)
			}
		})
	}
}

func TestValidateLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"lang": "en-US"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cv := NewCaptionValidator(server.URL)

	captions := []Caption{
		{StartTime: 1.0, EndTime: 3.0, Text: "Hello world"},
		{StartTime: 5.0, EndTime: 7.0, Text: "This is English"},
	}

	err := cv.validateLanguage(captions)
	if err != nil {
		t.Errorf("unexpected language validation error: %v", err)
	}
}

func TestValidateLanguageIncorrect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"lang": "es-ES"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cv := NewCaptionValidator(server.URL)

	captions := []Caption{
		{StartTime: 1.0, EndTime: 3.0, Text: "Hola mundo"},
	}

	err := cv.validateLanguage(captions)
	if err == nil {
		t.Error("expected language validation error, got none")
	}

	if err.DetectedLang != "es-ES" {
		t.Errorf("expected detected language 'es-ES', got '%s'", err.DetectedLang)
	}
}

func TestTimeParsingWebVTT(t *testing.T) {
	cv := NewCaptionValidator("http://test.com")

	tests := []struct {
		timeStr  string
		expected float64
	}{
		{"00:00:01.000", 1.0},
		{"00:01:30.500", 90.5},
		{"01:00:00.000", 3600.0},
	}

	for _, tt := range tests {
		result, err := cv.parseWebVTTTime(tt.timeStr)
		if err != nil {
			t.Errorf("failed to parse time %s: %v", tt.timeStr, err)
		}
		if result != tt.expected {
			t.Errorf("for time %s, expected %f, got %f", tt.timeStr, tt.expected, result)
		}
	}
}

func TestTimeParsingSRT(t *testing.T) {
	cv := NewCaptionValidator("http://test.com")

	tests := []struct {
		timeStr  string
		expected float64
	}{
		{"00:00:01,000", 1.0},
		{"00:01:30,500", 90.5},
		{"01:00:00,000", 3600.0},
	}

	for _, tt := range tests {
		result, err := cv.parseSRTTime(tt.timeStr)
		if err != nil {
			t.Errorf("failed to parse time %s: %v", tt.timeStr, err)
		}
		if result != tt.expected {
			t.Errorf("for time %s, expected %f, got %f", tt.timeStr, tt.expected, result)
		}
	}
}

func TestValidateFileUnsupportedFormat(t *testing.T) {
	// Create a temporary file with unsupported content
	tmpFile, err := os.CreateTemp("", "test_unsupported_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("This is just plain text, not a caption file")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Since ValidateFile calls os.Exit(1), we can't test it directly in a unit test
	// Instead, we test the underlying logic that would lead to the exit
	cv := NewCaptionValidator("http://test.com")
	format, err := cv.detectFormat(tmpFile.Name())
	
	// Should return "unknown" format with error
	if err == nil {
		t.Error("expected error for unsupported format, got none")
	}
	if format != "unknown" {
		t.Errorf("expected format 'unknown', got '%s'", format)
	}
	
	// The actual exit(1) behavior is tested in the integration test below
}

func TestUnsupportedFileTypeExitCode(t *testing.T) {
	// Test the actual main program behavior with unsupported file types
	// This test runs the main program as a subprocess to verify exit code 1
	
	// Create a temporary file with unsupported content
	tmpFile, err := os.CreateTemp("", "test_unsupported_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("This is just plain text, not a caption file")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Run the main program with the unsupported file
	cmd := exec.Command("go", "run", ".", 
		"-t_start=0", "-t_end=30", "-coverage=80", 
		"-endpoint=http://localhost:8080/detect", 
		tmpFile.Name())
	
	err = cmd.Run()
	
	// The program should exit with code 1 for unsupported file types
	if e, ok := err.(*exec.ExitError); ok {
		if e.ExitCode() == 1 {
			return // Test passed - got expected exit code 1
		}
		t.Fatalf("expected exit code 1, got %d", e.ExitCode())
	}
	t.Fatalf("process should have exited with code 1, but it didn't exit or returned success")
}

func TestJSONErrorOutputFormat(t *testing.T) {
	// Test coverage error JSON format
	cv := NewCaptionValidator("http://test.com")
	
	captions := []Caption{
		{StartTime: 1.0, EndTime: 2.0, Text: "Short"},
	}
	
	// Test coverage error
	coverageErr := cv.validateCoverage(captions, 0, 10, 80)
	if coverageErr == nil {
		t.Fatal("expected coverage error, got none")
	}
	
	// Verify JSON structure
	jsonData, err := json.Marshal(coverageErr)
	if err != nil {
		t.Fatalf("failed to marshal coverage error to JSON: %v", err)
	}
	
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	
	// Verify required fields
	if parsed["type"] != "caption_coverage" {
		t.Errorf("expected type 'caption_coverage', got '%v'", parsed["type"])
	}
	if _, ok := parsed["description"]; !ok {
		t.Error("missing 'description' field in JSON output")
	}
	if _, ok := parsed["required_coverage"]; !ok {
		t.Error("missing 'required_coverage' field in JSON output")
	}
	if _, ok := parsed["actual_coverage"]; !ok {
		t.Error("missing 'actual_coverage' field in JSON output")
	}
}

func TestLanguageErrorJSONFormat(t *testing.T) {
	// Test language error JSON format
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"lang": "es-ES"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cv := NewCaptionValidator(server.URL)
	captions := []Caption{
		{StartTime: 1.0, EndTime: 3.0, Text: "Hola mundo"},
	}

	langErr := cv.validateLanguage(captions)
	if langErr == nil {
		t.Fatal("expected language error, got none")
	}

	// Verify JSON structure
	jsonData, err := json.Marshal(langErr)
	if err != nil {
		t.Fatalf("failed to marshal language error to JSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify required fields
	if parsed["type"] != "incorrect_language" {
		t.Errorf("expected type 'incorrect_language', got '%v'", parsed["type"])
	}
	if _, ok := parsed["description"]; !ok {
		t.Error("missing 'description' field in JSON output")
	}
	if _, ok := parsed["detected_language"]; !ok {
		t.Error("missing 'detected_language' field in JSON output")
	}
	if _, ok := parsed["expected_language"]; !ok {
		t.Error("missing 'expected_language' field in JSON output")
	}
}

func TestValidationFailuresExitCode0(t *testing.T) {
	// Test that validation failures (coverage and language errors) exit with code 0
	// Create a WebVTT file with low coverage
	tmpFile, err := os.CreateTemp("", "test_low_coverage_*.webvtt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write WebVTT with minimal coverage (1 second out of 30 = 3.33%)
	content := `WEBVTT

00:00:01.000 --> 00:00:02.000
Short caption`

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Set up a mock server that returns non-English language
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"lang": "es-ES"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Run the main program with validation failures
	cmd := exec.Command("go", "run", ".", 
		"-t_start=0", "-t_end=30", "-coverage=80", 
		"-endpoint="+server.URL, 
		tmpFile.Name())
	
	// Capture output to verify JSON errors are printed
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	
	err = cmd.Run()
	
	// Should exit with code 0 despite validation failures
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			t.Fatalf("expected exit code 0 for validation failures, got %d", e.ExitCode())
		}
		t.Fatalf("unexpected error running program: %v", err)
	}
	
	// Verify that JSON errors were printed to stdout
	output := stdout.String()
	if output == "" {
		t.Error("expected JSON error output, got none")
	}
	
	// Should contain both coverage and language errors as JSON
	if !bytes.Contains(stdout.Bytes(), []byte("caption_coverage")) {
		t.Error("expected coverage error JSON in output")
	}
	if !bytes.Contains(stdout.Bytes(), []byte("incorrect_language")) {
		t.Error("expected language error JSON in output")
	}
}

func TestSuccessfulValidationExitCode0(t *testing.T) {
	// Test that successful validation also exits with code 0
	tmpFile, err := os.CreateTemp("", "test_success_*.webvtt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write WebVTT with good coverage (100% of 30 seconds)
	content := `WEBVTT

00:00:00.000 --> 00:00:30.000
This is a complete caption covering the entire time window`

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Set up a mock server that returns English
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"lang": "en-US"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Run the main program with successful validation
	cmd := exec.Command("go", "run", ".", 
		"-t_start=0", "-t_end=30", "-coverage=80", 
		"-endpoint="+server.URL, 
		tmpFile.Name())
	
	err = cmd.Run()
	
	// Should exit with code 0 for successful validation
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			t.Fatalf("expected exit code 0 for successful validation, got %d", e.ExitCode())
		}
		t.Fatalf("unexpected error running program: %v", err)
	}
}