package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Alerter handles real-time alerts for security-relevant audit outcomes.
type Alerter interface {
	Alert(ctx context.Context, event Event) error
}

// LogAlerter is the Phase 6 default alerting implementation.
// It emits structured security warning logs, ready to be forwarded to
// SIEM / Slack / PagerDuty in future phases.
type LogAlerter struct{}

func NewLogAlerter() *LogAlerter {
	return &LogAlerter{}
}

func (l *LogAlerter) Alert(ctx context.Context, event Event) error {
	slog.WarnContext(ctx, "SECURITY_ALERT: audit tool invocation blocked",
		"outcome", event.Outcome,
		"event_id", event.ID,
		"user_id", event.UserID,
		"server_id", event.DownstreamServerID,
		"tool", event.ToolName,
		"occurred_at", event.OccurredAt,
	)
	return nil
}

// AlertConsumer consumes security audit events from RabbitMQ and triggers alerts.
type AlertConsumer struct {
	alerter Alerter
	conn    *amqp.Connection
	ch      *amqp.Channel
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewAlertConsumer connects to RabbitMQ and binds to audit.denied and audit.rate_limited.
func NewAlertConsumer(alerter Alerter, amqpURL string) (*AlertConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("alert consumer: dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("alert consumer: open channel: %w", err)
	}

	// Declare alerting queue
	_, err = ch.QueueDeclare(
		QueueAlerting,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("alert consumer: declare queue: %w", err)
	}

	// Bind to denied and rate_limited outcomes
	for _, key := range []string{RoutingKeyDenied, RoutingKeyRateLimited} {
		if err := ch.QueueBind(QueueAlerting, key, ExchangeAudit, false, nil); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("alert consumer: bind routing key %q: %w", key, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &AlertConsumer{
		alerter: alerter,
		conn:    conn,
		ch:      ch,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Start begins processing alerting deliveries.
func (c *AlertConsumer) Start() error {
	msgs, err := c.ch.Consume(
		QueueAlerting,
		"mcp_gate_alert_consumer",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("alert consumer: start consume: %w", err)
	}

	c.wg.Add(1)
	go c.consumeLoop(msgs)

	slog.Info("alert consumer: started consuming", "queue", QueueAlerting)
	return nil
}

func (c *AlertConsumer) consumeLoop(msgs <-chan amqp.Delivery) {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			var event Event
			if err := json.Unmarshal(msg.Body, &event); err == nil {
				_ = c.alerter.Alert(c.ctx, event)
			}
			_ = msg.Ack(false)
		}
	}
}

// Stop terminates the alert consumer.
func (c *AlertConsumer) Stop() error {
	c.cancel()
	c.wg.Wait()

	var chErr, connErr error
	if c.ch != nil {
		chErr = c.ch.Close()
	}
	if c.conn != nil {
		connErr = c.conn.Close()
	}
	if chErr != nil {
		return chErr
	}
	return connErr
}
