package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const sampleForgeOutput = `
[⠒] Compiling...
No files changed, compilation skipped
Deployer: 0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC
Deployed to: 0xAbCd1234567890AbCd1234567890AbCd12345678
Transaction hash: 0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef
`

func TestDeployedToRe_happy(t *testing.T) {
	m := deployedToRe.FindStringSubmatch(sampleForgeOutput)
	if len(m) != 2 {
		t.Fatalf("regex did not match, got %v", m)
	}
	if m[1] != "0xAbCd1234567890AbCd1234567890AbCd12345678" {
		t.Errorf("address = %q", m[1])
	}
}

func TestDeployedToRe_noMatch(t *testing.T) {
	m := deployedToRe.FindStringSubmatch("forge error: reverted")
	if len(m) != 0 {
		t.Fatalf("regex should not match error output, got %v", m)
	}
}

func TestAppendContractToNetworkJSON(t *testing.T) {
	home := t.TempDir()
	l1Name := "testnet"
	netDir := filepath.Join(home, l1Name)
	if err := os.MkdirAll(netDir, 0700); err != nil {
		t.Fatal(err)
	}

	initial := networkJSON{
		Name:    l1Name,
		ChainID: 432260,
		RPCURL:  "http://127.0.0.1:9650/ext/bc/abc/rpc",
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	netPath := filepath.Join(netDir, "network.json")
	if err := os.WriteFile(netPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	r := &contractResource{cfg: &ProviderConfig{DataDir: home}}
	if err := r.appendContractToNetworkJSON(l1Name, "MyToken", "0xDEAD"); err != nil {
		t.Fatalf("appendContractToNetworkJSON: %v", err)
	}

	result, err := os.ReadFile(netPath)
	if err != nil {
		t.Fatal(err)
	}
	var net networkJSON
	if err := json.Unmarshal(result, &net); err != nil {
		t.Fatal(err)
	}

	if len(net.Contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(net.Contracts))
	}
	if net.Contracts[0].Name != "MyToken" {
		t.Errorf("name = %q", net.Contracts[0].Name)
	}
	if net.Contracts[0].Address != "0xDEAD" {
		t.Errorf("address = %q", net.Contracts[0].Address)
	}
	// DeployedAt should be parseable and recent.
	ts, err := time.Parse(time.RFC3339, net.Contracts[0].DeployedAt)
	if err != nil {
		t.Fatalf("deployedAt parse error: %v", err)
	}
	if time.Since(ts) > 5*time.Second {
		t.Errorf("deployedAt too far in the past: %v", ts)
	}
}
