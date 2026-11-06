package blassistant

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	assistant "github.com/milbertk/shao/dataBasesLedger/assistant"
)

// ---- Configuration (adjust here) ----
const (
	// Default model; ANTHROPIC_MODEL in .env wins (see assistantModel).
	AssistantModel     = "claude-haiku-5-5"
	maxMessageLength   = 1000 // characters per question
	maxOutputTokens    = 800  // keeps answers short and cheap
	maxHistoryMessages = 20   // previous messages sent as context
	maxToolRounds      = 5    // tool calls Claude may chain per question
	dailyMessageLimit  = 50   // questions per user per company per day
	dailyOffTopicLimit = 5    // unrelated questions per user per day
	defaultPlatform    = "servicehub"
	defaultArea        = "general"
)

// Platforms this agent serves: accepted names -> internal key.
var platformAliases = map[string]string{
	"servicehub":  "servicehub",
	"service hub": "servicehub",
	"shao":        "servicehub",
	"alephk":      "servicehub",
	"alephtelos":  "servicehub",
}

var platformNames = map[string]string{
	"servicehub": "AlephTelos", // display name (2026-10-25; before "Service Hub")
}

// Areas the frontend may send as context.
var allowedAreas = map[string]struct{}{
	"general":      {},
	"orders":       {},
	"weborders":    {},
	"bookings":     {},
	"appointments": {},
	"products":     {},
	"categories":   {},
	"customers":    {},
	"tracking":     {},
	"reports":      {},
	"users":        {},
	"settings":     {},
}

// Screen path (frontend) -> area, used when the frontend sends no area.
var pathAreas = map[string]string{
	"/shao/appointments":       "appointments",
	"/shao/selfbooking":        "appointments",
	"/shao/maintenancebooking": "bookings",
	"/shao/aprovalb":           "bookings",
	"/shao/orders":             "orders",
	"/shao/oporders":           "tracking",
	"/shao/approvala":          "weborders",
	"/shao/customersadd":       "customers",
	"/shao/products":           "products",
	"/shao/category":           "categories",
	"/shao/reports":            "reports",
	"/shao/page-configuration": "settings",
	"/shao/usersbcompany":      "users",
	// New menu addresses (2026-10-22).
	"/shao/book-for-customer": "appointments",
	"/shao/booking-schedules": "bookings",
	"/shao/booking-requests":  "bookings",
	"/shao/order-tracking":    "tracking",
	"/shao/order-stages":      "tracking",
	"/shao/web-orders":        "weborders",
	"/shao/customers":         "customers",
	"/shao/categories":        "categories",
	"/shao/team":              "users",
}

var (
	// ErrConversationNotFound: the id does not exist or belongs to another user.
	ErrConversationNotFound = errors.New("conversation not found")
	// ErrDailyLimitReached: the user used all the questions for today.
	ErrDailyLimitReached = errors.New("daily assistant limit reached")
)

// AskInput is the request body of the assistant endpoint. It accepts the same
// fields the frontend chat already sends. companyId in the body is ignored:
// the company always comes from the validated context.
type AskInput struct {
	ConversationID string `json:"conversationId"`
	Platform       string `json:"platform"`
	Area           string `json:"area"`
	Message        string `json:"message"`
	CompanyID      string `json:"companyId"`
	CompanyName    string `json:"companyName"`
	CurrentPath    string `json:"currentPath"`
	Language       string `json:"language"`
}

// AskResult is returned to the frontend. Reply and Answer hold the same text
// (Reply matches the usersAppAK agent response).
type AskResult struct {
	ConversationID  string   `json:"conversationId"`
	NewConversation bool     `json:"newConversation"`
	Reply           string   `json:"reply"`
	Answer          string   `json:"answer"`
	OnTopic         bool     `json:"onTopic"`
	ToolsUsed       []string `json:"toolsUsed"`
	RemainingToday  int      `json:"remainingToday"`
	Model           string   `json:"model"`
	InputTokens     int      `json:"inputTokens"`
	OutputTokens    int      `json:"outputTokens"`
}

