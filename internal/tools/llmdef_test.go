package tools

import (
	"strings"
	"testing"
)

// Covers AC-09.008: LLM tool def from native tool includes name and parameters JSON.
func TestLLMDefFromTool(t *testing.T) {
	t.Parallel()
	d := LLMDefFromTool(NewRunOnNode(nil))
	if d.Name != "run_on_node" || !strings.Contains(d.Parameters, `"node_id"`) {
		t.Fatalf("%+v", d)
	}
}
