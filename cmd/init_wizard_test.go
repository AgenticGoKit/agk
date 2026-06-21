package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunInitWizard(t *testing.T) {
	// name pre-filled; choose workflow (by number), ollama (by name), with a description.
	in := strings.NewReader("2\nollama\nA test agent\n")
	var out bytes.Buffer

	res := runInitWizard(in, &out, "my-proj")

	if res.ProjectName != "my-proj" {
		t.Errorf("ProjectName = %q, want my-proj", res.ProjectName)
	}
	if res.Template != "workflow" {
		t.Errorf("Template = %q, want workflow", res.Template)
	}
	if res.LLMProvider != "ollama" {
		t.Errorf("LLMProvider = %q, want ollama", res.LLMProvider)
	}
	if res.Description != "A test agent" {
		t.Errorf("Description = %q, want 'A test agent'", res.Description)
	}
}

func TestRunInitWizardPromptsForName(t *testing.T) {
	// No name pre-filled: first answer is the project name; then accept defaults.
	in := strings.NewReader("cool-bot\n\n\n\n")
	var out bytes.Buffer

	res := runInitWizard(in, &out, "")

	if res.ProjectName != "cool-bot" {
		t.Errorf("ProjectName = %q, want cool-bot", res.ProjectName)
	}
	// Empty answers should fall back to the defaults.
	if res.Template != "quickstart" {
		t.Errorf("Template = %q, want quickstart (default)", res.Template)
	}
	if res.LLMProvider != "openai" {
		t.Errorf("LLMProvider = %q, want openai (default)", res.LLMProvider)
	}
}

func TestRunInitWizardEOFUsesDefaults(t *testing.T) {
	// Empty input (immediate EOF): name default + all defaults, no infinite loop.
	res := runInitWizard(strings.NewReader(""), new(bytes.Buffer), "")
	if res.ProjectName != "my-agent" || res.Template != "quickstart" || res.LLMProvider != "openai" {
		t.Errorf("unexpected defaults: %+v", res)
	}
}

func TestPrompterChooseInvalidThenValid(t *testing.T) {
	p := newPrompter(strings.NewReader("nope\nworkflow\n"), new(bytes.Buffer))
	got := p.choose("Template:", []string{"quickstart", "workflow"}, "quickstart")
	if got != "workflow" {
		t.Errorf("choose = %q, want workflow", got)
	}
}
