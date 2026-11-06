// Package harness owns Wisp's bounded tool workflows. No agent framework or arbitrary code execution.
package harness

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

type Spec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type Tool struct {
	Spec
	Run func(context.Context, json.RawMessage) (json.RawMessage, error)
}
type Call struct {
	ID    string
	Tool  string
	Input json.RawMessage
}
type Event struct {
	Tool       string `json:"tool"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
}
type Result struct {
	Outputs map[string]json.RawMessage
	Trace   []Event
}
type Runtime struct {
	tools    map[string]Tool
	MaxCalls int
	MaxBytes int
	Timeout  time.Duration
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,47}$`)

func New() *Runtime {
	return &Runtime{tools: map[string]Tool{}, MaxCalls: 6, MaxBytes: 48000, Timeout: 10 * time.Second}
}
func (r *Runtime) Register(t Tool) error {
	if !namePattern.MatchString(t.Name) || t.Run == nil {
		return errors.New("invalid tool")
	}
	if _, ok := r.tools[t.Name]; ok {
		return errors.New("duplicate tool")
	}
	r.tools[t.Name] = t
	return nil
}
func (r *Runtime) Run(ctx context.Context, calls []Call) (Result, error) {
	out := Result{Outputs: map[string]json.RawMessage{}, Trace: []Event{}}
	if len(calls) > r.MaxCalls || r.MaxCalls < 1 || r.MaxBytes < 1 || r.Timeout <= 0 {
		return out, errors.New("workflow exceeds limits")
	}
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	seen := map[string]bool{}
	budget := 0
	for _, c := range calls {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		tool, ok := r.tools[c.Tool]
		if !ok || c.ID == "" || seen[c.ID] || len(c.Input) > 8192 || !json.Valid(c.Input) {
			return out, errors.New("tool call rejected")
		}
		seen[c.ID] = true
		began := time.Now()
		raw, err := tool.Run(ctx, c.Input)
		event := Event{Tool: c.Tool, Status: "completed", DurationMS: time.Since(began).Milliseconds()}
		if err != nil {
			event.Status = "failed"
			out.Trace = append(out.Trace, event)
			return out, errors.New("tool failed")
		}
		budget += len(c.Input) + len(raw)
		if !json.Valid(raw) || budget > r.MaxBytes {
			event.Status = "rejected"
			out.Trace = append(out.Trace, event)
			return out, errors.New("tool output exceeds limits")
		}
		if err = ctx.Err(); err != nil {
			event.Status = "expired"
			out.Trace = append(out.Trace, event)
			return out, err
		}
		out.Trace = append(out.Trace, event)
		out.Outputs[c.ID] = raw
	}
	return out, nil
}
