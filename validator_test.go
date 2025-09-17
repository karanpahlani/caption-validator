package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	cv := NewCaptionValidator("http://test.com")

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "WebVTT format",
			content:  "WEBVTT\n\n00:00:01.000 --> 00:00:05.000\nHello world",
			expected: "webvtt",
		},
		{
			name:     "SRT format",
			content:  "1\n00:00:01,000 --> 00:00:05,000\nHello world",
			expected: "srt",
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
			if err != nil {
				t.Fatalf("detectFormat failed: %v", err)
			}

			if format != tt.expected {
				t.Errorf("expected format %s, got %s", tt.expected, format)
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