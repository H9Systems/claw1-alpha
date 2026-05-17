package main

import (
	"fmt"
	"math/big"
	"os/exec"
	"strings"
)

const (
	trexTransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	ceqDecimals       = 18
)

type trexTransfer struct {
	Block  string
	TxHash string
	From   string
	To     string
	Amount string
}

type trexRecipient struct {
	Name    string
	Address string
}

type trexSimulation struct {
	Approved bool
	Message  string
	Token    string
	To       string
	Amount   string
}

func trexTokenAddress(net *networkJSON) string {
	return findLatestContract(net, "ERC3643Token", "CEQ_Token")
}

func identityRegistryAddress(net *networkJSON) string {
	return findLatestContract(net, "IdentityRegistry")
}

func findLatestContract(net *networkJSON, names ...string) string {
	if net == nil {
		return ""
	}
	for i := len(net.Contracts) - 1; i >= 0; i-- {
		for _, name := range names {
			if net.Contracts[i].Name == name {
				return net.Contracts[i].Address
			}
		}
	}
	return ""
}

func trexRecipients() []trexRecipient {
	return []trexRecipient{
		{Name: "verified demo investor", Address: "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"},
		{Name: "unverified wallet", Address: "0x1111111111111111111111111111111111111111"},
		{Name: "issuer label", Address: "0x0000000000000000000000000000000000001001"},
		{Name: "regulator label", Address: "0x0000000000000000000000000000000000001003"},
	}
}

func selectedTrexRecipient(cursor int) trexRecipient {
	recipients := trexRecipients()
	if len(recipients) == 0 {
		return trexRecipient{}
	}
	if cursor < 0 || cursor >= len(recipients) {
		cursor = 0
	}
	return recipients[cursor]
}

func trexBalance(rpcURL, token, address string) string {
	out, err := exec.Command("cast", "call", token, "balanceOf(address)(uint256)", address, "--rpc-url", rpcURL).CombinedOutput()
	if err != nil {
		return "unreachable"
	}
	return formatCEQ(parseUintOutput(string(out)).String()) + " CEQ"
}

func trexIsVerified(rpcURL, identity, address string) bool {
	if identity == "" {
		return false
	}
	out, err := exec.Command("cast", "call", identity, "isVerified(address)(bool)", address, "--rpc-url", rpcURL).CombinedOutput()
	if err != nil {
		return false
	}
	result := strings.TrimSpace(string(out))
	return result == "true" || strings.HasSuffix(result, "1")
}

