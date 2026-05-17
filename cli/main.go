package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Pages ─────────────────────────────────────────────────────────────────────

type page int

const (
	pageWizard page = iota
	pageDeploy
	pageReceipt
)

// ── Root model ────────────────────────────────────────────────────────────────

type model struct {
	page     page
	width    int
	height   int
	wizard   wizardModel
	deploy   deployModel
	receipt  receiptModel
	repoRoot string
}

func initialModel(repoRoot string) model {
	return model{
		page:     pageWizard,
		width:    120,
		height:   40,
		wizard:   newWizardModel(repoRoot),
		repoRoot: repoRoot,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q", "Q":
			return m, tea.Quit
		case "enter":
			if m.page == pageWizard && m.wizard.activate() {
				cfg, err := m.wizard.validate()
				if err != nil {
					m.wizard.err = err.Error()
					return m, nil
				}
				m.wizard.err = ""
				m.deploy = newDeployModel(cfg)
				m.page = pageDeploy
				return m, m.deploy.start()
			}
			if m.page == pageDeploy && m.deploy.done && m.deploy.err == nil {
				m.receipt = newReceiptModel(m.deploy.cfg.target, m.repoRoot)
				m.page = pageReceipt
				return m, m.receipt.init()
			}
		case "c", "C":
			if m.page == pageReceipt && m.receipt.net != nil {
				return m, copyToClipboard(m.receipt.net.RPCURL)
			}
		}

	// Delegate deploy messages
	case logLineMsg, stepAdvanceMsg, deployDoneMsg:
		var cmd tea.Cmd
		m.deploy, cmd = m.deploy.Update(msg)
		return m, cmd

	// Delegate receipt messages
	case blockHeightMsg, networkLoadedMsg, tickMsg, copyDoneMsg:
		var cmd tea.Cmd
		m.receipt, cmd = m.receipt.Update(msg)
		return m, cmd
	}

	// Forward keyboard to wizard when on that page
	if m.page == pageWizard {
		var cmd tea.Cmd
		m.wizard, cmd = m.wizard.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	switch m.page {
	case pageWizard:
		return m.wizard.View(m.width)
	case pageDeploy:
		return m.deploy.View(m.width)
	case pageReceipt:
		return m.receipt.View(m.width)
	}
	return ""
}

// ── Entry ─────────────────────────────────────────────────────────────────────

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: run claw1 from within the claw1-alpha repository")
		fmt.Fprintln(os.Stderr, "  (could not find terraform/ and contracts/ directories)")
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "deploy":
			os.Exit(runDeployCLI(repoRoot, os.Args[2:]))
		case "destroy":
			os.Exit(runDestroyCLI(repoRoot, os.Args[2:]))
		case "inspect":
			os.Exit(runInspectCLI(os.Args[2:]))
		case "wallet":
			os.Exit(runWalletCLI(os.Args[2:]))
		case "demo":
			os.Exit(runDemoCLI(repoRoot, os.Args[2:]))
		case "upgrade":
			os.Exit(runUpgradeCLI(repoRoot, os.Args[2:]))
		case "version", "--version", "-v":
			fmt.Println("claw1 " + version)
			os.Exit(0)
		}
	}

	// Sub-command: `claw1 receipt` — jump straight to receipt view
	if len(os.Args) > 1 && os.Args[1] == "receipt" {
		target := targetLocal
		if len(os.Args) > 2 && os.Args[2] == "--oci" {
			target = targetOCI
		}
		m := model{
			page:     pageReceipt,
			width:    120,
			height:   40,
			receipt:  newReceiptModel(target, repoRoot),
			repoRoot: repoRoot,
		}
		p := tea.NewProgram(m, tea.WithAltScreen())
		p.Send(nil) // trigger init via receipt.init() in Init()
		// We need to call receipt init cmd manually
		m2 := model{
			page:     pageReceipt,
			width:    120,
			height:   40,
			receipt:  newReceiptModel(target, repoRoot),
			repoRoot: repoRoot,
		}
		prog := tea.NewProgram(receiptOnlyModel{m2.receipt}, tea.WithAltScreen())
		if _, err := prog.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	p := tea.NewProgram(initialModel(repoRoot), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// receiptOnlyModel wraps receiptModel for direct launch.
type receiptOnlyModel struct {
	r receiptModel
}

func (m receiptOnlyModel) Init() tea.Cmd { return m.r.init() }
func (m receiptOnlyModel) View() string  { return m.r.View(120) }
func (m receiptOnlyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		k := msg.(tea.KeyMsg)
		if k.String() == "q" || k.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if k.String() == "c" && m.r.net != nil {
			return m, copyToClipboard(m.r.net.RPCURL)
		}
	case tea.WindowSizeMsg:
		return m, nil
	}
	var cmd tea.Cmd
	m.r, cmd = m.r.Update(msg)
	return m, cmd
}

// ── Utilities ─────────────────────────────────────────────────────────────────

func findRepoRoot() (string, error) {
	// Walk up from CWD looking for terraform/ + contracts/
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "terraform")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "contracts")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found")
}

func copyToClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		// Try xclip, xsel, pbcopy in order
		for _, args := range [][]string{
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
			{"pbcopy"},
		} {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdin = strings.NewReader(s)
			if err := cmd.Run(); err == nil {
				return copyDoneMsg("RPC URL copied to clipboard")
			}
		}
		return copyDoneMsg("copy failed — paste manually: " + s)
	}
}
