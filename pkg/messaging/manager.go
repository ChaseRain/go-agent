package messaging

import (
	"fmt"
	"sync"

	"go-agent/pkg/models"
)

// MessageManager handles message history and context
type MessageManager struct {
	messages     []models.Message
	mu           sync.RWMutex
	maxMessages  int
	maxTokens    int
	tokenCounter TokenCounter
}

// TokenCounter interface for counting tokens
type TokenCounter interface {
	Count(text string) int
}

// SimpleTokenCounter is a basic token counter (estimates ~4 chars per token)
type SimpleTokenCounter struct{}

func (s *SimpleTokenCounter) Count(text string) int {
	return len(text) / 4
}

// NewMessageManager creates a new MessageManager instance
func NewMessageManager(maxMessages int, maxTokens int) *MessageManager {
	return &MessageManager{
		messages:     make([]models.Message, 0),
		maxMessages:  maxMessages,
		maxTokens:    maxTokens,
		tokenCounter: &SimpleTokenCounter{},
	}
}

// AddMessage adds a message to the history
func (mm *MessageManager) AddMessage(message models.Message) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.messages = append(mm.messages, message)
	
	// Trim if exceeds max messages
	if mm.maxMessages > 0 && len(mm.messages) > mm.maxMessages {
		// Keep system messages and recent messages
		mm.trimMessages()
	}
	
	return nil
}

// GetHistory returns the full message history
func (mm *MessageManager) GetHistory() []models.Message {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	// Return a copy to avoid external modifications
	history := make([]models.Message, len(mm.messages))
	copy(history, mm.messages)
	return history
}

// ClearHistory clears the message history
func (mm *MessageManager) ClearHistory() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.messages = make([]models.Message, 0)
	return nil
}

// GetContextMessages returns messages for context with token limit
func (mm *MessageManager) GetContextMessages(maxTokens int) []models.Message {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	if maxTokens <= 0 {
		maxTokens = mm.maxTokens
	}
	
	// Always include system messages
	contextMessages := make([]models.Message, 0)
	systemMessages := make([]models.Message, 0)
	userMessages := make([]models.Message, 0)
	
	for _, msg := range mm.messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			userMessages = append(userMessages, msg)
		}
	}
	
	// Add system messages first
	contextMessages = append(contextMessages, systemMessages...)
	currentTokens := mm.countMessageTokens(systemMessages)
	
	// Add user messages from most recent, respecting token limit
	for i := len(userMessages) - 1; i >= 0; i-- {
		msgTokens := mm.tokenCounter.Count(userMessages[i].Content)
		if currentTokens+msgTokens > maxTokens {
			break
		}
		// Insert at the correct position after system messages
		insertPos := len(systemMessages)
		if insertPos < len(contextMessages) {
			contextMessages = append(contextMessages[:insertPos+1], 
				append([]models.Message{userMessages[i]}, contextMessages[insertPos+1:]...)...)
		} else {
			contextMessages = append(contextMessages, userMessages[i])
		}
		currentTokens += msgTokens
	}
	
	return contextMessages
}

// GetLastNMessages returns the last N messages
func (mm *MessageManager) GetLastNMessages(n int) []models.Message {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	if n <= 0 || n > len(mm.messages) {
		n = len(mm.messages)
	}
	
	result := make([]models.Message, n)
	copy(result, mm.messages[len(mm.messages)-n:])
	return result
}

// GetMessagesByRole returns messages filtered by role
func (mm *MessageManager) GetMessagesByRole(role string) []models.Message {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	filtered := make([]models.Message, 0)
	for _, msg := range mm.messages {
		if msg.Role == role {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// UpdateLastMessage updates the last message in history
func (mm *MessageManager) UpdateLastMessage(content string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if len(mm.messages) == 0 {
		return fmt.Errorf("no messages to update")
	}
	
	mm.messages[len(mm.messages)-1].Content = content
	return nil
}

// GetSummary generates a summary of the conversation
func (mm *MessageManager) GetSummary() string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	if len(mm.messages) == 0 {
		return "No messages in history"
	}
	
	summary := fmt.Sprintf("Conversation with %d messages:\n", len(mm.messages))
	
	// Count messages by role
	roleCounts := make(map[string]int)
	for _, msg := range mm.messages {
		roleCounts[msg.Role]++
	}
	
	for role, count := range roleCounts {
		summary += fmt.Sprintf("- %s: %d messages\n", role, count)
	}
	
	// Add first and last user messages for context
	userMessages := mm.GetMessagesByRole("user")
	if len(userMessages) > 0 {
		summary += fmt.Sprintf("\nFirst user message: %.100s...\n", userMessages[0].Content)
		if len(userMessages) > 1 {
			lastMsg := userMessages[len(userMessages)-1]
			summary += fmt.Sprintf("Last user message: %.100s...\n", lastMsg.Content)
		}
	}
	
	return summary
}

// SetTokenCounter sets a custom token counter
func (mm *MessageManager) SetTokenCounter(counter TokenCounter) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.tokenCounter = counter
}

// Private methods

func (mm *MessageManager) trimMessages() {
	// Keep all system messages
	systemMessages := make([]models.Message, 0)
	otherMessages := make([]models.Message, 0)
	
	for _, msg := range mm.messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}
	
	// Calculate how many non-system messages to keep
	keepCount := mm.maxMessages - len(systemMessages)
	if keepCount < 0 {
		keepCount = 0
	}
	
	// Keep the most recent messages
	if len(otherMessages) > keepCount {
		otherMessages = otherMessages[len(otherMessages)-keepCount:]
	}
	
	// Reconstruct messages array
	mm.messages = append(systemMessages, otherMessages...)
}

func (mm *MessageManager) countMessageTokens(messages []models.Message) int {
	total := 0
	for _, msg := range messages {
		total += mm.tokenCounter.Count(msg.Content)
		if msg.ReasoningContent != "" {
			total += mm.tokenCounter.Count(msg.ReasoningContent)
		}
	}
	return total
}