// BAskAssistant validates the question, applies the daily limits, lets Claude
// read the company data through the read-only tools, and stores the question
// and the final answer.
//
//   - companyID, userID and role are authoritative (validated context).
//   - Empty conversationId -> a new conversation is created.
//   - Off-topic questions are counted; after dailyOffTopicLimit the assistant
//     only answers questions about the platform.
func BAskAssistant(companyID, userID, role string, input AskInput) (*AskResult, error) {
	companyID = strings.TrimSpace(companyID)
	userID = strings.TrimSpace(userID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if userID == "" {
		return nil, errors.New("userId is required")
	}

	// ---- Validate input ----
	message := strings.TrimSpace(input.Message)
	if message == "" {
		return nil, errors.New("message is required")
	}
	if utf8.RuneCountInString(message) > maxMessageLength {
		return nil, fmt.Errorf("invalid message: maximum %d characters", maxMessageLength)
	}

	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	if platform == "" {
		platform = defaultPlatform
	}
	platformKey, ok := platformAliases[platform]
	if !ok {
		return nil, fmt.Errorf("invalid platform: %s", truncateRunes(platform, 40))
	}
	platformName := platformNames[platformKey]

	currentPath := truncateRunes(strings.TrimSpace(input.CurrentPath), 200)
	area := resolveArea(input.Area, currentPath)

	ctx := promptContext{
		PlatformName: platformName,
		Area:         area,
		CompanyName:  truncateRunes(strings.TrimSpace(input.CompanyName), 150),
		CurrentPath:  currentPath,
		Language:     truncateRunes(strings.TrimSpace(input.Language), 10),
		Now:          bltimezone.CompanyNow(companyID), // the company's clock (2026-11-04)
	}

	// ---- Plan + monthly cost cap (checked BEFORE calling Claude) ----
	if _, err := BCheckAssistantAccess(companyID); err != nil {
		return nil, err
	}

	// ---- Daily limits (checked BEFORE calling Claude: no credits spent) ----
	totalToday, offTopicToday, err := assistant.CountUserMessagesToday(companyID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check assistant limits: %w", err)
	}
	if totalToday >= dailyMessageLimit {
		return nil, ErrDailyLimitReached
	}
	ctx.StrictMode = offTopicToday >= dailyOffTopicLimit

	// ---- Load or create the conversation ----
	conversation := assistant.AIConversationS{
		ConversationID: strings.TrimSpace(input.ConversationID),
		CompanyID:      companyID,
		UserID:         userID,
		Platform:       platformKey,
		Area:           area,
	}

	newConversation := false
	history := []assistant.AIMessageS{}
	if conversation.ConversationID == "" {
		title := truncateRunes(message, 80)
		conversation.Title = &title
		if err := conversation.Insert(); err != nil {
			return nil, fmt.Errorf("failed to create conversation: %w", err)
		}
		newConversation = true
	} else {
		if err := conversation.Select(); err != nil {
			if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "invalid input syntax for type uuid") {
				return nil, ErrConversationNotFound
			}
			return nil, fmt.Errorf("failed to load conversation: %w", err)
		}
		history, err = assistant.SelectRecentMessages(conversation.ConversationID, maxHistoryMessages)
		if err != nil {
			return nil, fmt.Errorf("failed to load conversation history: %w", err)
		}
		conversation.Area = area
	}

	// ---- Ask Claude (with the read-only tools) ----
	tools := &toolContext{CompanyID: companyID, UserID: userID, Role: role, Now: ctx.Now}
	outcome, err := runAssistantConversation(buildSystemPrompt(ctx), buildClaudeMessages(history, message), tools)
	if err != nil {
		return nil, err
	}

	onTopic, answer := parseTopicTag(outcome.Text)
	if len(outcome.ToolsUsed) > 0 {
		// Tools only read platform data: the question is about the platform.
		onTopic = true
	}

	// ---- Save the question and the final answer ----
	model := assistantModel()
	userMessage := assistant.AIMessageS{
		ConversationID: conversation.ConversationID,
		CompanyID:      companyID,
		UserID:         userID,
		Role:           "user",
		Content:        message,
		OnTopic:        &onTopic,
	}
	if err := userMessage.Insert(); err != nil {
		return nil, fmt.Errorf("failed to save question: %w", err)
	}

	assistantMessage := assistant.AIMessageS{
		ConversationID: conversation.ConversationID,
		CompanyID:      companyID,
		UserID:         userID,
		Role:           "assistant",
		Content:        answer,
		OnTopic:        &onTopic,
		Model:          &model,
		InputTokens:    outcome.InputTokens,
		OutputTokens:   outcome.OutputTokens,
	}
	if err := assistantMessage.Insert(); err != nil {
		return nil, fmt.Errorf("failed to save answer: %w", err)
	}

	// Updating the conversation timestamp must not fail the answer.
	_ = conversation.Touch()

	remaining := dailyMessageLimit - totalToday - 1
	if remaining < 0 {
		remaining = 0
	}

	return &AskResult{
		ConversationID:  conversation.ConversationID,
		NewConversation: newConversation,
		Reply:           answer,
		Answer:          answer,
		OnTopic:         onTopic,
		ToolsUsed:       outcome.ToolsUsed,
		RemainingToday:  remaining,
		Model:           model,
		InputTokens:     outcome.InputTokens,
		OutputTokens:    outcome.OutputTokens,
	}, nil
}

// conversationOutcome is the final text plus what it cost.
type conversationOutcome struct {
	Text         string
	ToolsUsed    []string
	InputTokens  int
	OutputTokens int
}

