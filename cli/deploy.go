package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Messages ─────────────────────────────────────────────────────────────────

type logLineMsg string
type stepAdvanceMsg int
type deployDoneMsg struct{ err error }

// ── Deploy step ───────────────────────────────────────────────────────────────

type stepStatus int

const (
	stepWaiting stepStatus = iota
	stepRunning
	stepDone
	stepFailed
)

type deployStep struct {
	name    string
	status  stepStatus
	elapsed time.Duration
	started time.Time
}

// ── Model ─────────────────────────────────────────────────────────────────────

type deployModel struct {
	steps   []deployStep
	logs    []string
	maxLogs int
	logCh   chan string
	advCh   chan int
	err     error
	cfg     deployConfig
	done    bool
}

func newDeployModel(cfg deployConfig) deployModel {
	steps := []deployStep{
		{name: "Write credentials"},
		{name: "Provision OCI infrastructure"},
		{name: "Bootstrap Avalanche L1"},
		{name: "Deploy compliance contracts"},
	}
	if cfg.target == targetLocal {
		steps = []deployStep{
			{name: "Build Terraform provider"},
			{name: "Deploy Avalanche L1"},
			{name: "Deploy compliance contracts"},
		}
	}
	return deployModel{
		steps:   steps,
		maxLogs: 200,
		logCh:   make(chan string, 200),
		advCh:   make(chan int, 10),
		cfg:     cfg,
	}
}

// start kicks off the deploy goroutine and returns the first waitForLog cmd.
func (m *deployModel) start() tea.Cmd {
	go m.run()
	return waitForLog(m.logCh, m.advCh)
}

func waitForLog(logCh chan string, advCh chan int) tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-logCh:
			if !ok {
				return deployDoneMsg{}
			}
			return logLineMsg(line)
		case step := <-advCh:
			return stepAdvanceMsg(step)
		}
	}
}

func (m deployModel) Update(msg tea.Msg) (deployModel, tea.Cmd) {
	switch msg := msg.(type) {
	case logLineMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > m.maxLogs {
			m.logs = m.logs[len(m.logs)-m.maxLogs:]
		}
		return m, waitForLog(m.logCh, m.advCh)

	case stepAdvanceMsg:
		idx := int(msg)
		if idx < len(m.steps) {
			if idx > 0 {
				prev := idx - 1
				m.steps[prev].status = stepDone
				m.steps[prev].elapsed = time.Since(m.steps[prev].started)
			}
			m.steps[idx].status = stepRunning
			m.steps[idx].started = time.Now()
		}
		return m, waitForLog(m.logCh, m.advCh)

	case deployDoneMsg:
		m.done = true
		if msg.err != nil {
			m.err = msg.err
			for i := range m.steps {
				if m.steps[i].status == stepRunning {
					m.steps[i].status = stepFailed
				}
			}
		} else {
			for i := range m.steps {
				if m.steps[i].status == stepRunning {
					m.steps[i].status = stepDone
					m.steps[i].elapsed = time.Since(m.steps[i].started)
				}
			}
		}
	}
	return m, nil
}

