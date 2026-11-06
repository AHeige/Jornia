package harness

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBoundedWorkflow(t *testing.T) {
	r := New()
	calls := 0
	tool := Tool{Spec: Spec{Name: "read_state"}, Run: func(context.Context, json.RawMessage) (json.RawMessage, error) {
		calls++
		return json.RawMessage(`{"secret":"private player text"}`), nil
	}}
	if err := r.Register(tool); err != nil {
		t.Fatal(err)
	}
	if r.Register(tool) == nil {
		t.Fatal("duplicate accepted")
	}
	call := Call{ID: "one", Tool: tool.Name, Input: json.RawMessage(`{}`)}
	out, err := r.Run(context.Background(), []Call{call})
	if err != nil || len(out.Outputs) != 1 {
		t.Fatal(out, err)
	}
	raw, _ := json.Marshal(out.Trace)
	if strings.Contains(string(raw), "private") || strings.Contains(string(raw), "secret") {
		t.Fatal("private payload in trace")
	}
	if _, err = r.Run(context.Background(), []Call{call, call}); err == nil {
		t.Fatal("duplicate call accepted")
	}
	call.Tool = "shell"
	if _, err = r.Run(context.Background(), []Call{call}); err == nil {
		t.Fatal("unknown tool accepted")
	}
	call.Tool = tool.Name
	r.MaxBytes = 2
	if _, err = r.Run(context.Background(), []Call{call}); err == nil {
		t.Fatal("budget ignored")
	}
	r.MaxCalls = 1
	if _, err = r.Run(context.Background(), []Call{call, call}); err == nil {
		t.Fatal("call budget ignored")
	}
}
func TestFailureNeverRetriesAndDeadline(t *testing.T) {
	r := New()
	calls := 0
	r.Register(Tool{Spec: Spec{Name: "read_state"}, Run: func(context.Context, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, errors.New("private error")
	}})
	_, err := r.Run(context.Background(), []Call{{ID: "1", Tool: "read_state", Input: json.RawMessage(`{}`)}})
	if err == nil || calls != 1 || strings.Contains(err.Error(), "private") {
		t.Fatal(err, calls)
	}
	r = New()
	r.Timeout = time.Millisecond
	r.Register(Tool{Spec: Spec{Name: "read_state"}, Run: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
		<-ctx.Done()
		return json.RawMessage(`{}`), nil
	}})
	out, err := r.Run(context.Background(), []Call{{ID: "1", Tool: "read_state", Input: json.RawMessage(`{}`)}})
	if err == nil || len(out.Outputs) != 0 || out.Trace[0].Status != "expired" {
		t.Fatal(out, err)
	}
}
