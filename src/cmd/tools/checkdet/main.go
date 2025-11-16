package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func main() {
	config := flag.String("config", "config/scenario1.json", "config file to reuse for determinism check")
	flag.Parse()

	tmp1 := "logs/determinism_run1.log"
	tmp2 := "logs/determinism_run2.log"

	if err := runSimulator(*config, tmp1); err != nil {
		fmt.Fprintf(os.Stderr, "first run failed: %v\n", err)
		os.Exit(1)
	}
	if err := runSimulator(*config, tmp2); err != nil {
		fmt.Fprintf(os.Stderr, "second run failed: %v\n", err)
		os.Exit(1)
	}

	b1, err := os.ReadFile(tmp1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", tmp1, err)
		os.Exit(1)
	}
	b2, err := os.ReadFile(tmp2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", tmp2, err)
		os.Exit(1)
	}

	if !bytes.Equal(b1, b2) {
		fmt.Fprintln(os.Stderr, "logs differ between runs")
		os.Exit(1)
	}

	sum := sha256.Sum256(b1)
	fmt.Printf("Determinism check passed. Log hash: %x\n", sum[:])
}

func runSimulator(config, out string) error {
	cmd := exec.Command("go", "run", "./cmd/simulator", "-config", config, "-out", out)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
