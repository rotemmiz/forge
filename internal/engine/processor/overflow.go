package processor

import (
	"github.com/rotemmiz/forge/internal/engine/catalog"
	"github.com/rotemmiz/forge/internal/engine/message"
)

// usableContextRatio is the fraction of a model's context window Forge treats as
// usable headroom before triggering compaction (opencode overflow.ts).
const usableContextRatio = 0.8

// overflowRatio is how full the usable window / output budget may get before the
// step-finish handler flags the run for compaction.
const overflowRatio = 0.9

// isOverflow reports whether a usage block has exceeded the model's safe
// input/output budget and the run should compact (plan 02 §Overflow detection).
// A model with no known context limit never overflows.
func isOverflow(tokens message.TokenCounts, model catalog.Model) bool {
	if model.Limit.Context <= 0 {
		return false
	}
	usable := float64(model.Limit.Context) * usableContextRatio
	if model.Limit.Output > 0 && tokens.Output >= float64(model.Limit.Output)*overflowRatio {
		return true
	}
	return tokens.Input >= usable*overflowRatio
}
