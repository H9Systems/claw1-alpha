package provider

import "testing"

// sampleOutput is the exact stdout format from avalanche blockchain deploy --local (v1.9.6).
// v1.9.6 changes: "RPC Endpoint:" label (was "RPC URL:"), ewoq key shown in a table
// without 0x prefix (was "Private key: 0x...").
const sampleOutput = `
Deploying [claw1demobank] to Local Network
Backend controller started, call graph Node URIs to interact with the chain
Waiting for the network to be ready...
Chain ID: 432260
VM ID: cjydMFhHTpGEiHW4bEJLwKi7jLQ4JXAT

Subnet ID: 2CZp5JsBfELt45EKVBkJCGtV2XNLZ
Blockchain ID: 2JFz3KvHb7Wh9L
RPC Endpoint:            http://127.0.0.1:49512/ext/bc/2JFz3KvHb7Wh9L/rpc
Funded address:          0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B with 1000000 (10^18)
Network Name:            Local Network

+--------+------------------------------------------------------------------+
| ewoq   | 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef |
+--------+------------------------------------------------------------------+
`

func TestParseDeployOutput_happy(t *testing.T) {
	rpcURL, key, blockchainID, subnetID, err := parseDeployOutput(sampleOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpcURL != "http://127.0.0.1:49512/ext/bc/2JFz3KvHb7Wh9L/rpc" {
		t.Errorf("rpcURL = %q", rpcURL)
	}
	if key != "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" {
		t.Errorf("key = %q", key)
	}
	if blockchainID != "2JFz3KvHb7Wh9L" {
		t.Errorf("blockchainID = %q, want %q", blockchainID, "2JFz3KvHb7Wh9L")
	}
	if subnetID != "2CZp5JsBfELt45EKVBkJCGtV2XNLZ" {
		t.Errorf("subnetID = %q, want %q", subnetID, "2CZp5JsBfELt45EKVBkJCGtV2XNLZ")
	}
}

func TestParseDeployOutput_blockchainIDFromURL(t *testing.T) {
	// BlockchainID is always derivable from the RPC URL path, even without an explicit line.
	out := `
RPC Endpoint: http://127.0.0.1:9650/ext/bc/TestChainABC123/rpc
+--------+------------------------------------------------------------------+
| ewoq   | aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa |
+--------+------------------------------------------------------------------+
`
	_, _, blockchainID, _, err := parseDeployOutput(out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blockchainID != "TestChainABC123" {
		t.Errorf("blockchainID = %q, want %q", blockchainID, "TestChainABC123")
	}
}

func TestParseDeployOutput_missingRPC(t *testing.T) {
	_, _, _, _, err := parseDeployOutput("Private key: 0xdeadbeef")
	if err == nil {
		t.Fatal("expected error when RPC URL is absent")
	}
}

func TestParseDeployOutput_missingKey(t *testing.T) {
	_, _, _, _, err := parseDeployOutput("RPC URL: http://127.0.0.1:49512/ext/bc/abc/rpc")
	if err == nil {
		t.Fatal("expected error when private key is absent")
	}
}

func TestParseDeployOutput_empty(t *testing.T) {
	_, _, _, _, err := parseDeployOutput("")
	if err == nil {
		t.Fatal("expected error on empty output")
	}
}

func TestRpcRe_doesNotMatchPartial(t *testing.T) {
	// Must not match a line that has no port number
	out := "RPC URL: http://127.0.0.1/ext/bc/abc/rpc"
	m := rpcRe.FindStringSubmatch(out)
	if len(m) == 2 {
		t.Errorf("should not match URL without port, got %q", m[1])
	}
}
