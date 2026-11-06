package blassistant

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	assistant "github.com/milbertk/shao/dataBasesLedger/assistant"
	"github.com/milbertk/shao/dataBasesLedger/products"
)

// ---- Public showcase assistant configuration (adjust here) ----
const (
	ShowcasePlatform           = "showcase"
	showcaseMaxMessageLength   = 500 // characters per question
	showcaseMaxOutputTokens    = 600
	showcaseVisitorDailyLimit  = 20  // questions per visitor per company per day
	showcaseIPDailyLimit       = 40  // questions per IP per company per day (in memory)
	showcaseCompanyDailyLimit  = 300 // questions per company per day, all visitors
	showcaseOffTopicDailyLimit = 3   // unrelated questions per visitor per day
	maxShowcaseCartLines       = 50
)

var (
	// ErrShowcaseNotFound: the company has no public catalog.
	ErrShowcaseNotFound = errors.New("showcase not found")
	// ErrShowcaseBusy: the company used its daily budget of public questions.
	ErrShowcaseBusy = errors.New("showcase assistant daily budget reached")
)

var visitorIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// ShowcaseCartLine is one line of the visitor's cart, sent by the page.
type ShowcaseCartLine struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// ShowcaseAskInput is the body of the public assistant endpoint.
type ShowcaseAskInput struct {
	ConversationID string             `json:"conversationId"`
	Message        string             `json:"message"`
	VisitorID      string             `json:"visitorId"`
	CompanyName    string             `json:"companyName"`
	Language       string             `json:"language"`
	Cart           []ShowcaseCartLine `json:"cart"`
}

// ShowcaseAskResult is returned to the showcase page.
type ShowcaseAskResult struct {
	ConversationID  string           `json:"conversationId"`
	NewConversation bool             `json:"newConversation"`
	Reply           string           `json:"reply"`
	Actions         []ShowcaseAction `json:"actions"`
	OnTopic         bool             `json:"onTopic"`
	RemainingToday  int              `json:"remainingToday"`
	InputTokens     int              `json:"inputTokens"`
	OutputTokens    int              `json:"outputTokens"`
}

// ShowcaseVisitorKey identifies the visitor for limits and conversations:
// the signed-in user id when there is one, otherwise "guest:<visitorId>"
// (a random id the page keeps), otherwise "guest-ip:<ip>".
func ShowcaseVisitorKey(loggedUserID, visitorID, clientIP string) string {
	if id := strings.TrimSpace(loggedUserID); id != "" {
		return truncateRunes(id, 128)
	}
	if id := strings.TrimSpace(visitorID); visitorIDPattern.MatchString(id) {
		return "guest:" + id
	}
	return "guest-ip:" + truncateRunes(strings.TrimSpace(clientIP), 100)
}

