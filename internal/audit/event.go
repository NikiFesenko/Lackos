// Package audit implements the asynchronous audit pipeline.
// The proxy publishes every decision to RabbitMQ, and the worker consumer
// persists events to PostgreSQL and routes alerts.
package audit

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Allowed outcomes for audit events per database constraint.
const (
	OutcomeAllowed     = "allowed"
	OutcomeDenied      = "denied"
	OutcomeRateLimited = "rate_limited"
	OutcomeError       = "error"
)

// RabbitMQ Exchange and Queue names.
const (
	ExchangeAudit = "mcp_gate.audit"     // topic exchange
	ExchangeDLX   = "mcp_gate.audit.dlx" // dead-letter topic exchange

	QueuePersist  = "mcp_gate.audit.persist"  // stores to Postgres
	QueueAlerting = "mcp_gate.audit.alerting" // routes to security alerting
	QueueDLQ      = "mcp_gate.audit.dlq"      // dead-letter queue
)

// Standard routing keys.
const (
	RoutingKeyAllowed     = "audit.allowed"
	RoutingKeyDenied      = "audit.denied"
	RoutingKeyRateLimited = "audit.rate_limited"
	RoutingKeyError       = "audit.error"
	RoutingKeyAll         = "audit.*"
)

// RoutingKey returns the AMQP routing key for a given outcome.
func RoutingKey(outcome string) string {
	switch outcome {
	case OutcomeAllowed:
		return RoutingKeyAllowed
	case OutcomeDenied:
		return RoutingKeyDenied
	case OutcomeRateLimited:
		return RoutingKeyRateLimited
	case OutcomeError:
		return RoutingKeyError
	default:
		return "audit." + outcome
	}
}

// Event represents an audit record payload transported over RabbitMQ.
type Event struct {
	ID                  string          `json:"id"`
	UserID              string          `json:"user_id"`
	DownstreamServerID  string          `json:"downstream_server_id"`
	ToolName            string          `json:"tool_name"`
	InputParamsRedacted json.RawMessage `json:"input_params_redacted,omitempty"`
	Outcome             string          `json:"outcome"`
	ResponseSummary     json.RawMessage `json:"response_summary,omitempty"`
	LatencyMs           *int32          `json:"latency_ms,omitempty"`
	OccurredAt          time.Time       `json:"occurred_at"`
}

// NewEvent creates a new audit Event with a generated UUID and current timestamp.
func NewEvent(
	userID string,
	serverID string,
	toolName string,
	outcome string,
	inputParams json.RawMessage,
	responseSummary json.RawMessage,
	latencyMs *int32,
) Event {
	return Event{
		ID:                  uuid.New().String(),
		UserID:              userID,
		DownstreamServerID:  serverID,
		ToolName:            toolName,
		InputParamsRedacted: inputParams,
		Outcome:             outcome,
		ResponseSummary:     responseSummary,
		LatencyMs:           latencyMs,
		OccurredAt:          time.Now().UTC(),
	}
}

// Validate checks whether the Event contains required fields.
func (e Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("audit: event id is required")
	}
	if e.UserID == "" {
		return fmt.Errorf("audit: user_id is required")
	}
	if e.DownstreamServerID == "" {
		return fmt.Errorf("audit: downstream_server_id is required")
	}
	if e.ToolName == "" {
		return fmt.Errorf("audit: tool_name is required")
	}
	switch e.Outcome {
	case OutcomeAllowed, OutcomeDenied, OutcomeRateLimited, OutcomeError:
		// Valid outcome
	default:
		return fmt.Errorf("audit: invalid outcome %q", e.Outcome)
	}
	return nil
}
