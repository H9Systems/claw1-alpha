package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type cliOptions struct {
	target           deployTarget
	yes              bool
	json             bool
	dryRun           bool
	preserveEvidence bool
	evidenceBucket   string
}

func parseCommonFlags(args []string) (cliOptions, []string, error) {
	args = normalizeCommonFlags(args)
	fs := flag.NewFlagSet("claw1", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var opt cliOptions
	opt.target = targetLocal
	fs.BoolVar(&opt.yes, "yes", false, "approve destructive/non-interactive actions")
	fs.BoolVar(&opt.json, "json", false, "emit JSONL workflow events")
	fs.BoolVar(&opt.dryRun, "dry-run", false, "plan without changing infrastructure")
	fs.BoolVar(&opt.preserveEvidence, "preserve-evidence", false, "keep local evidence bundle")
	fs.StringVar(&opt.evidenceBucket, "evidence-bucket", "", "explicit OCI Object Storage bucket for evidence")
	oci := fs.Bool("oci", false, "target OCI deployment")
	local := fs.Bool("local", false, "target local deployment")
	if err := fs.Parse(args); err != nil {
		return opt, nil, err
	}
	if *oci {
		opt.target = targetOCI
	}
	if *local {
		opt.target = targetLocal
	}
	return opt, fs.Args(), nil
}

func normalizeCommonFlags(args []string) []string {
	var flags []string
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--yes", "--json", "--dry-run", "--preserve-evidence", "--oci", "--local":
			flags = append(flags, args[i])
		case "--evidence-bucket":
			flags = append(flags, args[i])
			if i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
		default:
			rest = append(rest, args[i])
		}
	}
	return append(flags, rest...)
}

func runDeployCLI(repoRoot string, args []string) int {
	opt, _, err := parseCommonFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	sink := eventSink{json: opt.json}
	runID := newRunID()
	sink.emit(workflowEvent{RunID: runID, Workflow: "deploy", Step: "start", Status: "started", Message: "programmatic deploy uses the same engine as the TUI"})
	if opt.target == targetOCI && !opt.yes {
		sink.emit(workflowEvent{RunID: runID, Workflow: "deploy", Step: "confirm", Status: "failed_closed", ErrorCode: "missing_yes", Message: "OCI deploy requires --yes in programmatic mode"})
		return 2
	}
	cfg := deployConfig{target: opt.target, repoRoot: repoRoot}
	m := newDeployModel(cfg)
	m.logCh = make(chan string, 500)
	m.advCh = make(chan int, 20)
	done := make(chan struct{})
	go func() {
		m.run()
		close(done)
	}()
	for {
		select {
		case line, ok := <-m.logCh:
			if ok {
				sink.emit(workflowEvent{RunID: runID, Workflow: "deploy", Step: "log", Status: "running", Message: line})
				continue
			}
			m.logCh = nil
		case idx := <-m.advCh:
			sink.emit(workflowEvent{RunID: runID, Workflow: "deploy", Step: fmt.Sprintf("step_%d", idx), Status: "running"})
		case <-done:
			sink.emit(workflowEvent{RunID: runID, Workflow: "deploy", Step: "complete", Status: "succeeded"})
			return 0
		}
	}
}

