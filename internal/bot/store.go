package bot

import (
	"sync"
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	Role      string    // "user" or "assistant"
	Content   string    // The message content
	Timestamp time.Time // When the message was sent
}

// History represents a chat history between a user and the bot
type History struct {
	Messages []Message
	mu       sync.Mutex
}

// Store manages conversation histories for multiple users
type Store struct {
	history map[int64]*History // Map of chat ID to conversation
	mu      sync.RWMutex
}

// NewStore creates a new conversation store
func NewStore() *Store {
	return &Store{
		history: make(map[int64]*History),
	}
}

// AddUserMessage adds a user message to the conversation history
func (s *Store) AddUserMessage(chatID int64, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create conversation for this chat
	conv, exists := s.history[chatID]
	if !exists {
		conv = &History{
			Messages: []Message{},
		}
		s.history[chatID] = conv
	}

	// Add the message to the conversation
	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.Messages = append(
		conv.Messages, Message{
			Role:      "user",
			Content:   content,
			Timestamp: time.Now(),
		},
	)
}

// AddBotMessage adds a bot message to the conversation history
func (s *Store) AddBotMessage(chatID int64, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get conversation for this chat (should exist after user message)
	conv, exists := s.history[chatID]
	if !exists {
		// This shouldn't happen in normal flow, but handle it anyway
		conv = &History{
			Messages: []Message{},
		}
		s.history[chatID] = conv
	}

	// Add the message to the conversation
	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.Messages = append(
		conv.Messages, Message{
			Role:      "assistant",
			Content:   content,
			Timestamp: time.Now(),
		},
	)
}

// History returns the conversation history for a chat
// Limited to the most recent 'limit' messages
func (s *Store) History(chatID int64, limit int) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, exists := s.history[chatID]
	if !exists {
		return []Message{}
	}

	conv.mu.Lock()
	defer conv.mu.Unlock()

	messages := conv.Messages
	if limit > 0 && len(messages) > limit {
		// Return only the most recent 'limit' messages
		return messages[len(messages)-limit:]
	}
	return messages
}

// Clear clears the conversation history for a chat
func (s *Store) Clear(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.history, chatID)
}

// Length returns the number of messages in the conversation history for a chat
func (s *Store) Length(chatID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, exists := s.history[chatID]
	if !exists {
		return 0
	}

	conv.mu.Lock()
	defer conv.mu.Unlock()

	return len(conv.Messages)
}
