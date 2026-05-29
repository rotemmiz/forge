package processor

import (
	"context"
	"time"

	"github.com/rotemmiz/forge/internal/bus"
	"github.com/rotemmiz/forge/internal/engine/llm"
	"github.com/rotemmiz/forge/internal/engine/message"
)

// updatePart persists a part and emits message.part.updated.
func (p *Processor) updatePart(ctx context.Context, part message.Part) {
	if err := p.cfg.Store.PutPart(ctx, part); err != nil {
		return // a write failure on a cancelled ctx is expected; nothing to surface here
	}
	if p.cfg.Bus != nil {
		p.cfg.Bus.Publish(bus.NewEvent("message.part.updated", map[string]any{
			"sessionID": p.cfg.SessionID,
			"part":      part,
			"time":      time.Now().UnixMilli(),
		}))
	}
}

// publishDelta emits a streaming message.part.delta (SSE-only; the accumulated
// part is persisted on its -end event).
func (p *Processor) publishDelta(partID, messageID, field, delta string) {
	if p.cfg.Bus == nil {
		return
	}
	p.cfg.Bus.Publish(bus.NewEvent("message.part.delta", map[string]any{
		"sessionID": p.cfg.SessionID,
		"messageID": messageID,
		"partID":    partID,
		"field":     field,
		"delta":     delta,
	}))
}

// updateMessage persists the assistant message and emits message.updated.
func (p *Processor) updateMessage(ctx context.Context) {
	if err := p.cfg.Store.PutMessage(ctx, message.Info{Assistant: p.assistant}); err != nil {
		return
	}
	if p.cfg.Bus != nil {
		p.cfg.Bus.Publish(bus.NewEvent("message.updated", map[string]any{
			"sessionID": p.cfg.SessionID,
			"info":      message.Info{Assistant: p.assistant},
		}))
	}
}

// cleanup finalizes in-flight parts: open text/reasoning are written, any tool
// still running is marked interrupted, and the assistant message is completed.
// A cancelled context marks the message aborted (processor.ts:691-749).
func (p *Processor) cleanup(ctx context.Context) {
	if t := p.takeCurrentText(); t != nil {
		end := time.Now().UnixMilli()
		if t.Time == nil {
			t.Time = &message.PartTime{Start: end}
		}
		t.Time.End = &end
		p.updatePart(ctx, t)
	}
	p.finishAllReasoning(ctx)

	p.mu.Lock()
	order := append([]string(nil), p.toolOrder...)
	p.mu.Unlock()
	for _, callID := range order {
		p.mu.Lock()
		part := p.tools[callID]
		interrupted := part != nil && (part.Status() == message.ToolRunning || part.Status() == message.ToolPending)
		p.mu.Unlock()
		if interrupted {
			p.interruptCall(ctx, callID)
		}
	}

	completed := time.Now().UnixMilli()
	p.mu.Lock()
	if ctx.Err() != nil && p.assistant.Error == nil {
		p.assistant.Error = &message.Error{Name: "MessageAbortedError", Data: map[string]any{"message": "aborted"}}
	}
	p.assistant.Time.Completed = &completed
	p.mu.Unlock()
	p.updateMessage(ctx)
}

func (p *Processor) takeCurrentText() *message.TextPart {
	p.mu.Lock()
	defer p.mu.Unlock()
	t := p.currentText
	p.currentText = nil
	return t
}

// interruptCall marks a still-running tool as error+interrupted.
func (p *Processor) interruptCall(ctx context.Context, callID string) {
	p.mu.Lock()
	part, ok := p.tools[callID]
	if !ok {
		p.mu.Unlock()
		return
	}
	start := startOf(part)
	st := message.ToolStateError{Status: message.ToolError, Input: inputOf(part),
		Error: "Tool execution aborted", Metadata: map[string]any{"interrupted": true}}
	st.Time.Start, st.Time.End = start, time.Now().UnixMilli()
	part.State = mustState(st)
	p.mu.Unlock()
	p.updatePart(ctx, part)
}

// tokensFromUsage converts provider usage into the stored token block.
func tokensFromUsage(u *llm.TokenUsage) message.TokenCounts {
	var tc message.TokenCounts
	if u == nil {
		return tc
	}
	tc.Input = u.Input
	tc.Output = u.Output
	tc.Reasoning = u.Reasoning
	tc.Cache.Read = u.CacheRead
	tc.Cache.Write = u.CacheWrite
	total := tc.Input + tc.Output + tc.Reasoning + tc.Cache.Read + tc.Cache.Write
	tc.Total = &total
	return tc
}