// claudeCaller is replaceable in tests.
var claudeCaller = callClaude

// toolRunner is replaceable in tests.
var toolRunner = runAssistantTool

// toolExecutor runs one tool by name and returns the tool_result text.
type toolExecutor func(name string, input json.RawMessage) (string, bool)

// runAssistantConversation runs the internal (company member) agent.
func runAssistantConversation(system string, messages []claudeMessage, tools *toolContext) (*conversationOutcome, error) {
	execute := func(name string, input json.RawMessage) (string, bool) {
		return toolRunner(tools, name, input)
	}
	return runToolConversation(system, messages, assistantToolDefinitions(), execute, maxOutputTokens)
}

// runToolConversation calls Claude, runs the tools it asks for, sends the
// results back, and repeats until Claude gives a final answer (or the round
// limit is reached). Shared by the internal and the showcase agents.
func runToolConversation(
	system string,
	messages []claudeMessage,
	definitions []claudeTool,
	execute toolExecutor,
	maxTokens int,
) (*conversationOutcome, error) {
	outcome := &conversationOutcome{ToolsUsed: []string{}}
	seenTools := map[string]bool{}

	for round := 0; ; round++ {
		reply, err := claudeCaller(assistantModel(), maxTokens, system, messages, definitions)
		if err != nil {
			return nil, err
		}
		outcome.InputTokens += reply.InputTokens
		outcome.OutputTokens += reply.OutputTokens

		if reply.StopReason != "tool_use" || len(reply.ToolUses) == 0 {
			outcome.Text = reply.Text
			return outcome, nil
		}

		if round >= maxToolRounds {
			// Too many lookups for one question: stop and keep any text so far.
			outcome.Text = reply.Text
			if strings.TrimSpace(stripTopicTags(outcome.Text)) == "" {
				outcome.Text = "I could not finish looking up that information. Please ask a more specific question (for example, one date or one order)."
			}
			return outcome, nil
		}

		// Send Claude's turn back exactly as received, then the tool results.
		messages = append(messages, claudeMessage{Role: "assistant", Content: reply.RawContent})

		results := make([]claudeToolResult, 0, len(reply.ToolUses))
		for _, use := range reply.ToolUses {
			content, isError := execute(use.Name, use.Input)
			results = append(results, claudeToolResult{
				Type:      "tool_result",
				ToolUseID: use.ID,
				Content:   content,
				IsError:   isError,
			})
			if !seenTools[use.Name] {
				seenTools[use.Name] = true
				outcome.ToolsUsed = append(outcome.ToolsUsed, use.Name)
			}
		}
		messages = append(messages, claudeMessage{Role: "user", Content: results})
	}
}

// buildClaudeMessages converts the stored history plus the new question into
// the API format. The API requires the list to start with a "user" message
// and to alternate roles, so any broken pair is skipped.
func buildClaudeMessages(history []assistant.AIMessageS, newMessage string) []claudeMessage {
	messages := make([]claudeMessage, 0, len(history)+1)
	for _, item := range history {
		role := strings.TrimSpace(item.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		if len(messages) == 0 && role != "user" {
			continue
		}
		if len(messages) > 0 && messages[len(messages)-1].Role == role {
			messages[len(messages)-1] = claudeMessage{Role: role, Content: item.Content}
			continue
		}
		messages = append(messages, claudeMessage{Role: role, Content: item.Content})
	}

	// The history must end with an assistant answer before the new question.
	if len(messages) > 0 && messages[len(messages)-1].Role == "user" {
		messages = messages[:len(messages)-1]
	}

	return append(messages, claudeMessage{Role: "user", Content: newMessage})
}

// resolveArea uses the area sent by the frontend, or derives it from the
// current screen path. Unknown values fall back to "general".
func resolveArea(area, currentPath string) string {
	area = strings.ToLower(strings.TrimSpace(area))
	if _, ok := allowedAreas[area]; ok {
		return area
	}
	if fromPath, ok := pathAreas[strings.ToLower(strings.TrimSpace(currentPath))]; ok {
		return fromPath
	}
	return defaultArea
}

// parseTopicTag reads the [ON_TOPIC]/[OFF_TOPIC] tag and returns the clean
// answer. A missing tag is treated as on-topic.
func parseTopicTag(text string) (bool, string) {
	onTopic := !strings.Contains(text, "[OFF_TOPIC]")

	answer := strings.TrimSpace(stripTopicTags(text))
	if answer == "" {
		answer = "Sorry, I could not generate an answer. Please try again."
	}
	return onTopic, answer
}

func stripTopicTags(text string) string {
	text = strings.ReplaceAll(text, "[OFF_TOPIC]", "")
	return strings.ReplaceAll(text, "[ON_TOPIC]", "")
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
