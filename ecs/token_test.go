package ecs

import (
	"testing"
	"time"
)

// TestToken ...
func TestToken(t *testing.T) {
	validP := time.Millisecond * 10
	dt1 := "dummyToken_1"
	dt2 := "dummyToken_2"

	start := time.Now()
	// New
	token := NewTokenwithDuration(dt1, validP)
	AssertNotEqualFatal(t, nil, token, "")
	// Expired
	AssertEqual(t, time.Since(start) > validP, token.Expired(), "")
	// Get
	rt, exp := token.Get()
	AssertEqual(t, dt1, rt, "")
	AssertEqual(t, time.Since(start) > validP, exp, "")
	// Expired
	time.Sleep(validP)
	AssertEqual(t, true, token.Expired(), "")
	// Refresh
	start = time.Now()
	prev := token.Refresh(dt2)
	AssertEqual(t, dt1, prev, "")
	rt, exp = token.Get()
	AssertEqual(t, dt2, rt, "")
	AssertEqual(t, time.Since(start) > validP, exp, "")
	// ForceExpire
	token.ForceExpire()
	AssertEqual(t, true, token.Expired(), "")
}
