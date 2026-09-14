package notifier

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/galaxy-io/filament/events"
)

// The webhook event shapes. These are the customer-facing contract for each
// exported event kind, kept apart from the internal catalog so the catalog
// can change without breaking receivers. Fields are only ever added.
type (
	// RunCompleted reports a run that landed every resource.
	RunCompleted struct {
		Type    string    `json:"type"`
		RunID   string    `json:"run_id"`
		At      time.Time `json:"at"`
		Records int64     `json:"records"`
		Bytes   int64     `json:"bytes"`
	}

	// RunFailed reports a run that ended in failure.
	RunFailed struct {
		Type  string    `json:"type"`
		RunID string    `json:"run_id"`
		At    time.Time `json:"at"`
	}
)

// Project renders an exported fact as its webhook event. Every kind in
// Exported must have a case here.
func Project(f events.Fact) (json.RawMessage, error) {
	var body any
	switch d := f.Data.(type) {
	case events.RunCompletedEvent:
		body = RunCompleted{Type: f.Name, RunID: string(f.Run), At: f.At, Records: d.Records, Bytes: d.Bytes}
	case events.RunFailedEvent:
		body = RunFailed{Type: f.Name, RunID: string(f.Run), At: f.At}
	default:
		return nil, fmt.Errorf("notifier: %s has no webhook payload", f.Name)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("notifier: encode %s payload: %w", f.Name, err)
	}
	return raw, nil
}
