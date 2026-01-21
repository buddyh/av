// Package version handles version detection and comparison
package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// GetInstalledClaude returns the installed Claude Code version
func GetInstalledClaude() string {
	// Method 1: Check symlink target
	home, _ := os.UserHomeDir()
	claudePath := filepath.Join(home, ".local", "bin", "claude")

	target, err := os.Readlink(claudePath)
	if err == nil {
		// Extract version from path like /Users/buddy/.local/share/claude/versions/2.1.14
		if idx := strings.LastIndex(target, "/"); idx != -1 {
			return target[idx+1:]
		}
	}

	// Method 2: Run claude --version
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		return ""
	}

	// Parse "2.1.14 (Claude Code)"
	version := strings.TrimSpace(string(out))
	if idx := strings.Index(version, " "); idx != -1 {
		return version[:idx]
	}
	return version
}

// GetInstalledCodex returns the installed Codex version
func GetInstalledCodex() string {
	out, err := exec.Command("codex", "--version").Output()
	if err != nil {
		return ""
	}

	// Parse "codex-cli 0.80.0"
	version := strings.TrimSpace(string(out))
	parts := strings.Fields(version)
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return version
}

// FetchLatestClaude gets the latest Claude Code version from GitHub
func FetchLatestClaude() string {
	// Try GitHub releases API first
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get("https://api.github.com/repos/anthropics/claude-code/releases/latest")
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		var release struct {
			TagName string `json:"tag_name"`
		}
		if json.NewDecoder(resp.Body).Decode(&release) == nil {
			// Remove 'v' prefix if present
			return strings.TrimPrefix(release.TagName, "v")
		}
	}

	// Fallback: fetch CHANGELOG.md and parse first version
	resp, err = client.Get("https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	// Match version pattern like "## 2.1.14" or "# 2.1.14"
	re := regexp.MustCompile(`##?\s*(\d+\.\d+\.\d+)`)
	matches := re.FindSubmatch(body)
	if len(matches) > 1 {
		return string(matches[1])
	}

	return ""
}

// FetchLatestCodex gets the latest Codex version from npm
func FetchLatestCodex() string {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get("https://registry.npmjs.org/@openai/codex")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var pkg struct {
		DistTags struct {
			Latest string `json:"latest"`
		} `json:"dist-tags"`
	}

	if json.NewDecoder(resp.Body).Decode(&pkg) != nil {
		return ""
	}

	return pkg.DistTags.Latest
}

// Compare returns -1 if a < b, 0 if a == b, 1 if a > b
func Compare(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return -1
	}
	if b == "" {
		return 1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		var aNum, bNum int
		fmt.Sscanf(aParts[i], "%d", &aNum)
		fmt.Sscanf(bParts[i], "%d", &bNum)
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}

	if len(aParts) < len(bParts) {
		return -1
	}
	if len(aParts) > len(bParts) {
		return 1
	}
	return 0
}
