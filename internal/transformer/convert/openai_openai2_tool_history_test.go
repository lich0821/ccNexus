package convert

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOpenAIReqToOpenAI2PreservesToolHistory(t *testing.T) {
	tests := []struct {
		name     string
		messages string
		want     string
	}{
		{
			name: "tool-only assistant turn",
			messages: `[
				{"role":"user","content":"Read the file"},
				{"role":"assistant","content":null,"tool_calls":[{"id":"call_read","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"notes.txt\"}"}}]},
				{"role":"tool","tool_call_id":"call_read","content":"file contents"},
				{"role":"assistant","content":"The file says hello."}
			]`,
			want: `[
				{"type":"message","role":"user","content":[{"type":"input_text","text":"Read the file"}]},
				{"type":"function_call","call_id":"call_read","name":"read_file","arguments":"{\"path\":\"notes.txt\"}"},
				{"type":"function_call_output","call_id":"call_read","output":"file contents"},
				{"type":"message","role":"assistant","content":[{"type":"output_text","text":"The file says hello."}]}
			]`,
		},
		{
			name: "empty assistant text and multipart tool output",
			messages: `[
				{"role":"assistant","content":"","tool_calls":[{"id":"call_parts","type":"function","function":{"name":"read_file","arguments":"{ \"path\": \"你好.txt\" }"}}]},
				{"role":"tool","tool_call_id":"call_parts","content":[{"type":"text","text":"line 1\n"},{"type":"text","text":"你好"}]}
			]`,
			want: `[
				{"type":"function_call","call_id":"call_parts","name":"read_file","arguments":"{ \"path\": \"你好.txt\" }"},
				{"type":"function_call_output","call_id":"call_parts","output":"line 1\n你好"}
			]`,
		},
		{
			name: "text and parallel calls preserve result IDs and order",
			messages: `[
				{"role":"assistant","content":"Checking both files.","tool_calls":[
					{"id":"call_a","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.txt\"}"}},
					{"id":"call_b","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"b.txt\"}"}}
				]},
				{"role":"tool","tool_call_id":"call_b","content":"B"},
				{"role":"tool","tool_call_id":"call_a","content":""}
			]`,
			want: `[
				{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Checking both files."}]},
				{"type":"function_call","call_id":"call_a","name":"read_file","arguments":"{\"path\":\"a.txt\"}"},
				{"type":"function_call","call_id":"call_b","name":"read_file","arguments":"{\"path\":\"b.txt\"}"},
				{"type":"function_call_output","call_id":"call_b","output":"B"},
				{"type":"function_call_output","call_id":"call_a","output":""}
			]`,
		},
		{
			name: "ordinary messages keep their existing shape",
			messages: `[
				{"role":"user","content":"hello"},
				{"role":"assistant","content":""},
				{"role":"assistant","content":"hello back"}
			]`,
			want: `[
				{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]},
				{"type":"message","role":"assistant","content":[{"type":"output_text","text":""}]},
				{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello back"}]}
			]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := []byte(`{"model":"source","messages":` + tt.messages + `}`)
			converted, err := OpenAIReqToOpenAI2(request, "target")
			if err != nil {
				t.Fatalf("OpenAIReqToOpenAI2 failed: %v", err)
			}
			var got map[string]interface{}
			if err := json.Unmarshal(converted, &got); err != nil {
				t.Fatal(err)
			}
			var want []interface{}
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got["input"], want) {
				t.Errorf("input = %s; want %s", converted, tt.want)
			}
		})
	}
}

func TestOpenAIReqToOpenAI2RejectsMalformedToolCalls(t *testing.T) {
	for _, request := range []string{
		`{"messages":`,
		`{"messages":[{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":{"path":"a.txt"}}}]}]}`,
	} {
		if _, err := OpenAIReqToOpenAI2([]byte(request), "target"); err == nil {
			t.Errorf("expected unmarshal error for %s", request)
		}
	}
}
