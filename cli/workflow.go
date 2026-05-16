package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type workflowEvent struct {
	RunID         string `json:"run_id"`
	TS            string `json:"ts"`
	Workflow      string `json:"workflow"`
	Step          string `json:"step"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
	ResourceID    string `json:"resource_id,omitempty"`
	ChainID       string `json:"chain_id,omitempty"`
	TxHash        string `json:"tx_hash,omitempty"`
	MessageID     string `json:"message_id,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	ManualCommand string `json:"manual_command,omitempty"`
}

type eventSink struct {
	json bool
}

func newRunID() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

func (s eventSink) emit(ev workflowEvent) {
	if ev.TS == "" {
		ev.TS = time.Now().UTC().Format(time.RFC3339)
	}
	if s.json {
		b, _ := json.Marshal(ev)
		fmt.Println(string(b))
		return
	}
	prefix := fmt.Sprintf("[%s] %s/%s %s", ev.Status, ev.Workflow, ev.Step, ev.RunID)
	if ev.Message != "" {
		fmt.Printf("%s: %s\n", prefix, ev.Message)
	} else {
		fmt.Println(prefix)
	}
	if ev.ResourceID != "" {
		fmt.Println("  resource:", ev.ResourceID)
	}
	if ev.ManualCommand != "" {
		fmt.Println("  manual:", ev.ManualCommand)
	}
}

func evidenceDir(deployment, runID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".claw1", deployment, "evidence", runID)
	return dir, os.MkdirAll(dir, 0700)
}

func writeEvidenceJSON(dir, name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), b, 0600)
}

func deploymentName(target deployTarget) string {
	if target == targetOCI {
		return "claw1demobank-oci"
	}
	return "claw1demobank"
}

func networkPath(target deployTarget) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claw1", deploymentName(target), "network.json")
}