func runDestroyCLI(repoRoot string, args []string) int {
	opt, _, err := parseCommonFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if opt.target == targetOCI && !opt.yes {
		opt.dryRun = true
	}
	sink := eventSink{json: opt.json}
	runID := newRunID()
	deployment := deploymentName(opt.target)
	evDir, _ := evidenceDir(deployment, runID)
	sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "start", Status: "started", Message: "building Terraform and OCI inventory"})

	inv := collectDestroyInventory(repoRoot, opt.target)
	_ = writeEvidenceJSON(evDir, "pre-destroy-inventory.json", inv)
	for _, r := range inv.Resources {
		sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "inventory", Status: "found", ResourceID: r.ID, Message: r.Type})
	}
	if opt.dryRun {
		sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "dry_run", Status: "succeeded", Message: fmt.Sprintf("%d resources would be checked/destroyed", len(inv.Resources))})
		if opt.target == targetOCI && !opt.yes {
			sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "confirm", Status: "failed_closed", ErrorCode: "dry_run_default", Message: "OCI destroy defaults to dry-run; rerun with --yes to destroy"})
			return 1
		}
		return 0
	}
	if opt.target == targetOCI && !opt.yes {
		sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "confirm", Status: "failed_closed", ErrorCode: "missing_yes", Message: "OCI destroy requires --yes in programmatic mode"})
		return 2
	}
	tfDir := filepath.Join(repoRoot, "terraform")
	if opt.target == targetOCI {
		tfDir = filepath.Join(repoRoot, "terraform", "oci")
	}
	sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "terraform_destroy", Status: "running", Message: tfDir})
	if err := streamCmd(sink, runID, "destroy", tfDir, "terraform", "destroy", "-auto-approve", "-input=false"); err != nil {
		sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "terraform_destroy", Status: "failed_closed", ErrorCode: "terraform_destroy_failed", Message: err.Error()})
		return 1
	}
	post := collectDestroyInventory(repoRoot, opt.target)
	_ = writeEvidenceJSON(evDir, "post-destroy-inventory.json", post)
	if len(post.Resources) > 0 {
		for _, r := range post.Resources {
			sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "verify", Status: "requires_manual_cleanup", ResourceID: r.ID, Message: r.Type, ManualCommand: r.ManualCommand})
		}
		sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "complete", Status: "failed_closed", ErrorCode: "resources_remaining", Message: "cleanup did not verify clean"})
		return 1
	}
	retained := "local evidence: " + evDir
	if opt.evidenceBucket != "" {
		retained += "; OCI evidence bucket requested: " + opt.evidenceBucket
	} else if opt.preserveEvidence {
		retained += "; no OCI resources intentionally retained"
	}
	sink.emit(workflowEvent{RunID: runID, Workflow: "destroy", Step: "complete", Status: "succeeded", Message: retained})
	return 0
}

type destroyInventory struct {
	Target    string            `json:"target"`
	Collected string            `json:"collected_at"`
	Resources []destroyResource `json:"resources"`
	Warnings  []string          `json:"warnings,omitempty"`
}

type destroyResource struct {
	Type          string `json:"type"`
	ID            string `json:"id"`
	ManualCommand string `json:"manual_command,omitempty"`
}

func collectDestroyInventory(repoRoot string, target deployTarget) destroyInventory {
	inv := destroyInventory{Target: deploymentName(target), Collected: time.Now().UTC().Format(time.RFC3339)}
	tfDir := filepath.Join(repoRoot, "terraform")
	if target == targetOCI {
		tfDir = filepath.Join(repoRoot, "terraform", "oci")
	}
	if out, err := exec.Command("terraform", "-chdir="+tfDir, "state", "list").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			inv.Resources = append(inv.Resources, destroyResource{
				Type:          "terraform_state",
				ID:            strings.TrimSpace(line),
				ManualCommand: "terraform -chdir=" + tfDir + " destroy -target=" + strings.TrimSpace(line),
			})
		}
	} else {
		inv.Warnings = append(inv.Warnings, "terraform state list failed: "+err.Error())
	}
	if target == targetOCI {
		if _, err := exec.LookPath("oci"); err == nil {
			query := "query all resources where displayName =~ 'claw1'"
			out, err := exec.Command("oci", "search", "resource", "structured-search", "--query-text", query).CombinedOutput()
			if err == nil {
				inv.Resources = append(inv.Resources, destroyResource{
					Type:          "oci_search_result",
					ID:            strings.TrimSpace(string(out)),
					ManualCommand: "oci search resource structured-search --query-text \"" + query + "\"",
				})
			} else {
				inv.Warnings = append(inv.Warnings, "oci search failed: "+string(out))
			}
		} else {
			inv.Warnings = append(inv.Warnings, "oci CLI not found; direct OCI inventory unavailable")
		}
	}
	return inv
}

