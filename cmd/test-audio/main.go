package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
)

func main() {
	size := flag.Int("size", 10240, "File size in bytes (default 10KB)")
	output := flag.String("output", "test_audio.opus", "Output file path")
	flag.Parse()

	fmt.Printf("Generating test audio file: %s (%d bytes)\n", *output, *size)

	// Generate random data to simulate audio
	data := make([]byte, *size)
	if _, err := rand.Read(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating random data: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Test file created successfully\n")
	fmt.Printf("  File: %s\n", *output)
	fmt.Printf("  Size: %d bytes\n", *size)
}
