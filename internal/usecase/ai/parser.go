package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// Parser handles parsing and validation of Groq API responses
type Parser struct{}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{}
}

// ParseGroqJSONResponse parses the JSON response from Groq into AnalysisResult
func (p *Parser) ParseGroqJSONResponse(jsonString string) (*entities.AnalysisResult, error) {
	// Extract JSON from response (Groq might wrap it in markdown code blocks)
	jsonString = extractJSON(jsonString)

	var result entities.AnalysisResult
	if err := json.Unmarshal([]byte(jsonString), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Validate required fields
	if result.ExecutiveSummary == "" {
		return nil, fmt.Errorf("missing executive_summary in response")
	}

	return &result, nil
}

// ExtractActionItems converts analysis result action items to ActionItem entities
func (p *Parser) ExtractActionItems(ctx context.Context, roomID uuid.UUID, summaryID uuid.UUID, analysisResult *entities.AnalysisResult) ([]*entities.ActionItem, error) {
	if analysisResult == nil {
		return nil, fmt.Errorf("analysis result is nil")
	}

	actionItems := make([]*entities.ActionItem, 0)

	// Extract from action_items field
	for _, item := range analysisResult.ActionItems {
		actionItem := entities.NewActionItem(roomID, item.Title)
		actionItem.SummaryID = &summaryID
		actionItem.Description = item.Description
		actionItem.Type = item.Type
		actionItem.Priority = item.Priority
		actionItem.Status = entities.ActionItemStatusPending
		actionItem.TranscriptReference = item.TranscriptReference
		actionItem.TimestampInMeeting = item.TimestampInMeeting

		// TODO: Map assigned_to speaker label to actual user UUID
		// This requires participant tracking and speaker identification
		// For now, leave as nil

		actionItems = append(actionItems, actionItem)
	}

	// Extract from next_steps field
	for _, step := range analysisResult.NextSteps {
		actionItem := entities.NewActionItem(roomID, step.Description)
		actionItem.SummaryID = &summaryID
		actionItem.Type = entities.ActionItemTypeFollowUp
		actionItem.Priority = step.Priority
		actionItem.Status = entities.ActionItemStatusPending

		// TODO: Parse due_date_mentioned into actual date
		// For now, store in description
		if step.DueDateMentioned != "" {
			actionItem.Description = fmt.Sprintf("Due: %s\nOwner: %s", step.DueDateMentioned, step.Owner)
		}

		actionItems = append(actionItems, actionItem)
	}

	// Extract from decisions field as decision-type action items
	for _, decision := range analysisResult.Decisions {
		actionItem := entities.NewActionItem(roomID, decision.DecisionText)
		actionItem.SummaryID = &summaryID
		actionItem.Type = entities.ActionItemTypeDecision
		actionItem.Priority = entities.ActionItemPriorityHigh  // Decisions are high priority
		actionItem.Status = entities.ActionItemStatusCompleted // Decisions are already made
		actionItem.Description = fmt.Sprintf("Owner: %s\nImpact: %s", decision.Owner, decision.Impact)
		actionItem.TimestampInMeeting = decision.TimestampSeconds

		actionItems = append(actionItems, actionItem)
	}

	return actionItems, nil
}

// DetectLanguageMix analyzes transcript to detect mixed language usage
func (p *Parser) DetectLanguageMix(transcript string) (isMixed bool, primaryLang string, ratio string) {
	if transcript == "" {
		return false, "unknown", ""
	}

	words := strings.Fields(transcript)
	if len(words) == 0 {
		return false, "unknown", ""
	}

	vnWords := 0
	enWords := 0

	// Sample first 500 words for performance
	sampleSize := 500
	if len(words) > sampleSize {
		words = words[:sampleSize]
	}

	for _, word := range words {
		if isVietnameseWord(word) {
			vnWords++
		} else if isEnglishWord(word) {
			enWords++
		}
	}

	total := vnWords + enWords
	if total == 0 {
		return false, "unknown", ""
	}

	vnRatio := float64(vnWords) / float64(total)
	enRatio := float64(enWords) / float64(total)

	// Mixed if both languages > 20%
	if vnRatio > 0.2 && enRatio > 0.2 {
		primaryLang = "vi"
		if enRatio > vnRatio {
			primaryLang = "en"
		}
		return true, primaryLang, fmt.Sprintf("vi:%.0f%% en:%.0f%%", vnRatio*100, enRatio*100)
	}

	// Determine primary language
	if vnRatio > enRatio {
		return false, "vi", fmt.Sprintf("vi:%.0f%%", vnRatio*100)
	}

	return false, "en", fmt.Sprintf("en:%.0f%%", enRatio*100)
}

// ValidateTranscriptLength checks if transcript meets minimum requirements
func (p *Parser) ValidateTranscriptLength(transcript string, durationSeconds int) error {
	const (
		minChars    = 100
		minWords    = 20
		minDuration = 60 // 1 minute
	)

	if len(transcript) < minChars {
		return fmt.Errorf("transcript too short: %d characters (minimum: %d)", len(transcript), minChars)
	}

	words := strings.Fields(transcript)
	if len(words) < minWords {
		return fmt.Errorf("transcript too short: %d words (minimum: %d)", len(words), minWords)
	}

	if durationSeconds < minDuration {
		return fmt.Errorf("meeting too short: %d seconds (minimum: %d)", durationSeconds, minDuration)
	}

	return nil
}

// CalculateSentimentPerSpeaker extracts per-speaker sentiment from analysis result
func (p *Parser) CalculateSentimentPerSpeaker(analysisResult *entities.AnalysisResult) map[string]float64 {
	if analysisResult == nil {
		return make(map[string]float64)
	}

	return analysisResult.SpeakerSentiment
}

// extractJSON extracts JSON content from markdown code blocks or plain text
func extractJSON(content string) string {
	content = strings.TrimSpace(content)

	// Check if wrapped in markdown code block
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
	}

	return strings.TrimSpace(content)
}

