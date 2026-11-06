package blassistant

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ErrAssistantUnavailable means the AI provider could not be reached or is not
// configured. The handler maps it to 503.
var ErrAssistantUnavailable = errors.New("assistant unavailable")

const (
	claudeAPIURL     = "https://api.anthropic.com/v1/messages"
	claudeAPIVersion = "2023-06-01"
)

var claudeHTTPClient = &http.Client{Timeout: 60 * time.Second}

// assistantModel returns the Claude model for both assistants: ANTHROPIC_MODEL
// from .env, or AssistantModel when it is empty. Change the model in .env when
// Anthropic retires the current one (no rebuild needed, only a restart).
func assistantModel() string {
	if model := strings.TrimSpace(os.Getenv("ANTHROPIC_MODEL")); model != "" {
		return model
	}
	return AssistantModel
}

// claudeMessage is one message of the conversation. Content is either a plain
// string, the raw content blocks returned by Claude (assistant tool turns), or
// a list of tool results (user turn after a tool call).
type claudeMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// claudeTool describes one tool Claude may ask the backend to run.
type claudeTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// claudeToolResult answers one tool_use block.
type claudeToolResult struct {
	Type      string `json:"type"` // always "tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []claudeMessage `json:"messages"`
	Tools     []claudeTool    `json:"tools,omitempty"`
}

type claudeContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type claudeResponse struct {
	Content    []json.RawMessage `json:"content"`
	StopReason string            `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// claudeToolUse is one tool call requested by Claude.
type claudeToolUse struct {
	ID    string
	Name  string
	Input json.RawMessage
}

type claudeReply struct {
	Text         string
	StopReason   string
	ToolUses     []claudeToolUse
	RawContent   []json.RawMessage // sent back as-is in the next request
	InputTokens  int
	OutputTokens int
}

// callClaude sends one Messages API request. The API key is read from the
// ANTHROPIC_API_KEY environment variable and never leaves the backend.
func callClaude(
	model string,
	maxTokens int,
	system string,
	messages []claudeMessage,
	tools []claudeTool,
) (*claudeReply, error) {
	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("%w: ANTHROPIC_API_KEY is not set", ErrAssistantUnavailable)
	}

	body, err := json.Marshal(claudeRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Tools:     tools,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build claude request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build claude request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := claudeHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAssistantUnavailable, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response: %v", ErrAssistantUnavailable, err)
	}

	var parsed claudeResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("%w: invalid response (status %d)", ErrAssistantUnavailable, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK || parsed.Error != nil {
		detail := ""
		if parsed.Error != nil {
			detail = parsed.Error.Type + ": " + parsed.Error.Message
		}
		return nil, fmt.Errorf("%w: status %d %s", ErrAssistantUnavailable, resp.StatusCode, detail)
	}

	reply := &claudeReply{
		StopReason:   parsed.StopReason,
		RawContent:   parsed.Content,
		InputTokens:  parsed.Usage.InputTokens,
		OutputTokens: parsed.Usage.OutputTokens,
	}

	var text strings.Builder
	for _, rawBlock := range parsed.Content {
		var block claudeContentBlock
		if err := json.Unmarshal(rawBlock, &block); err != nil {
			continue
		}
		switch block.Type {
		case "text":
			text.WriteString(block.Text)
		case "tool_use":
			reply.ToolUses = append(reply.ToolUses, claudeToolUse{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}
	reply.Text = text.String()

	return reply, nil
}
