package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agenticgokit/agk/pkg/registry"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// checkLevel is the severity of a single diagnostic result.
type checkLevel int

const (
	checkOK checkLevel = iota
	checkWarn
	checkError
)

// checkResult is the outcome of one diagnostic.
type checkResult struct {
	name   string
	detail string
	level  checkLevel
	hint   string
}

var doctorProvider string

// doctorCmd runs environment diagnostics so common first-run failures (missing Go,
// unset API keys, Ollama down, registry unreachable) surface before an agent runs.
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose your environment for building and running agents",
	Long: `Run a series of environment checks and report anything that would block
building or running AgenticGoKit agents.

Checks:
  • Go toolchain (present and >= go1.24)
  • LLM credentials (OPENAI_API_KEY / ANTHROPIC_API_KEY / AZURE_OPENAI_API_KEY)
  • Ollama reachability (for local models)
  • Template registry reachability
  • Current project workspace (.agk/)

Exits non-zero if any check is an error, so it can be used in CI preflight.

Examples:
  agk doctor
  agk doctor --provider openai`,
	Args: cobra.NoArgs,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().StringVar(&doctorProvider, "provider", "",
		"Require credentials for a specific provider (openai|anthropic|azure|ollama)")
}

func runDoctor(_ *cobra.Command, _ []string) error {
	color.Cyan("\n🩺 AGK Doctor — environment diagnostics\n")

	checks := []checkResult{
		checkGoToolchain(),
		checkLLMProviders(doctorProvider),
		checkOllama(),
		checkRegistry(),
		checkWorkspace(),
	}

	var warns, errs int
	for _, c := range checks {
		printCheck(c)
		switch c.level {
		case checkWarn:
			warns++
		case checkError:
			errs++
		case checkOK:
		}
	}

	fmt.Println()
	switch {
	case errs > 0:
		color.Red("✗ %d error(s), %d warning(s) — fix errors before running agents", errs, warns)
		os.Exit(1)
	case warns > 0:
		color.Yellow("⚠ %d warning(s) — you can still build and run", warns)
	default:
		color.Green("✓ All checks passed — you're ready to build agents")
	}
	return nil
}

func printCheck(c checkResult) {
	var icon string
	var colorize func(string, ...interface{}) string
	switch c.level {
	case checkOK:
		icon, colorize = "✓", color.GreenString
	case checkWarn:
		icon, colorize = "⚠", color.YellowString
	case checkError:
		icon, colorize = "✗", color.RedString
	}
	fmt.Printf("  %s %-24s %s\n", colorize(icon), c.name, c.detail)
	if c.hint != "" && c.level != checkOK {
		fmt.Printf("    %s %s\n", color.HiBlackString("↳"), color.HiBlackString(c.hint))
	}
}

func checkGoToolchain() checkResult {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return checkResult{"Go toolchain", "`go` not found on PATH", checkError,
			"Install Go 1.24+ from https://go.dev/dl/"}
	}
	// Output looks like: "go version go1.26.4 linux/amd64"
	verStr := "unknown"
	if fields := strings.Fields(strings.TrimSpace(string(out))); len(fields) >= 3 {
		verStr = fields[2]
	}
	major, minor := parseGoVersion(verStr)
	if major < 1 || (major == 1 && minor < 24) {
		return checkResult{"Go toolchain", verStr + " (need >= go1.24)", checkError,
			"Upgrade Go to 1.24 or newer"}
	}
	return checkResult{"Go toolchain", verStr, checkOK, ""}
}

func parseGoVersion(v string) (int, int) {
	v = strings.TrimPrefix(v, "go")
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return 0, 0
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	return major, minor
}

func checkLLMProviders(focus string) checkResult {
	keys := map[string]string{
		"openai":    "OPENAI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"azure":     "AZURE_OPENAI_API_KEY",
	}

	if focus != "" && focus != "ollama" {
		env, ok := keys[focus]
		if !ok {
			return checkResult{"LLM provider", "unknown provider: " + focus, checkWarn,
				"valid: openai, anthropic, azure, ollama"}
		}
		if os.Getenv(env) == "" {
			return checkResult{"LLM provider (" + focus + ")", env + " not set", checkError,
				"export " + env + "=..."}
		}
		return checkResult{"LLM provider (" + focus + ")", env + " is set", checkOK, ""}
	}

	var configured []string
	for name, env := range keys {
		if os.Getenv(env) != "" {
			configured = append(configured, name)
		}
	}
	sort.Strings(configured)

	if len(configured) == 0 {
		return checkResult{"LLM credentials", "no cloud provider keys set", checkWarn,
			"export OPENAI_API_KEY / ANTHROPIC_API_KEY, or use Ollama locally"}
	}
	return checkResult{"LLM credentials", "configured: " + strings.Join(configured, ", "), checkOK, ""}
}

func checkOllama() checkResult {
	host := ollamaHost()
	client := &http.Client{Timeout: 3 * time.Second}

	resp, err := client.Get(host + "/api/tags")
	if err != nil {
		return checkResult{"Ollama", "not reachable at " + host, checkWarn,
			"Start Ollama with `ollama serve` if you want local models"}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return checkResult{"Ollama", fmt.Sprintf("HTTP %d from %s", resp.StatusCode, host), checkWarn, ""}
	}

	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)

	return checkResult{"Ollama", fmt.Sprintf("running at %s (%d model(s))", host, len(body.Models)), checkOK, ""}
}

func ollamaHost() string {
	h := os.Getenv("OLLAMA_HOST")
	if h == "" {
		return "http://localhost:11434"
	}
	if !strings.HasPrefix(h, "http") {
		h = "http://" + h
	}
	return strings.TrimRight(h, "/")
}

func checkRegistry() checkResult {
	index, err := registry.FetchIndex(registry.DefaultRegistryURL)
	if err != nil {
		return checkResult{"Template registry", "unreachable", checkWarn,
			"Check your network; built-in templates still work offline"}
	}
	return checkResult{"Template registry",
		fmt.Sprintf("reachable (%d template(s))", len(index.Templates)), checkOK, ""}
}

func checkWorkspace() checkResult {
	if _, err := os.Stat("go.mod"); err != nil {
		return checkResult{"Project", "no go.mod in current directory", checkWarn,
			"Run `agk init <name>` or cd into a project directory"}
	}
	runs := countSubdirs(runsDirName)
	return checkResult{"Project",
		fmt.Sprintf("go.mod found; %d trace run(s) in %s", runs, runsDirName), checkOK, ""}
}

func countSubdirs(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			n++
		}
	}
	return n
}
