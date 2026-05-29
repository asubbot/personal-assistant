package llmrouter

// DecideCompleteError maps a completion error class to next routing action.
func DecideCompleteError(class FailureClass, hasNext bool) Action {
	if IsTransportRetryable(class) && hasNext {
		return ActionSwitchNextTransport
	}
	return ActionStop
}
