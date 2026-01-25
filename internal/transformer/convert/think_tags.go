package convert

import "strings"

func splitThinkTaggedText(text string) []map[string]interface{} {
	var blocks []map[string]interface{}
	for {
		openIdx := strings.Index(text, "<think>")
		if openIdx == -1 {
			if text != "" {
				blocks = append(blocks, map[string]interface{}{"type": "text", "text": text})
			}
			return blocks
		}
		if openIdx > 0 {
			blocks = append(blocks, map[string]interface{}{"type": "text", "text": text[:openIdx]})
		}
		text = text[openIdx+len("<think>"):]
		closeIdx := strings.Index(text, "</think>")
		if closeIdx == -1 {
			if text != "" {
				blocks = append(blocks, map[string]interface{}{"type": "text", "text": text})
			}
			return blocks
		}
		if closeIdx > 0 {
			blocks = append(blocks, map[string]interface{}{"type": "thinking", "thinking": text[:closeIdx]})
		}
		text = text[closeIdx+len("</think>"):]
	}
}
