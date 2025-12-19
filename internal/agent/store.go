package agent

import "sync"

// Conversation represents a conversation history
type Conversation struct {
	Messages []Message
	mu       sync.Mutex
}

// Store manages conversation histories for multiple sessions
type Store struct {
	conversations map[int64]*Conversation
	mu            sync.RWMutex
}

// NewStore creates a new conversation store
func NewStore() *Store {
	return &Store{
		conversations: make(map[int64]*Conversation),
	}
}

// AddMessage adds a message to the conversation history
func (s *Store) AddMessage(conversationID int64, role Role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, exists := s.conversations[conversationID]
	if !exists {
		conv = &Conversation{
			Messages: []Message{},
		}
		s.conversations[conversationID] = conv
	}

	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.Messages = append(conv.Messages, Message{
		Role:    role,
		Content: content,
	})
}

// History returns the conversation history
// Limited to the most recent 'limit' messages
func (s *Store) History(conversationID int64, limit int) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, exists := s.conversations[conversationID]
	if !exists {
		return []Message{}
	}

	conv.mu.Lock()
	defer conv.mu.Unlock()

	messages := conv.Messages
	if limit > 0 && len(messages) > limit {
		return messages[len(messages)-limit:]
	}
	return messages
}

// Clear clears the conversation history
func (s *Store) Clear(conversationID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.conversations, conversationID)
}

// Length returns the number of messages in the conversation
func (s *Store) Length(conversationID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, exists := s.conversations[conversationID]
	if !exists {
		return 0
	}

	conv.mu.Lock()
	defer conv.mu.Unlock()

	return len(conv.Messages)
}
