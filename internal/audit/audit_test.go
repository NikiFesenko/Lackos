package audit_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/audit"
)

func TestEvent_Validation(t *testing.T) {
	latency := int32(45)
	ev := audit.NewEvent(
		"user-123",
		"srv-456",
		"get_employee_record",
		audit.OutcomeAllowed,
		json.RawMessage(`{"emp":"1"}`),
		json.RawMessage(`{"status":"ok"}`),
		&latency,
	)

	require.NoError(t, ev.Validate())
	assert.NotEmpty(t, ev.ID)
	assert.Equal(t, "user-123", ev.UserID)
	assert.Equal(t, "srv-456", ev.DownstreamServerID)
	assert.Equal(t, "get_employee_record", ev.ToolName)
	assert.Equal(t, audit.OutcomeAllowed, ev.Outcome)
	assert.Equal(t, &latency, ev.LatencyMs)
	assert.False(t, ev.OccurredAt.IsZero())

	// Missing UserID
	badEv := ev
	badEv.UserID = ""
	assert.Error(t, badEv.Validate())

	// Missing DownstreamServerID
	badEv = ev
	badEv.DownstreamServerID = ""
	assert.Error(t, badEv.Validate())

	// Missing ToolName
	badEv = ev
	badEv.ToolName = ""
	assert.Error(t, badEv.Validate())

	// Invalid Outcome
	badEv = ev
	badEv.Outcome = "unknown_outcome"
	assert.Error(t, badEv.Validate())
}

func TestRoutingKey(t *testing.T) {
	assert.Equal(t, audit.RoutingKeyAllowed, audit.RoutingKey(audit.OutcomeAllowed))
	assert.Equal(t, audit.RoutingKeyDenied, audit.RoutingKey(audit.OutcomeDenied))
	assert.Equal(t, audit.RoutingKeyRateLimited, audit.RoutingKey(audit.OutcomeRateLimited))
	assert.Equal(t, audit.RoutingKeyError, audit.RoutingKey(audit.OutcomeError))
	assert.Equal(t, "audit.custom", audit.RoutingKey("custom"))
}

func TestMemoryPublisher(t *testing.T) {
	pub := audit.NewMemoryPublisher()
	ctx := context.Background()

	ev1 := audit.NewEvent("u1", "s1", "t1", audit.OutcomeAllowed, nil, nil, nil)
	ev2 := audit.NewEvent("u2", "s2", "t2", audit.OutcomeDenied, nil, nil, nil)

	require.NoError(t, pub.Publish(ctx, ev1))
	require.NoError(t, pub.Publish(ctx, ev2))

	events := pub.Events()
	assert.Len(t, events, 2)
	assert.Equal(t, ev1.ID, events[0].ID)
	assert.Equal(t, ev2.ID, events[1].ID)

	pub.Reset()
	assert.Empty(t, pub.Events())
	assert.NoError(t, pub.Close())
}

func TestLogAlerter(t *testing.T) {
	alerter := audit.NewLogAlerter()
	ev := audit.NewEvent("u1", "s1", "blocked_tool", audit.OutcomeDenied, nil, nil, nil)

	err := alerter.Alert(context.Background(), ev)
	assert.NoError(t, err)
}

func TestNopPublisher(t *testing.T) {
	var pub audit.NopPublisher
	ev := audit.NewEvent("u1", "s1", "t1", audit.OutcomeAllowed, nil, nil, nil)
	assert.NoError(t, pub.Publish(context.Background(), ev))
	assert.NoError(t, pub.Close())
}

func TestEvent_JSONSerialization(t *testing.T) {
	lat := int32(100)
	now := time.Now().UTC().Truncate(time.Millisecond)
	ev := audit.Event{
		ID:                  "ev-1",
		UserID:              "u-1",
		DownstreamServerID:  "s-1",
		ToolName:            "tool",
		InputParamsRedacted: json.RawMessage(`{"key":"[REDACTED]"}`),
		Outcome:             audit.OutcomeDenied,
		ResponseSummary:     json.RawMessage(`{"error":"denied"}`),
		LatencyMs:           &lat,
		OccurredAt:          now,
	}

	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var decoded audit.Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ev.ID, decoded.ID)
	assert.Equal(t, ev.UserID, decoded.UserID)
	assert.Equal(t, ev.DownstreamServerID, decoded.DownstreamServerID)
	assert.Equal(t, ev.ToolName, decoded.ToolName)
	assert.JSONEq(t, string(ev.InputParamsRedacted), string(decoded.InputParamsRedacted))
	assert.Equal(t, ev.Outcome, decoded.Outcome)
	assert.JSONEq(t, string(ev.ResponseSummary), string(decoded.ResponseSummary))
	assert.Equal(t, *ev.LatencyMs, *decoded.LatencyMs)
}
