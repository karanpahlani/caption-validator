// Caption validator validates WebVTT and SRT caption files for coverage and language
package main

import (
	"flag"
	"log"
)

func main() {
	var tStart = flag.Float64("t_start", 0, "Start time in seconds")
	var tEnd = flag.Float64("t_end", 0, "End time in seconds")
	var coverage = flag.Float64("coverage", 80, "Required coverage percentage")
	var endpoint = flag.String("endpoint", "", "Language detection endpoint URL")
	flag.Parse()

	// Validate arguments
	if flag.NArg() < 1 {
		log.Fatal("Usage: caption-validator [flags] captions-filepath")
	}
	if *endpoint == "" {
		log.Fatal("Language detection endpoint is required (use -endpoint flag)")
	}
	if *tEnd <= *tStart {
		log.Fatal("End time must be greater than start time")
	}

	// Validate caption file
	validator := NewCaptionValidator(*endpoint)
	if err := validator.ValidateFile(flag.Arg(0), *tStart, *tEnd, *coverage); err != nil {
		log.Fatal(err)
	}
}