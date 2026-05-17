package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
)

type explorerBlock struct {
	Number       string
	Hash         string
	TxCount      int
	GasUsed      string
	Timestamp    string
	Transactions []string
}

type explorerSnapshot struct {
	BlockHeight string
	Blocks      []explorerBlock
	Err         string
}

func loadExplorerSnapshot(target deployTarget, limit int) explorerSnapshot {
	snap := loadNetworkSnapshot(target)
	if snap.net == nil {
		return explorerSnapshot{Err: "network.json not found"}
	}
	if limit <= 0 {
		limit = 5
	}
	latestHex, err := rpcString(snap.net.RPCURL, "eth_blockNumber", []any{})
	if err != nil {
		return explorerSnapshot{Err: err.Error()}
	}
	latest := hexBig(latestHex)
	out := explorerSnapshot{BlockHeight: latest.String()}
	for i := 0; i < limit && latest.Sign() >= 0; i++ {
		block := loadExplorerBlock(snap.net.RPCURL, "0x"+latest.Text(16))
		if block.Number != "" {
			out.Blocks = append(out.Blocks, block)
		}
		latest.Sub(latest, big.NewInt(1))
	}
	return out
}

func loadExplorerBlock(rpcURL, blockNum string) explorerBlock {
	var block struct {
		Number       string   `json:"number"`
		Hash         string   `json:"hash"`
		GasUsed      string   `json:"gasUsed"`
		Timestamp    string   `json:"timestamp"`
		Transactions []string `json:"transactions"`
	}
	if err := rpcJSON(rpcURL, "eth_getBlockByNumber", []any{blockNum, false}, &block); err != nil {
		return explorerBlock{}
	}
	return explorerBlock{
		Number:       hexBig(block.Number).String(),
		Hash:         block.Hash,
		TxCount:      len(block.Transactions),
		GasUsed:      hexBig(block.GasUsed).String(),
		Timestamp:    formatBlockTime(block.Timestamp),
		Transactions: block.Transactions,
	}
}

func rpcJSON(rpcURL, method string, params []any, result any) error {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method, "params": params, "id": 1})
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(rpcURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var r struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	if len(r.Error) > 0 {
		return fmt.Errorf("rpc error: %s", string(r.Error))
	}
	if len(r.Result) == 0 || string(r.Result) == "null" {
		return fmt.Errorf("empty rpc result")
	}
	return json.Unmarshal(r.Result, result)
}

func hexBig(hex string) *big.Int {
	n := new(big.Int)
	n.SetString(strings.TrimPrefix(hex, "0x"), 16)
	return n
}

func formatBlockTime(hex string) string {
	ts := hexBig(hex).Int64()
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).Format("15:04:05")
}
