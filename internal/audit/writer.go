package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/mcp-gate/mcp-gate/internal/db"
)

// Writer consumes audit events from RabbitMQ and persists them to PostgreSQL.
type Writer struct {
	queries db.Querier
	conn    *amqp.Connection
	ch      *amqp.Channel
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewWriter sets up the RabbitMQ topology (main exchange, DLX, DLQ, persist queue)
// and prepares the Writer consumer.
func NewWriter(queries db.Querier, amqpURL string) (*Writer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("audit writer: dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("audit writer: open channel: %w", err)
	}

	// 1. Declare dead-letter exchange (topic)
	err = ch.ExchangeDeclare(
		ExchangeDLX,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("audit writer: declare dlx: %w", err)
	}

	// 2. Declare dead-letter queue (DLQ)
	_, err = ch.QueueDeclare(
		QueueDLQ,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("audit writer: declare dlq: %w", err)
	}

	// Bind DLQ to DLX for all keys
	err = ch.QueueBind(QueueDLQ, "#", ExchangeDLX, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("audit writer: bind dlq: %w", err)
	}

	// 3. Declare primary topic exchange
	err = ch.ExchangeDeclare(
		ExchangeAudit,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("audit writer: declare audit exchange: %w", err)
	}

	// 4. Declare persist queue with DLX routing arguments
	queueArgs := amqp.Table{
		"x-dead-letter-exchange": ExchangeDLX,
	}
	_, err = ch.QueueDeclare(
		QueuePersist,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		queueArgs,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("audit writer: declare persist queue: %w", err)
	}

	// Bind persist queue to all audit events (audit.*)
	err = ch.QueueBind(QueuePersist, RoutingKeyAll, ExchangeAudit, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("audit writer: bind persist queue: %w", err)
	}

	// Set prefetch count to maintain high throughput without memory exhaustion
	if err := ch.Qos(100, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("audit writer: set qos: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Writer{
		queries: queries,
		conn:    conn,
		ch:      ch,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Start begins consuming audit events in a background goroutine.
func (w *Writer) Start() error {
	msgs, err := w.ch.Consume(
		QueuePersist,
		"mcp_gate_audit_writer", // consumer tag
		false,                   // manual ack
		false,                   // exclusive
		false,                   // no-local
		false,                   // no-wait
		nil,                     // args
	)
	if err != nil {
		return fmt.Errorf("audit writer: start consume: %w", err)
	}

	w.wg.Add(1)
	go w.consumeLoop(msgs)

	slog.Info("audit writer: started consuming", "queue", QueuePersist)
	return nil
}

func (w *Writer) consumeLoop(msgs <-chan amqp.Delivery) {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			slog.Info("audit writer: stopping consume loop")
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Warn("audit writer: deliveries channel closed")
				return
			}
			w.handleDelivery(msg)
		}
	}
}

func (w *Writer) handleDelivery(msg amqp.Delivery) {
	var event Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("audit writer: unmarshal error; routing to DLQ", "err", err, "body", string(msg.Body))
		// Reject without requeue -> triggers RabbitMQ dead-letter exchange
		_ = msg.Nack(false, false)
		return
	}

	if err := event.Validate(); err != nil {
		slog.Error("audit writer: validation error; routing to DLQ", "err", err, "event_id", event.ID)
		_ = msg.Nack(false, false)
		return
	}

	if err := w.persist(w.ctx, event); err != nil {
		slog.Error("audit writer: failed to persist event to postgres", "event_id", event.ID, "err", err)
		// Requeue for transient database connection drops
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}

func (w *Writer) persist(ctx context.Context, event Event) error {
	var idUUID, userUUID, serverUUID pgtype.UUID
	if err := idUUID.Scan(event.ID); err != nil {
		return fmt.Errorf("invalid event id UUID %q: %w", event.ID, err)
	}
	if err := userUUID.Scan(event.UserID); err != nil {
		return fmt.Errorf("invalid user_id UUID %q: %w", event.UserID, err)
	}
	if err := serverUUID.Scan(event.DownstreamServerID); err != nil {
		return fmt.Errorf("invalid downstream_server_id UUID %q: %w", event.DownstreamServerID, err)
	}

	occurredAt := pgtype.Timestamptz{
		Time:  event.OccurredAt,
		Valid: !event.OccurredAt.IsZero(),
	}

	// Idempotent insertion: ON CONFLICT (id) DO NOTHING
	_, err := w.queries.InsertAuditEventWithID(ctx, db.InsertAuditEventWithIDParams{
		ID:                  idUUID,
		UserID:              userUUID,
		DownstreamServerID:  serverUUID,
		ToolName:            event.ToolName,
		InputParamsRedacted: event.InputParamsRedacted,
		Outcome:             event.Outcome,
		ResponseSummary:     event.ResponseSummary,
		LatencyMs:           event.LatencyMs,
		OccurredAt:          occurredAt,
	})
	return err
}

// Stop cancels consumption and cleanly closes the RabbitMQ channel & connection.
func (w *Writer) Stop() error {
	w.cancel()
	w.wg.Wait()

	var chErr, connErr error
	if w.ch != nil {
		chErr = w.ch.Close()
	}
	if w.conn != nil {
		connErr = w.conn.Close()
	}
	if chErr != nil {
		return chErr
	}
	return connErr
}
