package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// InitWizardResult holds the answers collected by the interactive init wizard.
type InitWizardResult struct {
	ProjectName string
	Template    string
	LLMProvider string
	Description string
}

// prompter reads typed answers from a reader and writes prompts to a writer.
// It is decoupled from os.Stdin/os.Stdout so the wizard can be unit-tested.
type prompter struct {
	reader *bufio.Reader
	out    io.Writer
}

func newPrompter(in io.Reader, out io.Writer) *prompter {
	return &prompter{reader: bufio.NewReader(in), out: out}
}

// ask prompts for a free-text value, returning def when the user enters nothing
// (or input ends).
func (p *prompter) ask(label, def string) string {
	if def != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	line, _ := p.reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// choose prompts the user to pick one of options, by number or name. def must be a
// member of options and is returned on empty input or EOF, so this never loops forever.
func (p *prompter) choose(label string, options []string, def string) string {
	fmt.Fprintln(p.out, label)
	for i, o := range options {
		marker := " "
		if o == def {
			marker = "*"
		}
		fmt.Fprintf(p.out, "  %s %d) %s\n", marker, i+1, o)
	}

	for {
		ans := p.ask("Choose (number or name)", def)
		if n, err := strconv.Atoi(ans); err == nil && n >= 1 && n <= len(options) {
			return options[n-1]
		}
		for _, o := range options {
			if strings.EqualFold(o, ans) {
				return o
			}
		}
		fmt.Fprintf(p.out, "  invalid choice: %q\n", ans)
		// On EOF, ask() returns def (a valid option), so the loop terminates.
	}
}

// runInitWizard walks the user through project setup, pre-filling the project name
// when one was already provided on the command line.
func runInitWizard(in io.Reader, out io.Writer, name string) InitWizardResult {
	fmt.Fprintln(out, "🧙 AGK interactive project setup")
	fmt.Fprintln(out)

	p := newPrompter(in, out)

	res := InitWizardResult{ProjectName: name}
	if res.ProjectName == "" {
		res.ProjectName = p.ask("Project name", "my-agent")
	}
	res.Template = p.choose("Template:", []string{"quickstart", "workflow"}, "quickstart")
	res.LLMProvider = p.choose("LLM provider:", []string{"openai", "anthropic", "ollama"}, "openai")
	res.Description = p.ask("Description (optional)", "")

	fmt.Fprintln(out)
	return res
}
