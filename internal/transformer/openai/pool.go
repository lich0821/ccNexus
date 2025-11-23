package openai

import (
	"bytes"
	"sync"
)

// BufferPool provides reusable byte buffers
var BufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer gets a buffer from pool
func GetBuffer() *bytes.Buffer {
	buf := BufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to pool
func PutBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 { // Don't pool large buffers
		return
	}
	BufferPool.Put(buf)
}

// SSEEventPool provides reusable SSE events
var SSEEventPool = sync.Pool{
	New: func() interface{} {
		return &SSEEvent{
			Data: make(map[string]interface{}),
		}
	},
}

// GetSSEEvent gets an event from pool
func GetSSEEvent() *SSEEvent {
	event := SSEEventPool.Get().(*SSEEvent)
	event.Event = ""
	event.ID = ""
	event.Retry = 0
	for k := range event.Data {
		delete(event.Data, k)
	}
	return event
}

// PutSSEEvent returns an event to pool
func PutSSEEvent(event *SSEEvent) {
	if event != nil {
		SSEEventPool.Put(event)
	}
}
