# Caption Validator

A Go application that validates caption files (WebVTT and SRT formats) for coverage and language detection.

## Features

- Supports WebVTT and SRT caption file formats
- Validates caption coverage within specified time ranges
- Detects language via configurable web endpoint
- Returns validation errors as JSON objects
- Dockerized for easy deployment

## Quick Start

### 1. Run Unit Tests
```bash
go test -v
```

### 2. Start Mock Language Detection Server
**Terminal 1:**
```bash
cd mock && go run mock-server.go
```
You should see: `Mock language detection server starting on :8081`

**If you see "address already in use" error:**
```bash
# Kill any process using port 8081
lsof -ti:8081 | xargs kill -9
# Then restart the server
cd mock && go run mock-server.go
```

### 3. Test the Application
**Terminal 2 - Test successful validation (no output expected):**
```bash
go run . -t_start=0 -t_end=15 -coverage=60 -endpoint=http://localhost:8081/detect testdata/sample.webvtt
```

**Test with validation failures:**
```bash
go run . -t_start=0 -t_end=30 -coverage=80 -endpoint=http://localhost:8081/detect testdata/sample.webvtt
```

**Test unsupported file format (exit code 1):**
```bash
echo "not a caption file" > test.txt
go run . -t_start=0 -t_end=30 -coverage=80 -endpoint=http://localhost:8081/detect test.txt
```

## Parameters

- `-t_start`: Start time in seconds (required)
- `-t_end`: End time in seconds (required) 
- `-coverage`: Required coverage percentage (default: 80)
- `-endpoint`: Language detection endpoint URL (required)

## Docker Usage

### Build and Test with Docker
```bash
# Build the image
docker build -t caption-validator .

# Test successful validation (no output expected)
docker run -v $(pwd)/testdata:/captions caption-validator \
  -t_start=0 -t_end=30 -coverage=60 \
  -endpoint=http://host.docker.internal:8081/detect \
  /captions/sample.webvtt

# Test with validation failures
docker run -v $(pwd)/testdata:/captions caption-validator \
  -t_start=0 -t_end=30 -coverage=80 \
  -endpoint=http://host.docker.internal:8081/detect \
  /captions/sample.webvtt
```

## Expected Output

### Validation Failures (JSON objects)
**Coverage failure (with mock server returning en-US):**
```json
{"type": "caption_coverage", "required_coverage": 80, "actual_coverage": 70, "start_time": 0, "end_time": 30, "description": "Caption coverage of 70.00% is below required 80.00%"}
```

**Language failure example (requires endpoint returning non-en-US):**
```json
{"type": "incorrect_language", "detected_language": "es-ES", "expected_language": "en-US", "description": "Detected language 'es-ES' does not match expected 'en-US'"}
```

**To test language validation failure:**
1. Modify `mock/mock-server.go` line 32: change `"en-US"` to `"es-ES"`
2. Restart the mock server: `lsof -ti:8081 | xargs kill -9 && cd mock && go run mock-server.go`
3. Run the Docker command again to see both coverage and language errors

### Success
No output indicates successful validation.

## Language Detection API

Your language detection endpoint should accept POST requests with plaintext body and return JSON:

```json
{
  "lang": "en-US"
}
```

Expected language is `en-US`. Any other value triggers a validation error.

## Exit Codes

- `0`: Success (validation passed or failed with JSON output)
- `1`: Unsupported file format or program error