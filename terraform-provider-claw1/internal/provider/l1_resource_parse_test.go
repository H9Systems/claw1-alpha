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

RPC Endpoint:            http://127.0.0.1:49512/ext/bc/2JFz3KvHb7Wh9L/rpc
Funded address:          0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B with 1000000 (10^18)
Network Name:            Local Network

+--------+------------------------------------------------------------------+
| ewoq   | 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef |
+--------+------------------------------------------------------------------+
`

func TestParseDeployOutput_happy(t *testing.T) {
	rpcURL, key, err := parseDeployOutput(sampleOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpcURL != "http://127.0.0.1:49512/ext/bc/2JFz3KvHb7Wh9L/rpc" {
		t.Errorf("rpcURL = %q", rpcURL)
	}
	if key != "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" {
		t.Errorf("key = %q", key)
	}
}

func TestParseDeployOutput_missingRPC(t *testing.T) {
	_, _, err := parseDeployOutput("Private key: 0xdeadbeef")
	if err == nil {
		t.Fatal("expected error when RPC URL is absent")
	}
}

func TestParseDeployOutput_missingKey(t *testing.T) {
	_, _, err := parseDeployOutput("RPC URL: http://127.0.0.1:49512/ext/bc/abc/rpc")
	if err == nil {
		t.Fatal("expected error when private key is absent")
	}
}

func TestParseDeployOutput_empty(t *testing.T) {
	_, _, err := parseDeployOutput("")
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
