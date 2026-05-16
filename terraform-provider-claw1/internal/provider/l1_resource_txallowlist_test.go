package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInjectTxAllowList_writesConfigPrecompile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	genesisDir := filepath.Join(home, ".avalanche-cli", "subnets", "demo")
	if err := os.MkdirAll(genesisDir, 0700); err != nil {
		t.Fatal(err)
	}
	genesisPath := filepath.Join(genesisDir, "genesis.json")
	if err := os.WriteFile(genesisPath, []byte(`{"config":{"chainId":432260},"alloc":{}}`), 0600); err != nil {
		t.Fatal(err)
	}

	admin := "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	if err := injectTxAllowList("demo", admin); err != nil {
		t.Fatalf("injectTxAllowList returned error: %v", err)
	}

	data, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatal(err)
	}
	var genesis map[string]interface{}
	if err := json.Unmarshal(data, &genesis); err != nil {
		t.Fatal(err)
	}

	config := genesis["config"].(map[string]interface{})
	txAllowList := config["txAllowListConfig"].(map[string]interface{})
	if txAllowList["blockTimestamp"].(float64) != 0 {
		t.Fatalf("blockTimestamp = %v", txAllowList["blockTimestamp"])
	}
	admins := txAllowList["adminAddresses"].([]interface{})
	if len(admins) != 1 || admins[0] != admin {
		t.Fatalf("adminAddresses = %#v", admins)
	}
	if _, ok := genesis["txAllowListConfig"]; ok {
		t.Fatal("txAllowListConfig must be nested under config, not written at genesis root")
	}
}

func TestInjectTxAllowList_createsConfigWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	genesisDir := filepath.Join(home, ".avalanche-cli", "subnets", "demo")
	if err := os.MkdirAll(genesisDir, 0700); err != nil {
		t.Fatal(err)
	}
	genesisPath := filepath.Join(genesisDir, "genesis.json")
	if err := os.WriteFile(genesisPath, []byte(`{"alloc":{}}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := injectTxAllowList("demo", "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"); err != nil {
		t.Fatalf("injectTxAllowList returned error: %v", err)
	}

	data, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatal(err)
	}
	var genesis map[string]interface{}
	if err := json.Unmarshal(data, &genesis); err != nil {
		t.Fatal(err)
	}
	config := genesis["config"].(map[string]interface{})
	if _, ok := config["txAllowListConfig"]; !ok {
		t.Fatal("txAllowListConfig missing under generated config object")
	}
}

func TestEncodeReadAllowListCall(t *testing.T) {
	got, err := encodeReadAllowListCall("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")
	if err != nil {
		t.Fatalf("encodeReadAllowListCall returned error: %v", err)
	}
	want := "0xeb54dae10000000000000000000000008db97c7cece249c2b98bdc0226cc4c2a57bf52fc"
	if got != want {
		t.Fatalf("call data = %s, want %s", got, want)
	}
}

func TestEncodeReadAllowListCall_rejectsInvalidAddress(t *testing.T) {
	if _, err := encodeReadAllowListCall("0x1234"); err == nil {
		t.Fatal("expected invalid address error")
	}
}
