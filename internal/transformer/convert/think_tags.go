package convert

import (
	"strings"

	"github.com/lich0821/ccNexus/internal/transformer"
)

const (
	thinkTagOpen  = "<think>"
	thinkTagClose = "</think>"
)

func splitThinkTaggedText(text string) []map[string]interface{} {
	var blocks []map[string]interface{}
	for {
		openIdx := strings.Index(text, thinkTagOpen)
		if openIdx == -1 {
			if text != "" {
				blocks = append(blocks, map[string]interface{}{"type": "text", "text": text})
			}
			return blocks
		}
		if openIdx > 0 {
			blocks = append(blocks, map[string]interface{}{"type": "text", "text": text[:openIdx]})
		}
		text = text[openIdx+len(thinkTagOpen):]
		closeIdx := strings.Index(text, thinkTagClose)
		if closeIdx == -1 {
			if text != "" {
				blocks = append(blocks, map[string]interface{}{"type": "text", "text": text})
			}
			return blocks
		}
		if closeIdx > 0 {
			blocks = append(blocks, map[string]interface{}{"type": "thinking", "thinking": text[:closeIdx]})
		}
		text = text[closeIdx+len(thinkTagClose):]
	}
}

func consumeThinkTaggedStream(content string, ctx *transformer.StreamContext, emitText func(string), emitThinking func(string)) {
	for len(content) > 0 {
		if ctx.InThinkingTag {
			closeIdx := strings.Index(content, thinkTagClose)
			if closeIdx == -1 {
				text, buffer := splitTrailingPartialTag(content, thinkTagClose)
				if text != "" {
					emitThinking(text)
				}
				ctx.ThinkingBuffer = buffer
				return
			}
			if closeIdx > 0 {
				emitThinking(content[:closeIdx])
			}
			ctx.InThinkingTag = false
			content = content[closeIdx+len(thinkTagClose):]
			continue
		}

		openIdx := strings.Index(content, thinkTagOpen)
		if openIdx == -1 {
			text, buffer := splitTrailingPartialTag(content, thinkTagOpen)
			emitText(text)
			ctx.ThinkingBuffer = buffer
			return
		}
		emitText(content[:openIdx])
		ctx.InThinkingTag = true
		content = content[openIdx+len(thinkTagOpen):]
	}
}

func flushThinkTaggedStream(ctx *transformer.StreamContext, emitText func(string), emitThinking func(string)) {
	if ctx.InThinkingTag {
		if ctx.ThinkingBuffer != "" {
			emitThinking(ctx.ThinkingBuffer)
		}
	} else if ctx.ThinkingBuffer != "" {
		emitText(ctx.ThinkingBuffer)
	}
	ctx.InThinkingTag = false
	ctx.ThinkingBuffer = ""
	ctx.PendingThinkingText = ""
}

func makeThinkEmitters(ctx *transformer.StreamContext, result *[]byte) (func(string), func(string)) {
	emitText := func(text string) {
		if text == "" {
			return
		}
		if !ctx.ContentBlockStarted {
			ctx.ContentBlockStarted = true
			*result = append(*result, buildClaudeEvent("content_block_start", map[string]interface{}{
				"index": ctx.ContentIndex, "content_block": map[string]interface{}{"type": "text", "text": ""},
			})...)
		}
		*result = append(*result, buildClaudeEvent("content_block_delta", map[string]interface{}{
			"index": ctx.ContentIndex, "delta": map[string]interface{}{"type": "text_delta", "text": text},
		})...)
	}

	emitThinking := func(text string) {
		if text == "" {
			return
		}
		if !ctx.ThinkingBlockStarted {
			if ctx.ContentBlockStarted {
				*result = append(*result, buildClaudeEvent("content_block_stop", map[string]interface{}{"index": ctx.ContentIndex})...)
				ctx.ContentBlockStarted = false
				ctx.ContentIndex++
			}
			ctx.ThinkingBlockStarted = true
			ctx.ThinkingIndex = ctx.ContentIndex
			ctx.ContentIndex++
			*result = append(*result, buildClaudeEvent("content_block_start", map[string]interface{}{
				"index": ctx.ThinkingIndex, "content_block": map[string]interface{}{"type": "thinking", "thinking": ""},
			})...)
		}
		*result = append(*result, buildClaudeEvent("content_block_delta", map[string]interface{}{
			"index": ctx.ThinkingIndex, "delta": map[string]interface{}{"type": "thinking_delta", "thinking": text},
		})...)
	}

	return emitText, emitThinking
}

func splitTrailingPartialTag(s, tag string) (string, string) {
	if s == "" || tag == "" {
		return s, ""
	}
	max := len(tag) - 1
	if max > len(s) {
		max = len(s)
	}
	for i := max; i > 0; i-- {
		if strings.HasPrefix(tag, s[len(s)-i:]) {
			return s[:len(s)-i], s[len(s)-i:]
		}
	}
	return s, ""
}