func runInspectCLI(args []string) int {
	opt, _, err := parseCommonFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	sink := eventSink{json: opt.json}
	runID := newRunID()
	path := networkPath(opt.target)
	data, err := os.ReadFile(path)
	if err != nil {
		sink.emit(workflowEvent{RunID: runID, Workflow: "inspect", Step: "network", Status: "failed_closed", ErrorCode: "network_json_missing", Message: err.Error()})
		return 1
	}
	var net networkJSON
	if err := json.Unmarshal(data, &net); err != nil {
		sink.emit(workflowEvent{RunID: runID, Workflow: "inspect", Step: "network", Status: "failed_closed", ErrorCode: "network_json_invalid", Message: err.Error()})
		return 1
	}
	sink.emit(workflowEvent{RunID: runID, Workflow: "inspect", Step: "network", Status: "succeeded", ChainID: fmt.Sprintf("%d", net.ChainID), Message: net.RPCURL})
	block, err := rpcString(net.RPCURL, "eth_blockNumber", []any{})
	if err != nil {
		sink.emit(workflowEvent{RunID: runID, Workflow: "inspect", Step: "block", Status: "failed_closed", ErrorCode: "rpc_unreachable", Message: err.Error()})
		return 1
	}
	sink.emit(workflowEvent{RunID: runID, Workflow: "inspect", Step: "block", Status: "succeeded", Message: block})
	for _, c := range net.Contracts {
		sink.emit(workflowEvent{RunID: runID, Workflow: "inspect", Step: "contract", Status: "found", ResourceID: c.Address, Message: c.Name})
	}
	return 0
}

func runWalletCLI(args []string) int {
	opt, rest, err := parseCommonFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	sink := eventSink{json: opt.json}
	runID := newRunID()
	if len(rest) == 0 || rest[0] == "list" {
		wallets := demoWallets()
		for _, w := range wallets {
			sink.emit(workflowEvent{RunID: runID, Workflow: "wallet", Step: "list", Status: "found", ResourceID: w.Address, Message: w.Name})
		}
		return 0
	}
	return 2
}

type demoWallet struct {
	Name    string
	Address string
	Unsafe  string
}

func demoWallets() []demoWallet {
	return []demoWallet{
		{Name: "deployer", Address: "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC", Unsafe: "avalanche ewoq demo key"},
		{Name: "issuer", Address: "0x0000000000000000000000000000000000001001", Unsafe: "deterministic demo label"},
		{Name: "investor", Address: "0x0000000000000000000000000000000000001002", Unsafe: "deterministic demo label"},
		{Name: "regulator", Address: "0x0000000000000000000000000000000000001003", Unsafe: "deterministic demo label"},
	}
}

func runDemoCLI(repoRoot string, args []string) int {
	opt, _, err := parseCommonFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	sink := eventSink{json: opt.json}
	runID := newRunID()
	sink.emit(workflowEvent{RunID: runID, Workflow: "demo", Step: "preflight", Status: "started", Message: "checking one-command demo prerequisites"})
	for _, bin := range []string{"terraform", "forge", "avalanche"} {
		if _, err := exec.LookPath(bin); err != nil {
			sink.emit(workflowEvent{RunID: runID, Workflow: "demo", Step: "preflight", Status: "failed_closed", ErrorCode: "missing_binary", Message: bin})
			return 1
		}
	}
	sink.emit(workflowEvent{RunID: runID, Workflow: "demo", Step: "preflight", Status: "succeeded", Message: "run `claw1` for interactive TUI or `claw1 deploy --yes` for script mode"})
	_ = repoRoot
	return 0
}

func streamCmd(sink eventSink, runID, workflow, dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) != "" {
			sink.emit(workflowEvent{RunID: runID, Workflow: workflow, Step: "log", Status: "running", Message: line})
		}
	}
	return err
}

func rpcString(rpcURL, method string, params []any) (string, error) {
	if rpcURL == "" {
		return "", errors.New("empty RPC URL")
	}
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method, "params": params, "id": 1})
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(rpcURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var r struct {
		Result string          `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	if len(r.Error) > 0 {
		return "", fmt.Errorf("rpc error: %s", string(r.Error))
	}
	return r.Result, nil
}
