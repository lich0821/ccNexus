package convert

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestClaudeReqToGeminiToolChoice(t *testing.T) {
	tests := []struct {
		name       string
		toolChoice string
		wantMode   string
		wantNames  []interface{}
	}{
		{"omitted", ``, "AUTO", nil},
		{"auto", `,"tool_choice":{"type":"auto"}`, "AUTO", nil},
		{"any", `,"tool_choice":{"type":"any"}`, "ANY", nil},
		{"tool", `,"tool_choice":{"type":"tool","name":"get_weather"}`, "ANY", []interface{}{"get_weather"}},
		{"none", `,"tool_choice":{"type":"none"}`, "NONE", nil},
		{"string any", `,"tool_choice":"any"`, "ANY", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claudeReq := `{"model":"claude","max_tokens":100,"messages":[{"role":"user","content":"hi"}],
				"tools":[{"name":"get_weather","description":"d","input_schema":{"type":"object","properties":{}}}]` + tt.toolChoice + `}`

			out, err := ClaudeReqToGemini([]byte(claudeReq), "gemini-2.5-pro")
			if err != nil {
				t.Fatalf("ClaudeReqToGemini failed: %v", err)
			}

			var got map[string]interface{}
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("invalid output: %v", err)
			}
			toolConfig, _ := got["toolConfig"].(map[string]interface{})
			fcc, _ := toolConfig["functionCallingConfig"].(map[string]interface{})
			if fcc["mode"] != tt.wantMode {
				t.Errorf("mode = %v, want %v", fcc["mode"], tt.wantMode)
			}
			gotNames, _ := fcc["allowedFunctionNames"].([]interface{})
			if !reflect.DeepEqual(gotNames, tt.wantNames) {
				t.Errorf("allowedFunctionNames = %v, want %v", gotNames, tt.wantNames)
			}
		})
	}
}