func (m deployModel) View(width int) string {
	var b strings.Builder

	targetLabel := "OCI DEPLOYMENT"
	if m.cfg.target == targetLocal {
		targetLabel = "LOCAL DEPLOYMENT"
	}
	b.WriteString(styleHeader.Render("CLAW1") + "  " +
		styleDim.Render(targetLabel) + "\n\n")

	// Steps
	for i, s := range m.steps {
		switch s.status {
		case stepWaiting:
			b.WriteString("  " + circle() + "  " + styleDim.Render(s.name) +
				styleDim.Render("   ─────────────────  waiting") + "\n")
		case stepRunning:
			elapsed := time.Since(s.started).Round(time.Second)
			b.WriteString("  " + dot(yellow) + "  " + styleYellow.Render(s.name) +
				styleDim.Render(fmt.Sprintf("   running  %s", elapsed)) + "\n")
		case stepDone:
			b.WriteString("  " + dot(green) + "  " + styleGreen.Render(s.name) +
				styleDim.Render(fmt.Sprintf("   done  %s", s.elapsed.Round(time.Second))) + "\n")
		case stepFailed:
			b.WriteString("  " + dot(red) + "  " + styleRed.Render(s.name) +
				styleDim.Render("   FAILED") + "\n")
		}
		_ = i
	}

	// Log panel
	b.WriteString("\n" + styleSectionTitle.Render("LOG") + "\n")
	logHeight := 12
	start := 0
	if len(m.logs) > logHeight {
		start = len(m.logs) - logHeight
	}
	for _, line := range m.logs[start:] {
		// Truncate long lines
		if len(line) > width-8 {
			line = line[:width-8]
		}
		b.WriteString(styleDim.Render("  "+line) + "\n")
	}

	if m.err != nil {
		b.WriteString("\n" + styleRed.Render("  ✗ "+m.err.Error()) + "\n")
	}

	if m.done && m.err == nil {
		b.WriteString("\n" + styleGreen.Render("  ✓ Deployment complete — press Enter to view Sovereignty Receipt") + "\n")
	}

	b.WriteString(styleKeys.Render("\n  [Q] quit"))

	return styleBox.Width(width - 4).Render(b.String())
}

// ── Orchestration ─────────────────────────────────────────────────────────────

func (m *deployModel) run() {
	if m.cfg.target == targetOCI {
		m.runOCI()
	} else {
		m.runLocal()
	}
}

func (m *deployModel) runOCI() {
	// Step 0: write credentials
	m.advCh <- 0
	if err := writeOCIConfig(m.cfg); err != nil {
		m.logCh <- "[claw1] ERROR: " + err.Error()
		close(m.logCh)
		return
	}
	if err := writeTFVars(m.cfg); err != nil {
		m.logCh <- "[claw1] ERROR: " + err.Error()
		close(m.logCh)
		return
	}
	m.logCh <- "[claw1] Credentials written"

	// Step 1+2: terraform init + apply (provisions VM + bootstraps L1)
	m.advCh <- 1
	tfDir := filepath.Join(m.cfg.repoRoot, "terraform", "oci")
	if err := m.runCmd(tfDir, "terraform", "init", "-input=false"); err != nil {
		m.logCh <- "[terraform] init failed: " + err.Error()
		close(m.logCh)
		return
	}
	// Watch for bootstrap complete to advance step
	go func() {
		// terraform apply runs next; we watch logs for L1 ready signal
	}()
	if err := m.runCmd(tfDir, "terraform", "apply", "-auto-approve", "-input=false"); err != nil {
		m.logCh <- "[terraform] apply failed: " + err.Error()
		close(m.logCh)
		return
	}
	m.advCh <- 2 // L1 done (bootstrap ran inside terraform apply)
	m.advCh <- 3 // contracts

	// Step 3: contracts via run.sh --oci
	if err := m.runCmd(m.cfg.repoRoot, "bash", "run.sh", "--oci"); err != nil {
		m.logCh <- "[run.sh] contract deploy failed: " + err.Error()
		close(m.logCh)
		return
	}

	close(m.logCh)
}

func (m *deployModel) runLocal() {
	m.advCh <- 0
	providerDir := filepath.Join(m.cfg.repoRoot, "terraform-provider-claw1")
	if err := m.runCmd(providerDir, "make", "install"); err != nil {
		m.logCh <- "[make] install failed: " + err.Error()
		close(m.logCh)
		return
	}

	m.advCh <- 1
	tfDir := filepath.Join(m.cfg.repoRoot, "terraform")
	// Remove stale lock file so init regenerates it
	os.Remove(filepath.Join(tfDir, ".terraform.lock.hcl"))
	if err := m.runCmd(tfDir, "terraform", "init", "-upgrade", "-input=false"); err != nil {
		m.logCh <- "[terraform] init failed: " + err.Error()
		close(m.logCh)
		return
	}

	m.advCh <- 2
	if err := m.runCmd(tfDir, "terraform", "apply", "-auto-approve", "-input=false"); err != nil {
		m.logCh <- "[terraform] apply failed: " + err.Error()
		close(m.logCh)
		return
	}

	close(m.logCh)
}

