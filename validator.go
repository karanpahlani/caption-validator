package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Error types for validation failures
type CaptionCoverageError struct {
	Type             string  `json:"type"`
	RequiredCoverage float64 `json:"required_coverage"`
	ActualCoverage   float64 `json:"actual_coverage"`
	StartTime        float64 `json:"start_time"`
	EndTime          float64 `json:"end_time"`
	Description      string  `json:"description"`
}

type IncorrectLanguageError struct {
	Type         string `json:"type"`
	DetectedLang string `json:"detected_language"`
	ExpectedLang string `json:"expected_language"`
	Description  string `json:"description"`
}

// Core types
type CaptionValidator struct {
	endpoint string
}

type Caption struct {
	StartTime float64
	EndTime   float64
	Text      string
}

type LanguageResponse struct {
	Lang string `json:"lang"`
}

func NewCaptionValidator(endpoint string) *CaptionValidator {
	return &CaptionValidator{
		endpoint: endpoint,
	}
}

func (cv *CaptionValidator) ValidateFile(filepath string, tStart, tEnd, requiredCoverage float64) error {
	format, err := cv.detectFormat(filepath)
	if err != nil {
		return err
	}

	// Exit with code 1 for unsupported formats
	if format != "webvtt" && format != "srt" {
		os.Exit(1)
	}

	captions, err := cv.parseFile(filepath, format)
	if err != nil {
		return err
	}

	// Run validations and output errors as JSON
	coverageErr := cv.validateCoverage(captions, tStart, tEnd, requiredCoverage)
	if coverageErr != nil {
		if errorJSON, _ := json.Marshal(coverageErr); errorJSON != nil {
			fmt.Println(string(errorJSON))
		}
	}
	
	languageErr := cv.validateLanguage(captions)
	if languageErr != nil {
		if errorJSON, _ := json.Marshal(languageErr); errorJSON != nil {
			fmt.Println(string(errorJSON))
		}
	}

	return nil
}

// detectFormat determines if file is WebVTT or SRT by examining header
func (cv *CaptionValidator) detectFormat(filepath string) (string, error) {
	header := make([]byte, 100)
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}

	headerStr := string(header[:n])
	if strings.Contains(headerStr, "WEBVTT") {
		return "webvtt", nil
	}
	if regexp.MustCompile(`^\d+\s*$`).MatchString(strings.TrimSpace(strings.Split(headerStr, "\n")[0])) {
		return "srt", nil
	}
	return "unknown", fmt.Errorf("unsupported caption format")
}

func (cv *CaptionValidator) parseFile(filepath, format string) ([]Caption, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	switch format {
	case "webvtt":
		return cv.parseWebVTT(string(content))
	case "srt":
		return cv.parseSRT(string(content))
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// parseWebVTT extracts captions from WebVTT format
func (cv *CaptionValidator) parseWebVTT(content string) ([]Caption, error) {
	var captions []Caption
	lines := strings.Split(content, "\n")
	
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.Contains(line, "-->") {
			continue
		}
		
		times := strings.Split(line, "-->")
		if len(times) != 2 {
			continue
		}
		
		startTime, err1 := cv.parseWebVTTTime(strings.TrimSpace(times[0]))
		endTime, err2 := cv.parseWebVTTTime(strings.TrimSpace(times[1]))
		if err1 != nil || err2 != nil {
			continue
		}
		
		// Collect caption text
		var textParts []string
		i++
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			textParts = append(textParts, strings.TrimSpace(lines[i]))
			i++
		}
		
		captions = append(captions, Caption{
			StartTime: startTime,
			EndTime:   endTime,
			Text:      strings.Join(textParts, " "),
		})
	}
	return captions, nil
}

// parseSRT extracts captions from SRT format  
func (cv *CaptionValidator) parseSRT(content string) ([]Caption, error) {
	var captions []Caption
	for _, block := range strings.Split(content, "\n\n") {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 3 || !strings.Contains(lines[1], "-->") {
			continue
		}
		
		times := strings.Split(lines[1], "-->")
		if len(times) != 2 {
			continue
		}
		
		startTime, err1 := cv.parseSRTTime(strings.TrimSpace(times[0]))
		endTime, err2 := cv.parseSRTTime(strings.TrimSpace(times[1]))
		if err1 != nil || err2 != nil {
			continue
		}
		
		captions = append(captions, Caption{
			StartTime: startTime,
			EndTime:   endTime,
			Text:      strings.Join(lines[2:], " "),
		})
	}
	return captions, nil
}

