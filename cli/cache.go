package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// wizardCache holds all async-fetched data used by wizard View functions.
// View() reads only from this cache — never makes I/O calls directly.
type wizardCache struct {
	// Network (from file)
	snap networkSnapshot

	// Block height (RPC)
	blockHeight string
	blockErr    string

	// Explorer snapshot (RPC, multiple calls)
	explorer explorerSnapshot

	// Transfers — unfiltered (explorer tab)
	allTransfers    []trexTransfer
	allTransfersErr string

	// Transfers — sender-filtered (overview / T-REX tab)
	senderTransfers    []trexTransfer
	senderTransfersErr string

	// T-REX sender info (RPC + cast)
	senderBalance string
	senderNonce   string
	senderCEQ     string

	// T-REX simulation for current cursor
	simulation trexSimulation
	simCursor  int
	simDone    bool

	// T-REX per-recipient info (cast)
	recipientVerified map[string]bool
	recipientCEQ      map[string]string
}

// ── Async result messages ─────────────────────────────────────────────────────

type cacheNetMsg struct{ snap networkSnapshot }

type cacheBlockMsg struct{ height, errStr string }

type cacheExplorerMsg struct{ snap explorerSnapshot }

type cacheAllTransfersMsg struct {
	transfers []trexTransfer
	errStr    string
}

type cacheSenderTransfersMsg struct {
	transfers []trexTransfer
	errStr    string
}

type cacheSenderInfoMsg struct{ balance, nonce, ceq string }

type cacheRecipientInfoMsg struct {
	verified map[string]bool
	ceq      map[string]string
}

type cacheSimulationMsg struct {
	sim    trexSimulation
	cursor int
}

type cacheSendResultMsg struct{ action string }

type wizardTickMsg time.Time

// receiptExplorerMsg carries async explorer data for the receipt page.
type receiptExplorerMsg struct{ snap explorerSnapshot }

// ── Fetch commands ────────────────────────────────────────────────────────────

func fetchNetworkCmd(target deployTarget) tea.Cmd {
	return func() tea.Msg {
		return cacheNetMsg{snap: loadNetworkSnapshot(target)}
	}
}

func fetchBlockHeightCmd(rpcURL string) tea.Cmd {
	return func() tea.Msg {
		val, err := rpcString(rpcURL, "eth_blockNumber", []any{})
		if err != nil {
			return cacheBlockMsg{errStr: err.Error()}
		}
		return cacheBlockMsg{height: hexBig(val).String()}
	}
}

func fetchExplorerCmd(target deployTarget) tea.Cmd {
	return func() tea.Msg {
		return cacheExplorerMsg{snap: loadExplorerSnapshot(target, 6)}
	}
}

func fetchAllTransfersCmd(target deployTarget) tea.Cmd {
	return func() tea.Msg {
		transfers, err := trexTransferHistory(target, "", 8)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		return cacheAllTransfersMsg{transfers: transfers, errStr: errStr}
	}
}

func fetchSenderTransfersCmd(target deployTarget, sender string) tea.Cmd {
	return func() tea.Msg {
		transfers, err := trexTransferHistory(target, sender, 6)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		return cacheSenderTransfersMsg{transfers: transfers, errStr: errStr}
	}
}

func fetchSenderInfoCmd(rpcURL, token, senderAddr string) tea.Cmd {
	return func() tea.Msg {
		balance := walletBalance(rpcURL, senderAddr)
		nonce := walletNonce(rpcURL, senderAddr)
		ceq := "n/a"
		if token != "" {
			ceq = trexBalance(rpcURL, token, senderAddr)
		}
		return cacheSenderInfoMsg{balance: balance, nonce: nonce, ceq: ceq}
	}
}

func fetchRecipientInfoCmd(rpcURL, token, identity string, recipients []trexRecipient) tea.Cmd {
	return func() tea.Msg {
		verified := make(map[string]bool, len(recipients))
		ceq := make(map[string]string, len(recipients))
		for _, r := range recipients {
			verified[r.Address] = trexIsVerified(rpcURL, identity, r.Address)
			if token != "" {
				ceq[r.Address] = trexBalance(rpcURL, token, r.Address)
			} else {
				ceq[r.Address] = "n/a"
			}
		}
		return cacheRecipientInfoMsg{verified: verified, ceq: ceq}
	}
}

func fetchSimulationCmd(target deployTarget, recipientAddr string, cursor int) tea.Cmd {
	return func() tea.Msg {
		sim := simulateTrexTransfer(target, recipientAddr, "1")
		return cacheSimulationMsg{sim: sim, cursor: cursor}
	}
}

func asyncSendTransferCmd(target deployTarget, recipientAddr string) tea.Cmd {
	return func() tea.Msg {
		sim := simulateTrexTransfer(target, recipientAddr, "1")
		if !sim.Approved {
			return cacheSendResultMsg{action: sim.Message}
		}
		tx, err := sendTrexTransfer(target, recipientAddr, "1")
		if err != nil {
			return cacheSendResultMsg{action: "Transfer failed: " + oneLine(err.Error(), 120)}
		}
		return cacheSendResultMsg{action: "Sent 1 CEQ to " + shortAddr(recipientAddr) + " tx " + shortAddr(tx)}
	}
}

func fetchReceiptExplorerCmd(target deployTarget) tea.Cmd {
	return func() tea.Msg {
		return receiptExplorerMsg{snap: loadExplorerSnapshot(target, 6)}
	}
}

func wizardTickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return wizardTickMsg(t)
	})
}

// refreshTabData returns commands to fetch data for the given tab.
// Requires a non-nil network snapshot for RPC endpoints.
func refreshTabData(tab int, target deployTarget, snap networkSnapshot, trexCursor int) tea.Cmd {
	if snap.net == nil {
		return nil
	}
	rpcURL := snap.net.RPCURL

	// Fetch block height — useful for wizard status bar and receipt
	return fetchBlockHeightCmd(rpcURL)
}
