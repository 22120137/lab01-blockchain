package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
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
	s1, err := stateHashFromLog(tmp1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "state hash run1: %v\n", err)
		os.Exit(1)
	}
	s2, err := stateHashFromLog(tmp2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "state hash run2: %v\n", err)
		os.Exit(1)
	}
	if !bytes.Equal(s1, s2) {
		fmt.Fprintln(os.Stderr, "final state differs between runs")
		os.Exit(1)
	}

	sum := sha256.Sum256(b1)
	fmt.Printf("Determinism check passed. Log hash: %x State hash: %x\n", sum[:], s1)
}

func runSimulator(config, out string) error {
	cmd := exec.Command("go", "run", "./cmd/simulator", "-config", config, "-out", out)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stateHashFromLog(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var stateLine string
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "|STATE|"); idx != -1 {
			stateLine = line[idx+len("|STATE|"):]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if stateLine == "" {
		return nil, fmt.Errorf("no STATE line found")
	}
	parts := strings.SplitN(stateLine, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed STATE line")
	}
	var payload interface{}
	if err := json.Unmarshal([]byte(parts[1]), &payload); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	return sum[:], nil
}
