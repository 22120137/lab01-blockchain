package main

import "testing"

func TestRunSimulatorInvalidConfig(t *testing.T) {
	err := runSimulator("../config/does_not_exist.json", "logs/test_invalid.log")
	if err == nil {
		t.Fatalf("expected runSimulator to fail for missing config")
	}
}
