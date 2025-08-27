package messaging

import (
	"fmt"
	"testing"

	"go-agent/pkg/models"
)

func TestMessageManager_AddAndGetHistory(t *testing.T) {
	mm := NewMessageManager(10, 1000)
	
	// Add messages
	messages := []models.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}
	
	for _, msg := range messages {
		if err := mm.AddMessage(msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}
	
	// Get history
	history := mm.GetHistory()
	
	if len(history) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(history))
	}
	
	// Verify message content
	for i, msg := range history {
		if msg.Role != messages[i].Role {
			t.Errorf("Message %d: expected role %s, got %s", i, messages[i].Role, msg.Role)
		}
		if msg.Content != messages[i].Content {
			t.Errorf("Message %d: expected content %s, got %s", i, messages[i].Content, msg.Content)
		}
	}
}

func TestMessageManager_MaxMessages(t *testing.T) {
	mm := NewMessageManager(5, 1000) // Max 5 messages
	
	// Add more than max messages
	for i := 0; i < 10; i++ {
		msg := models.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d", i),
		}
		mm.AddMessage(msg)
	}
	
	history := mm.GetHistory()
	
	// Should not exceed max messages
	if len(history) > 5 {
		t.Errorf("Expected max 5 messages, got %d", len(history))
	}
}

func TestMessageManager_GetContextMessages(t *testing.T) {
	mm := NewMessageManager(100, 1000)
	
	// Add system message
	mm.AddMessage(models.Message{
		Role:    "system",
		Content: "System prompt",
	})
	
	// Add many user messages
	for i := 0; i < 20; i++ {
		mm.AddMessage(models.Message{
			Role:    "user",
			Content: fmt.Sprintf("This is a relatively long message number %d with some content", i),
		})
	}
	
	// Get context with token limit
	contextMessages := mm.GetContextMessages(200) // About 50 tokens
	
	// Should include system message
	if contextMessages[0].Role != "system" {
		t.Error("First message should be system message")
	}
	
	// Should have limited messages due to token constraint
	if len(contextMessages) > 10 {
		t.Errorf("Expected fewer messages due to token limit, got %d", len(contextMessages))
	}
}

func TestMessageManager_GetLastNMessages(t *testing.T) {
	mm := NewMessageManager(100, 1000)
	
	// Add 10 messages
	for i := 0; i < 10; i++ {
		mm.AddMessage(models.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d", i),
		})
	}
	
	// Get last 3 messages
	last3 := mm.GetLastNMessages(3)
	
	if len(last3) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(last3))
	}
	
	// Verify it's the last 3
	if last3[2].Content != "Message 9" {
		t.Errorf("Expected last message to be 'Message 9', got %s", last3[2].Content)
	}
}

func TestMessageManager_GetMessagesByRole(t *testing.T) {
	mm := NewMessageManager(100, 1000)
	
	// Add mixed messages
	mm.AddMessage(models.Message{Role: "system", Content: "System"})
	mm.AddMessage(models.Message{Role: "user", Content: "User 1"})
	mm.AddMessage(models.Message{Role: "assistant", Content: "Assistant 1"})
	mm.AddMessage(models.Message{Role: "user", Content: "User 2"})
	mm.AddMessage(models.Message{Role: "assistant", Content: "Assistant 2"})
	
	// Get user messages
	userMessages := mm.GetMessagesByRole("user")
	
	if len(userMessages) != 2 {
		t.Errorf("Expected 2 user messages, got %d", len(userMessages))
	}
	
	// Get assistant messages
	assistantMessages := mm.GetMessagesByRole("assistant")
	
	if len(assistantMessages) != 2 {
		t.Errorf("Expected 2 assistant messages, got %d", len(assistantMessages))
	}
}

func TestMessageManager_ClearHistory(t *testing.T) {
	mm := NewMessageManager(100, 1000)
	
	// Add messages
	mm.AddMessage(models.Message{Role: "user", Content: "Test"})
	mm.AddMessage(models.Message{Role: "assistant", Content: "Response"})
	
	// Clear history
	if err := mm.ClearHistory(); err != nil {
		t.Fatalf("Failed to clear history: %v", err)
	}
	
	// Verify empty
	history := mm.GetHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d messages", len(history))
	}
}

func BenchmarkMessageManager_AddMessage(b *testing.B) {
	mm := NewMessageManager(1000, 10000)
	msg := models.Message{
		Role:    "user",
		Content: "This is a test message with some content",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.AddMessage(msg)
	}
}

func BenchmarkMessageManager_GetContextMessages(b *testing.B) {
	mm := NewMessageManager(100, 4000)
	
	// Pre-populate with messages
	for i := 0; i < 50; i++ {
		mm.AddMessage(models.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d with some content", i),
		})
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.GetContextMessages(2000)
	}
}