// Time parsing functions for WebVTT (uses .) and SRT (uses ,) formats
func (cv *CaptionValidator) parseWebVTTTime(timeStr string) (float64, error) {
	return cv.parseTime(timeStr, `(\d{2}):(\d{2}):(\d{2})\.(\d{3})`, "WebVTT")
}

func (cv *CaptionValidator) parseSRTTime(timeStr string) (float64, error) {
	return cv.parseTime(timeStr, `(\d{2}):(\d{2}):(\d{2}),(\d{3})`, "SRT")
}

// parseTime converts time string to seconds using provided regex pattern
func (cv *CaptionValidator) parseTime(timeStr, pattern, format string) (float64, error) {
	matches := regexp.MustCompile(pattern).FindStringSubmatch(timeStr)
	if len(matches) != 5 {
		return 0, fmt.Errorf("invalid %s time format: %s", format, timeStr)
	}
	
	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])
	milliseconds, _ := strconv.Atoi(matches[4])
	
	return float64(hours*3600+minutes*60+seconds) + float64(milliseconds)/1000.0, nil
}

// validateCoverage checks if captions cover required percentage of time window
func (cv *CaptionValidator) validateCoverage(captions []Caption, tStart, tEnd, requiredCoverage float64) *CaptionCoverageError {
	totalDuration := tEnd - tStart
	coveredDuration := 0.0
	
	// Calculate overlapping duration for each caption
	for _, caption := range captions {
		if caption.EndTime <= tStart || caption.StartTime >= tEnd {
			continue // No overlap
		}
		
		// Calculate effective overlap within time window
		start := caption.StartTime
		if start < tStart {
			start = tStart
		}
		end := caption.EndTime
		if end > tEnd {
			end = tEnd
		}
		
		if end > start {
			coveredDuration += end - start
		}
	}
	
	actualCoverage := (coveredDuration / totalDuration) * 100
	if actualCoverage < requiredCoverage {
		return &CaptionCoverageError{
			Type:             "caption_coverage",
			RequiredCoverage: requiredCoverage,
			ActualCoverage:   actualCoverage,
			StartTime:        tStart,
			EndTime:          tEnd,
			Description:      fmt.Sprintf("Caption coverage of %.2f%% is below required %.2f%%", actualCoverage, requiredCoverage),
		}
	}
	return nil
}

// validateLanguage sends caption text to endpoint and validates en-US response
func (cv *CaptionValidator) validateLanguage(captions []Caption) *IncorrectLanguageError {
	// Combine all caption text
	var textParts []string
	for _, caption := range captions {
		if caption.Text != "" {
			textParts = append(textParts, caption.Text)
		}
	}
	
	text := strings.Join(textParts, " ")
	if text == "" {
		return nil
	}
	
	detectedLang, err := cv.detectLanguage(text)
	if err != nil {
		return &IncorrectLanguageError{
			Type:         "incorrect_language",
			DetectedLang: "unknown",
			ExpectedLang: "en-US",
			Description:  fmt.Sprintf("Failed to detect language: %v", err),
		}
	}
	
	if detectedLang != "en-US" {
		return &IncorrectLanguageError{
			Type:         "incorrect_language",
			DetectedLang: detectedLang,
			ExpectedLang: "en-US",
			Description:  fmt.Sprintf("Detected language '%s' does not match expected 'en-US'", detectedLang),
		}
	}
	return nil
}

// detectLanguage sends text to HTTP endpoint and returns detected language
func (cv *CaptionValidator) detectLanguage(text string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(cv.endpoint, "text/plain", strings.NewReader(text))
	if err != nil {
		return "", fmt.Errorf("failed to call language detection endpoint: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("language detection endpoint returned status: %d", resp.StatusCode)
	}
	
	var langResp LanguageResponse
	if err := json.NewDecoder(resp.Body).Decode(&langResp); err != nil {
		return "", fmt.Errorf("failed to decode language response: %w", err)
	}
	return langResp.Lang, nil
}