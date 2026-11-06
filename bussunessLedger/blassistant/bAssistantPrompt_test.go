package blassistant

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var promptPathPattern = regexp.MustCompile(`\[(/[^\]\s]+)\]`)

// TestAssistantPromptPaths fails when the assistant knowledge links to a
// screen that is not in assistantScreenPaths (the list that must match p3
// assistantRoutes.js). Update the knowledge, the list and p3 together.
func TestAssistantPromptPaths(t *testing.T) {
	known := map[string]bool{}
	for _, path := range assistantScreenPaths {
		known[path] = true
	}

	text := assistantKnowledge + assistantRules
	found := 0
	for _, match := range promptPathPattern.FindAllStringSubmatch(text, -1) {
		found++
		if !known[match[1]] {
			t.Errorf("the assistant knowledge uses the unknown path [%s]; add it to assistantScreenPaths and p3 assistantRoutes.js, or fix it", match[1])
		}
	}
	if found == 0 {
		t.Fatal("no [/path] found in the assistant knowledge")
	}
}

// TestAssistantKnowledgeVersion makes sure the version is set and sent to Claude.
func TestAssistantKnowledgeVersion(t *testing.T) {
	if strings.TrimSpace(assistantKnowledgeVersion) == "" {
		t.Fatal("assistantKnowledgeVersion is empty")
	}
	prompt := buildSystemPrompt(promptContext{PlatformName: "AlephTelos", Area: "general", Now: time.Now()})
	if !strings.Contains(prompt, "Knowledge version: "+assistantKnowledgeVersion) {
		t.Fatal("the system prompt does not include the knowledge version")
	}
}

// TestAssistantModelFromEnv checks ANTHROPIC_MODEL wins over the default.
func TestAssistantModelFromEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "")
	if got := assistantModel(); got != AssistantModel {
		t.Fatalf("empty ANTHROPIC_MODEL: got %q, want %q", got, AssistantModel)
	}
	t.Setenv("ANTHROPIC_MODEL", "  claude-test-model ")
	if got := assistantModel(); got != "claude-test-model" {
		t.Fatalf("ANTHROPIC_MODEL set: got %q", got)
	}
}
