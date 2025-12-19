package bot

import (
	"sync"

	"github.com/j0lvera/banray/internal/agent"
)

// History represents a chat history between a user and the bot
type History struct {
	Messages []agent.Message
	mu       sync.Mutex
}

// Store manages conversation histories for multiple users
type Store struct {
	history map[int64]*History
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

	conv, exists := s.history[chatID]
	if !exists {
		conv = &History{
			Messages: []agent.Message{},
		}
		s.history[chatID] = conv
	}

	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.Messages = append(conv.Messages, agent.Message{
		Role:    agent.RoleUser,
		Content: content,
	})
}

// AddBotMessage adds a bot message to the conversation history
func (s *Store) AddBotMessage(chatID int64, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, exists := s.history[chatID]
	if !exists {
		conv = &History{
			Messages: []agent.Message{},
		}
		s.history[chatID] = conv
	}

	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.Messages = append(conv.Messages, agent.Message{
		Role:    agent.RoleAssistant,
		Content: content,
	})
}

// History returns the conversation history for a chat
// Limited to the most recent 'limit' messages
func (s *Store) History(chatID int64, limit int) []agent.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, exists := s.history[chatID]
	if !exists {
		return []agent.Message{}
	}

	conv.mu.Lock()
	defer conv.mu.Unlock()

	messages := conv.Messages
	if limit > 0 && len(messages) > limit {
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
