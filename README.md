# Caption Validator

A Go application that validates caption files (WebVTT and SRT formats) for coverage and language detection.

## Features

- Supports WebVTT and SRT caption file formats
- Validates caption coverage within specified time ranges
- Detects language via configurable web endpoint
- Returns validation errors as JSON objects
- Dockerized for easy deployment

## Usage

### Command Line

```bash
go run . -t_start=0 -t_end=30 -coverage=80 -endpoint=http://localhost:8080/detect testdata/sample.webvtt
```

### Parameters

- `-t_start`: Start time in seconds (required)
- `-t_end`: End time in seconds (required) 
- `-coverage`: Required coverage percentage (default: 80)
- `-endpoint`: Language detection endpoint URL (required)

### Docker

Build the image:
```bash
docker build -t caption-validator .
```

Run with Docker:
```bash
docker run -v $(pwd)/testdata:/captions caption-validator \
  -t_start=0 -t_end=30 -coverage=80 \
  -endpoint=http://host.docker.internal:8080/detect \
  /captions/sample.webvtt
```

## Output

The program outputs JSON objects for validation failures:

```json
{"type": "caption_coverage", "required_coverage": 80, "actual_coverage": 45.5, "start_time": 0, "end_time": 30, "description": "Caption coverage of 45.50% is below required 80.00%"}
{"type": "incorrect_language", "detected_language": "es-ES", "expected_language": "en-US", "description": "Detected language 'es-ES' does not match expected 'en-US'"}
```

No output indicates successful validation.

## Testing

Run tests:
```bash
go test -v
```

## Language Detection API

The language detection endpoint should accept POST requests with plaintext body and return JSON:

```json
{
  "lang": "en-US"
}
```

Expected language is `en-US`. Any other value will trigger a validation error.

### Mock Server for Testing

A mock language detection server is provided in the `mock/` directory:

```bash
# Start mock server
cd mock && go run mock-server.go

# Test with mock server (in another terminal)
go run . -t_start=0 -t_end=15 -coverage=60 -endpoint=http://localhost:8081/detect testdata/sample.webvtt
```

## Exit Codes

- `0`: Success (validation passed or failed with JSON output)
- `1`: Unsupported file format or program error