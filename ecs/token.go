package ecs

import (
	"sync"
	"time"
)

const (
	// TokenDefaultvalidDurationInSec is the default value
	TokenDefaultvalidDurationInSec = 3600
)

// Token is ECS mgmt token for authentication
type Token struct {
	value         string
	validDuration time.Duration
	createdAt     time.Time
	mutex         *sync.RWMutex
}

// ForceExpire sets current token expired
func (t *Token) ForceExpire() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.createdAt = time.Now().Add(-t.validDuration)
}

// Expired checks whether token expired
func (t *Token) Expired() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return time.Since(t.createdAt) > t.validDuration
}

// Refresh is to get a new Token
func (t *Token) Refresh(v string) string {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	prev := t.value
	t.value = v
	t.createdAt = time.Now()
	return prev
}

// Get is to set a new token and return previous one
func (t *Token) Get() (string, bool) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.value, t.Expired()
}

// NewToken is to generate a new token with default valid duration
func NewToken(v string) *Token {
	return NewTokenwithDuration(v, TokenDefaultvalidDurationInSec*time.Second)
}

// NewTokenwithDuration is to generate a new token with valid duration
func NewTokenwithDuration(v string, d time.Duration) *Token {
	return &Token{
		value:         v,
		validDuration: d,
		createdAt:     time.Now(),
		mutex:         &sync.RWMutex{},
	}
}