// BAskShowcaseAssistant answers one visitor question on a company's public
// showcase. companyID comes from the public URL and scopes everything.
func BAskShowcaseAssistant(companyID, visitorKey, clientIP string, loggedIn bool, input ShowcaseAskInput) (*ShowcaseAskResult, error) {
	companyID = strings.TrimSpace(companyID)
	visitorKey = strings.TrimSpace(visitorKey)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if visitorKey == "" {
		return nil, errors.New("visitor is required")
	}

	message := strings.TrimSpace(input.Message)
	if message == "" {
		return nil, errors.New("message is required")
	}
	if utf8.RuneCountInString(message) > showcaseMaxMessageLength {
		return nil, fmt.Errorf("invalid message: maximum %d characters", showcaseMaxMessageLength)
	}
	if len(input.Cart) > maxShowcaseCartLines {
		input.Cart = input.Cart[:maxShowcaseCartLines]
	}

	now := bltimezone.CompanyNow(companyID) // the company's clock (2026-11-04)

	// ---- Limits, cheapest first (no credits spent when blocked) ----
	if !publicIPCounter.allow(companyID+"|"+strings.TrimSpace(clientIP), showcaseIPDailyLimit, now) {
		return nil, ErrDailyLimitReached
	}

	// The company must have a public catalog (avoids spending credits on
	// random company ids).
	probe, err := blpublicshowcase.BSearchShowcase(companyID, products.PublicProductSearchFilters{Limit: 1})
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax") {
			return nil, ErrShowcaseNotFound
		}
		return nil, fmt.Errorf("failed to load showcase: %w", err)
	}
	if len(probe) == 0 {
		return nil, ErrShowcaseNotFound
	}

	// Plan feature + monthly cost cap of the company.
	if _, err := BCheckAssistantAccess(companyID); err != nil {
		return nil, err
	}

	visitorToday, offTopicToday, err := assistant.CountUserMessagesToday(companyID, visitorKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check assistant limits: %w", err)
	}
	if visitorToday >= showcaseVisitorDailyLimit {
		return nil, ErrDailyLimitReached
	}
	companyToday, err := assistant.CountCompanyPlatformMessagesToday(companyID, ShowcasePlatform)
	if err != nil {
		return nil, fmt.Errorf("failed to check assistant budget: %w", err)
	}
	if companyToday >= showcaseCompanyDailyLimit {
		return nil, ErrShowcaseBusy
	}

	// ---- Load or create the conversation (scoped to company + visitor) ----
	conversation := assistant.AIConversationS{
		ConversationID: strings.TrimSpace(input.ConversationID),
		CompanyID:      companyID,
		UserID:         visitorKey,
		Platform:       ShowcasePlatform,
		Area:           ShowcasePlatform,
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
		if conversation.Platform != ShowcasePlatform {
			return nil, ErrConversationNotFound
		}
		history, err = assistant.SelectRecentMessages(conversation.ConversationID, maxHistoryMessages)
		if err != nil {
			return nil, fmt.Errorf("failed to load conversation history: %w", err)
		}
	}

	// ---- Context: the visitor's cart, checked against the public catalog ----
	tools := &showcaseToolContext{CompanyID: companyID, Now: now}
	cartSummary := buildCartSummary(tools, input.Cart)

	system := buildShowcaseSystemPrompt(showcasePromptContext{
		CompanyName: truncateRunes(strings.TrimSpace(input.CompanyName), 120),
		Language:    truncateRunes(strings.TrimSpace(input.Language), 10),
		Now:         now,
		CartSummary: cartSummary,
		LoggedIn:    loggedIn,
		StrictMode:  offTopicToday >= showcaseOffTopicDailyLimit,
	})

	execute := func(name string, input json.RawMessage) (string, bool) {
		return runShowcaseTool(tools, name, input)
	}
	outcome, err := runToolConversation(system, buildClaudeMessages(history, message),
		showcaseToolDefinitions(), execute, showcaseMaxOutputTokens)
	if err != nil {
		return nil, err
	}

	onTopic, answer := parseTopicTag(outcome.Text)
	if len(outcome.ToolsUsed) > 0 {
		onTopic = true
	}

	// ---- Save the question and the final answer ----
	model := assistantModel()
	userMessage := assistant.AIMessageS{
		ConversationID: conversation.ConversationID,
		CompanyID:      companyID,
		UserID:         visitorKey,
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
		UserID:         visitorKey,
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
	_ = conversation.Touch()

	actions := tools.actions
	if actions == nil {
		actions = []ShowcaseAction{}
	}
	remaining := showcaseVisitorDailyLimit - visitorToday - 1
	if remaining < 0 {
		remaining = 0
	}

	return &ShowcaseAskResult{
		ConversationID:  conversation.ConversationID,
		NewConversation: newConversation,
		Reply:           answer,
		Actions:         actions,
		OnTopic:         onTopic,
		RemainingToday:  remaining,
		InputTokens:     outcome.InputTokens,
		OutputTokens:    outcome.OutputTokens,
	}, nil
}

// buildCartSummary describes the cart lines that belong to this company's
// public catalog (unknown ids are ignored).
func buildCartSummary(tools *showcaseToolContext, cart []ShowcaseCartLine) string {
	var sb strings.Builder
	for _, line := range cart {
		if line.Quantity <= 0 {
			continue
		}
		item, err := tools.findItem(line.ProductID)
		if err != nil {
			continue
		}
		price := "price on request"
		if item.HasPrice && item.BasePrice != nil {
			price = fmt.Sprintf("price %.2f", *item.BasePrice)
		}
		sb.WriteString(fmt.Sprintf("  - %d x %s (productId %s, %s)\n",
			line.Quantity, item.Name, item.ProductInternalID, price))
	}
	return sb.String()
}
