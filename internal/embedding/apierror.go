package embedding

import (
	"fmt"
	"net/http"
	"pa/internal/openaicompat"
	"strings"
)

func (p *OpenAICompatible) embeddingAPIError(resp *http.Response) error {
	body := openaicompat.DecodeError(resp)
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	return formatEmbeddingAPIError(p.providerType, p.model, p.baseURL, status, body)
}

func formatEmbeddingAPIError(providerType, model, endpoint string, status int, body openaicompat.APIError) error {
	class := classifyEmbeddingHTTP(status, body.Code, body.Type)
	var b strings.Builder
	fmt.Fprintf(&b, "config.embedding type=%s model=%s endpoint=%s: ", providerType, model, endpoint)
	if class != "" {
		fmt.Fprintf(&b, "%s (HTTP %d), not llm_providers", class, status)
	} else {
		fmt.Fprintf(&b, "HTTP %d, not llm_providers", status)
	}
	if body.Code != "" {
		fmt.Fprintf(&b, " error.code=%s", body.Code)
	}
	if body.Type != "" {
		fmt.Fprintf(&b, " error.type=%s", body.Type)
	}
	if body.Message != "" {
		fmt.Fprintf(&b, ": %s", body.Message)
	}
	return fmt.Errorf("%s", b.String())
}

func classifyEmbeddingHTTP(status int, code, typ string) string {
	if status != http.StatusTooManyRequests {
		return ""
	}
	code = strings.ToLower(strings.TrimSpace(code))
	typ = strings.ToLower(strings.TrimSpace(typ))
	switch code {
	case "credit_balance_exhausted",
		"organization_spend_limit_exceeded",
		"project_spend_limit_exceeded",
		"organization_usage_limit_exceeded",
		"insufficient_quota":
		return "quota exhausted"
	case "slow_down", "rate_limit_exceeded":
		return "rate limited"
	}
	switch typ {
	case "insufficient_quota":
		return "quota exhausted"
	case "rate_limit_error":
		return "rate limited"
	}
	return ""
}