// isVietnameseWord checks if a word contains Vietnamese diacritics
func isVietnameseWord(word string) bool {
	vnChars := "àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩịòóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵđĐ"
	word = strings.ToLower(word)

	for _, char := range word {
		if strings.ContainsRune(vnChars, char) {
			return true
		}
	}

	// Common Vietnamese words without diacritics
	commonVN := []string{
		"toi", "ban", "cua", "cho", "nay", "khi", "co", "khong", "voi", "la",
		"nhu", "hay", "can", "phai", "rat", "roi", "thi", "se", "duoc", "tu",
	}

	for _, vn := range commonVN {
		if word == vn {
			return true
		}
	}

	return false
}

// isEnglishWord checks if a word looks like English (basic heuristic)
func isEnglishWord(word string) bool {
	// Remove punctuation
	word = strings.Trim(word, ".,!?;:'\"")
	word = strings.ToLower(word)

	// Check if contains only ASCII letters
	for _, char := range word {
		if (char < 'a' || char > 'z') && char != '\'' && char != '-' {
			return false
		}
	}

	// Common English words
	commonEN := []string{
		"the", "is", "are", "was", "were", "be", "have", "has", "had",
		"do", "does", "did", "will", "would", "can", "could", "should",
		"may", "might", "must", "shall", "this", "that", "these", "those",
		"and", "or", "but", "if", "then", "else", "when", "where", "what",
		"who", "which", "how", "why", "yes", "no", "not", "only", "just",
		"about", "from", "with", "into", "through", "during", "before", "after",
	}

	for _, en := range commonEN {
		if word == en {
			return true
		}
	}

	// If word is 3+ chars and all ASCII letters, likely English
	return len(word) >= 3
}

// ValidateAnalysisResult validates that all required fields are present
func (p *Parser) ValidateAnalysisResult(result *entities.AnalysisResult) error {
	if result == nil {
		return fmt.Errorf("analysis result is nil")
	}

	if result.ExecutiveSummary == "" {
		return fmt.Errorf("missing executive_summary")
	}

	// KeyPoints, Decisions, etc. can be empty for short meetings
	// Just ensure they're initialized
	if result.KeyPoints == nil {
		result.KeyPoints = make([]entities.KeyPoint, 0)
	}
	if result.Decisions == nil {
		result.Decisions = make([]entities.Decision, 0)
	}
	if result.Topics == nil {
		result.Topics = make([]string, 0)
	}
	if result.OpenQuestions == nil {
		result.OpenQuestions = make([]string, 0)
	}
	if result.NextSteps == nil {
		result.NextSteps = make([]entities.NextStep, 0)
	}
	if result.ActionItems == nil {
		result.ActionItems = make([]entities.ActionItemExtracted, 0)
	}
	if result.SpeakerSentiment == nil {
		result.SpeakerSentiment = make(map[string]float64)
	}
	if result.ParticipantBalance == nil {
		result.ParticipantBalance = make(map[string]entities.ParticipantMetrics)
	}

	return nil
}