// runCmd runs a command and streams its output line by line to logCh.
func (m *deployModel) runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	// Merge PATH so homebrew/snap binaries are found
	cmd.Env = append(os.Environ(),
		"HOME="+os.Getenv("HOME"),
		"PATH="+os.Getenv("PATH")+":/snap/bin:/usr/local/bin",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	scan := func(r interface{ Scan() bool; Text() string }) {
		defer wg.Done()
		for r.Scan() {
			m.logCh <- r.Text()
		}
	}
	go scan(bufio.NewScanner(stdout))
	go scan(bufio.NewScanner(stderr))
	wg.Wait()
	return cmd.Wait()
}

// ── Config writers ─────────────────────────────────────────────────────────────

const ociConfigTpl = `[DEFAULT]
user={{.User}}
fingerprint={{.Fingerprint}}
tenancy={{.Tenancy}}
region={{.Region}}
key_file={{.KeyPath}}
`

func writeOCIConfig(cfg deployConfig) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".oci")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "config"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	keyPath := cfg.keyPath
	if strings.HasPrefix(keyPath, "~/") {
		keyPath = filepath.Join(home, keyPath[2:])
	}

	return template.Must(template.New("").Parse(ociConfigTpl)).Execute(f, map[string]string{
		"User":        cfg.user,
		"Fingerprint": cfg.fingerprint,
		"Tenancy":     cfg.tenancy,
		"Region":      cfg.region,
		"KeyPath":     keyPath,
	})
}

const tfVarsTpl = `compartment_id      = "{{.Tenancy}}"
availability_domain = "{{.AvailabilityDomain}}"
region              = "{{.Region}}"
shape               = "{{.Shape}}"
shape_ocpus         = {{.Ocpus}}
shape_memory_gbs    = {{.Memory}}
`

func writeTFVars(cfg deployConfig) error {
	path := filepath.Join(cfg.repoRoot, "terraform", "oci", "terraform.tfvars")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Derive availability domain from region: region + "-AD-1" with OCI format
	// User can adjust later; this is a best-guess default
	adName := ociDefaultAD(cfg.region)

	return template.Must(template.New("").Parse(tfVarsTpl)).Execute(f, map[string]string{
		"Tenancy":            cfg.tenancy,
		"AvailabilityDomain": adName,
		"Region":             cfg.region,
		"Shape":              cfg.shape,
		"Ocpus":              cfg.ocpus,
		"Memory":             cfg.memoryGBs,
	})
}

// ociDefaultAD returns a best-guess availability domain string.
// OCI ADs are region-specific; user may need to correct this.
func ociDefaultAD(region string) string {
	defaults := map[string]string{
		"us-ashburn-1":   "aBCD:US-ASHBURN-AD-1",
		"us-phoenix-1":   "aBCD:US-PHOENIX-AD-1",
		"us-chicago-1":   "aBCD:US-CHICAGO-1-AD-1",
		"sa-bogota-1":    "aBCD:SA-BOGOTA-1-AD-1",
		"sa-saopaulo-1":  "aBCD:SA-SAOPAULO-1-AD-1",
		"sa-santiago-1":  "aBCD:SA-SANTIAGO-1-AD-1",
		"eu-frankfurt-1": "aBCD:EU-FRANKFURT-1-AD-1",
	}
	if ad, ok := defaults[region]; ok {
		return ad
	}
	return "aBCD:" + strings.ToUpper(strings.ReplaceAll(region, "-", "_")) + "-AD-1"
}

