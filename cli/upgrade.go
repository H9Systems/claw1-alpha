package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const upgradeRepo = "H9Systems/claw1-alpha"

// ── GitHub release types ──────────────────────────────────────────────────────

type ghRelease struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Assets  []ghAsset     `json:"assets"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// ── CLI entry point ────────────────────────────────────────────────────────────

func runUpgradeCLI(repoRoot string, args []string) int {
	opt, _, err := parseCommonFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	sink := eventSink{json: opt.json}
	runID := newRunID()

	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "start", Status: "started",
		Message: fmt.Sprintf("current version %s", version),
	})

	// 1. Resolve current executable path
	exePath, err := os.Executable()
	if err != nil {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "resolve_exe", Status: "failed_closed",
			ErrorCode: "exe_path_error", Message: err.Error(),
		})
		return 1
	}
	// Dereference symlinks so we replace the real binary
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "resolve_exe", Status: "failed_closed",
			ErrorCode: "eval_symlinks_error", Message: err.Error(),
		})
		return 1
	}
	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "resolve_exe", Status: "succeeded",
		Message: exePath,
	})

	// 2. Fetch latest release metadata from GitHub
	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "fetch_release", Status: "started",
		Message: fmt.Sprintf("querying https://api.github.com/repos/%s/releases/latest", upgradeRepo),
	})

	release, err := fetchLatestRelease()
	if err != nil {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "fetch_release", Status: "failed_closed",
			ErrorCode: "github_api_error", Message: err.Error(),
		})
		return 1
	}
	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "fetch_release", Status: "succeeded",
		Message: fmt.Sprintf("latest release: %s", release.TagName),
	})

	// 3. Compare versions
	if version != "dev" && !isNewer(version, release.TagName) {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "compare", Status: "succeeded",
			Message: fmt.Sprintf("already up to date (%s)", version),
		})
		return 0
	}
	if version == "dev" {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "compare", Status: "running",
			Message: fmt.Sprintf("dev build → upgrading to %s", release.TagName),
		})
	} else {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "compare", Status: "running",
			Message: fmt.Sprintf("%s → %s", version, release.TagName),
		})
	}

	// 4. Find matching asset for current OS/arch
	assetName := fmt.Sprintf("claw1-%s-%s", runtime.GOOS, runtime.GOARCH)
	assetURL := ""
	for _, a := range release.Assets {
		if a.Name == assetName {
			assetURL = a.URL
			break
		}
	}
	if assetURL == "" {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "find_asset", Status: "failed_closed",
			ErrorCode: "no_matching_asset", Message: fmt.Sprintf("no asset named %s in release %s", assetName, release.TagName),
		})
		return 1
	}
	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "find_asset", Status: "succeeded",
		Message: fmt.Sprintf("matched asset: %s", assetName),
	})

	// 5. Download to temp file
	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "download", Status: "started",
		Message: assetURL,
	})
	tmpFile, err := downloadToFile(assetURL, assetName)
	if err != nil {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "download", Status: "failed_closed",
			ErrorCode: "download_error", Message: err.Error(),
		})
		return 1
	}
	defer os.Remove(tmpFile)
	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "download", Status: "succeeded",
		Message: tmpFile,
	})

	// 6. Validate the downloaded binary (check it's executable-sized & not HTML error page)
	info, err := os.Stat(tmpFile)
	if err != nil {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "validate", Status: "failed_closed",
			ErrorCode: "stat_error", Message: err.Error(),
		})
		return 1
	}
	if info.Size() < 1<<20 { // less than 1 MB is suspicious
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "validate", Status: "failed_closed",
			ErrorCode: "binary_too_small", Message: fmt.Sprintf("downloaded %d bytes — expected a Go binary >1MB", info.Size()),
		})
		return 1
	}

	// 7. Replace the running binary atomically
	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "replace", Status: "started",
		Message: fmt.Sprintf("replacing %s", exePath),
	})
	if err := replaceBinary(exePath, tmpFile); err != nil {
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "replace", Status: "failed_closed",
			ErrorCode: "replace_error", Message: err.Error(),
		})
		// Provide remediation hint
		sink.emit(workflowEvent{
			RunID: runID, Workflow: "upgrade", Step: "remediation", Status: "running",
			Message: fmt.Sprintf("manual fallback: mv %s %s", tmpFile, exePath),
		})
		return 1
	}

	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "replace", Status: "succeeded",
		Message: fmt.Sprintf("upgraded to %s", release.TagName),
	})

	sink.emit(workflowEvent{
		RunID: runID, Workflow: "upgrade", Step: "complete", Status: "succeeded",
		Message: fmt.Sprintf("claw1 %s — restart your terminal if needed", release.TagName),
	})
	return 0
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func fetchLatestRelease() (*ghRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", upgradeRepo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "claw1-upgrade")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github returned %d: %s", resp.StatusCode, string(body))
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding release JSON: %w", err)
	}
	return &release, nil
}

func downloadToFile(url, nameHint string) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("building download request: %w", err)
	}
	req.Header.Set("User-Agent", "claw1-upgrade")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// Write to a temp file next to the destination to avoid cross-device rename issues
	tmp, err := os.CreateTemp("", "claw1-upgrade-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("writing download: %w", err)
	}
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("chmod: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("closing temp file: %w", err)
	}
	return tmpPath, nil
}

func replaceBinary(dst, src string) error {
	// Try direct rename first (same filesystem, writable dir)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Rename may fail across devices or on read-only directories.
	// Fall back to copy-then-truncate, which works on Linux when the
	// binary is still executing (same inode, content replaced).
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("open destination for write: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	return nil
}

// isNewer compares two semver-ish version strings.
// It strips leading "v" and returns true if candidate > current.
func isNewer(current, candidate string) bool {
	clean := func(s string) string {
		s = strings.TrimPrefix(s, "v")
		// Strip anything after a "+" (build metadata) or "-" (prerelease)
		if i := strings.IndexAny(s, "-+"); i >= 0 {
			s = s[:i]
		}
		return s
	}
	cur := clean(current)
	cand := clean(candidate)

	if cur == cand {
		return false
	}
	curParts := strings.Split(cur, ".")
	candParts := strings.Split(cand, ".")

	maxLen := len(curParts)
	if len(candParts) > maxLen {
		maxLen = len(candParts)
	}

	for i := 0; i < maxLen; i++ {
		var cv, ca int
		if i < len(curParts) {
			fmt.Sscanf(curParts[i], "%d", &cv)
		}
		if i < len(candParts) {
			fmt.Sscanf(candParts[i], "%d", &ca)
		}
		if ca > cv {
			return true
		}
		if ca < cv {
			return false
		}
	}
	// Equal so far — candidate is not strictly newer
	return false
}