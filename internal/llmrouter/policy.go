package llmrouter

// DecideCompleteError maps a completion error class to next routing action.
func DecideCompleteError(class FailureClass, hasNext bool) Action {
	if IsTransportRetryable(class) && hasNext {
		return ActionSwitchNextTransport
	}
	return ActionStop
}

// DecideToolFailure decides whether to escalate after qualifying tool/hermes failures.
func DecideToolFailure(st *State, escalationEnabled bool, maxEsc int, hasNext bool) Action {
	if st == nil || !escalationEnabled {
		return ActionStop
	}
	if st.EscUsed >= maxEsc {
		return ActionStop
	}
	if !hasNext {
		return ActionStop
	}
	return ActionEscalatePolicy
}