func trexPaused(rpcURL, token string) bool {
	out, err := exec.Command("cast", "call", token, "paused()(bool)", "--rpc-url", rpcURL).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func simulateTrexTransfer(target deployTarget, to, amount string) trexSimulation {
	snap := loadNetworkSnapshot(target)
	if snap.net == nil {
		return trexSimulation{Message: "Deploy a network first."}
	}
	token := trexTokenAddress(snap.net)
	identity := identityRegistryAddress(snap.net)
	if token == "" {
		return trexSimulation{Message: "T-REX token not found in network.json."}
	}
	if to == "" {
		to = selectedTrexRecipient(0).Address
	}
	wei, err := ceqToWei(amount)
	if err != nil {
		return trexSimulation{Token: token, To: to, Amount: amount, Message: err.Error()}
	}
	if trexPaused(snap.net.RPCURL, token) {
		return trexSimulation{Token: token, To: to, Amount: amount, Message: "Rejected: CEQ token is paused."}
	}
	balance := parseUintOutput(rawTrexBalance(snap.net.RPCURL, token, demoWallets()[0].Address))
	needed := parseUintOutput(wei)
	if balance.Cmp(needed) < 0 {
		return trexSimulation{Token: token, To: to, Amount: amount, Message: "Rejected: deployer CEQ balance is too low."}
	}
	if identity != "" && !trexIsVerified(snap.net.RPCURL, identity, to) {
		return trexSimulation{Token: token, To: to, Amount: amount, Message: "Rejected: recipient is not verified in IdentityRegistry."}
	}
	out, err := exec.Command("cast", "call", token, "transfer(address,uint256)(bool)", to, wei, "--from", demoWallets()[0].Address, "--rpc-url", snap.net.RPCURL).CombinedOutput()
	if err != nil {
		return trexSimulation{Token: token, To: to, Amount: amount, Message: "Rejected: " + trexRevertReason(strings.TrimSpace(string(out)))}
	}
	return trexSimulation{Approved: true, Token: token, To: to, Amount: amount, Message: "Approved: transfer can execute on the selected L1."}
}

func sendTrexTransfer(target deployTarget, to, amount string) (string, error) {
	snap := loadNetworkSnapshot(target)
	if snap.net == nil {
		return "", fmt.Errorf("deploy a network first")
	}
	token := trexTokenAddress(snap.net)
	if token == "" {
		return "", fmt.Errorf("T-REX token not found in network.json")
	}
	wei, err := ceqToWei(amount)
	if err != nil {
		return "", err
	}
	cmd := exec.Command("cast", "send",
		token,
		"transfer(address,uint256)",
		to,
		wei,
		"--rpc-url", snap.net.RPCURL,
		"--private-key", hexPrivateKey(snap.net.DeployerPrivateKey),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return parseTrexTxHash(string(out)), nil
}

func rawTrexBalance(rpcURL, token, address string) string {
	out, err := exec.Command("cast", "call", token, "balanceOf(address)(uint256)", address, "--rpc-url", rpcURL).CombinedOutput()
	if err != nil {
		return "0"
	}
	return strings.TrimSpace(string(out))
}

func trexTransferHistory(target deployTarget, addressFilter string, limit int) ([]trexTransfer, error) {
	snap := loadNetworkSnapshot(target)
	if snap.net == nil {
		return nil, fmt.Errorf("network.json not found")
	}
	token := trexTokenAddress(snap.net)
	if token == "" {
		return nil, fmt.Errorf("T-REX token not found")
	}
	type rpcLog struct {
		Topics          []string `json:"topics"`
		Data            string   `json:"data"`
		BlockNumber     string   `json:"blockNumber"`
		TransactionHash string   `json:"transactionHash"`
	}
	params := []any{map[string]any{
		"fromBlock": "0x0",
		"toBlock":   "latest",
		"address":   token,
		"topics":    []any{trexTransferTopic},
	}}
	var logs []rpcLog
	if err := rpcJSON(snap.net.RPCURL, "eth_getLogs", params, &logs); err != nil {
		return nil, err
	}
	filter := strings.ToLower(strings.TrimSpace(addressFilter))
	var transfers []trexTransfer
	for i := len(logs) - 1; i >= 0; i-- {
		log := logs[i]
		if len(log.Topics) < 3 {
			continue
		}
		from := topicAddress(log.Topics[1])
		to := topicAddress(log.Topics[2])
		if filter != "" && strings.ToLower(from) != filter && strings.ToLower(to) != filter {
			continue
		}
		transfers = append(transfers, trexTransfer{
			Block:  hexBig(log.BlockNumber).String(),
			TxHash: log.TransactionHash,
			From:   from,
			To:     to,
			Amount: formatCEQ(hexBig(log.Data).String()),
		})
		if limit > 0 && len(transfers) >= limit {
			break
		}
	}
	return transfers, nil
}

func topicAddress(topic string) string {
	t := strings.TrimPrefix(topic, "0x")
	if len(t) < 40 {
		return "0x" + t
	}
	return "0x" + strings.ToLower(t[len(t)-40:])
}

func parseUintOutput(output string) *big.Int {
	s := strings.TrimSpace(output)
	if strings.HasPrefix(s, "0x") {
		return hexBig(s)
	}
	fields := strings.Fields(s)
	if len(fields) > 0 {
		s = fields[0]
	}
	n := new(big.Int)
	if _, ok := n.SetString(strings.TrimPrefix(s, "0x"), 10); ok {
		return n
	}
	return big.NewInt(0)
}

func ceqToWei(amount string) (string, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return "", fmt.Errorf("amount is required")
	}
	r, ok := new(big.Rat).SetString(amount)
	if !ok || r.Sign() <= 0 {
		return "", fmt.Errorf("amount must be a positive CEQ value")
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(ceqDecimals), nil)
	r.Mul(r, new(big.Rat).SetInt(scale))
	if !r.IsInt() {
		return "", fmt.Errorf("amount has more than 18 decimal places")
	}
	return r.Num().String(), nil
}

func formatCEQ(wei string) string {
	n := parseUintOutput(wei)
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(ceqDecimals), nil)
	r := new(big.Rat).SetFrac(n, scale)
	return r.FloatString(4)
}

func parseWalletArgs(args []string) (to string, amount string) {
	amount = "1"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--to":
			if i+1 < len(args) {
				to = args[i+1]
				i++
			}
		case "--amount":
			if i+1 < len(args) {
				amount = args[i+1]
				i++
			}
		}
	}
	return to, amount
}

func parseTrexTxHash(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "transactionHash") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

func trexRevertReason(errMsg string) string {
	lower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(lower, "identity") && strings.Contains(lower, "not found"):
		return "identity not found in IdentityRegistry"
	case strings.Contains(lower, "notverified") || strings.Contains(lower, "not verified"):
		return "wallet not KYC-verified"
	case strings.Contains(lower, "transfernotpossible") || strings.Contains(lower, "transfer not possible"):
		return "T-REX compliance rule blocked transfer"
	default:
		if len(errMsg) > 120 {
			return errMsg[:120]
		}
		return errMsg
	}
}
