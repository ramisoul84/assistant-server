package state

import (
	"sync"
	"time"
)

// Stage represents where in the conversation the user currently is.
type Stage string

const (
	StageIdle               Stage = "idle"
	StageAwaitingCompletion Stage = "awaiting_completion" // user gave incomplete add intent
	StageGymSessionOpen     Stage = "gym_session_open"    // user is logging a gym session
)

// IncompleteIntent tells us what the user was trying to add when they gave incomplete info.
type IncompleteIntent string

const (
	IncompleteAppointment IncompleteIntent = "appointment"
	IncompleteExpense     IncompleteIntent = "expense"
)

// GymExercise holds one parsed exercise during an open session.
type GymExercise struct {
	Name     string
	Sets     int
	Reps     int
	WeightKg float64
	Notes    string
}

// Conversation holds the full state for one user's conversation.
type Conversation struct {
	Stage          Stage
	IncompleteFor  IncompleteIntent // set when Stage == StageAwaitingCompletion
	GymSessionID   int64            // DB id of the open gym session
	GymExercises   []GymExercise    // accumulated exercises in current session
	GymStartedAt   time.Time
	UpdatedAt      time.Time
}

// Store is a thread-safe in-memory conversation state store.
// Keyed by Telegram user ID.
type Store struct {
	mu    sync.RWMutex
	convs map[int64]*Conversation
}

func NewStore() *Store {
	return &Store{convs: make(map[int64]*Conversation)}
}

func (s *Store) Get(telegramUserID int64) *Conversation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if c, ok := s.convs[telegramUserID]; ok {
		return c
	}
	return &Conversation{Stage: StageIdle}
}

func (s *Store) Set(telegramUserID int64, c *Conversation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.UpdatedAt = time.Now()
	s.convs[telegramUserID] = c
}

func (s *Store) Reset(telegramUserID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.convs, telegramUserID)
}
