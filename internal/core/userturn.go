package core

import "sync/atomic"

var userTurnDepth atomic.Int32

// EnterUserTurn marks the start of handling a user message (LLM path). Used by background summarization to defer work (EP-002 REQ-02.015).
func EnterUserTurn() { userTurnDepth.Add(1) }

// LeaveUserTurn marks the end of handling a user message. Must pair with EnterUserTurn (defer).
func LeaveUserTurn() { userTurnDepth.Add(-1) }

// UserTurnInProgress reports whether at least one user turn is active.
func UserTurnInProgress() bool { return userTurnDepth.Load() > 0 }
