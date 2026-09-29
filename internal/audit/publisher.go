package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher defines the contract for sending audit events.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
	Close() error
}

// RabbitMQPublisher implements Publisher using RabbitMQ.
// It buffers outgoing events in a memory channel so the hot-path proxy request
// is never blocked by network latency to RabbitMQ.
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	ch      *amqp.Channel
	queue   chan Event
	wg      sync.WaitGroup
	closed  chan struct{}
	closeMu sync.Mutex
	isDone  bool
}

// NewRabbitMQPublisher connects to RabbitMQ, declares the audit exchange,
// and starts the background dispatch worker.
func NewRabbitMQPublisher(url string, bufferSize int) (*RabbitMQPublisher, error) {
	if bufferSize <= 0 {
		bufferSize = 2048
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("audit: dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("audit: open channel: %w", err)
	}

	// Declare topic exchange
	err = ch.ExchangeDeclare(
		ExchangeAudit, // name
		"topic",        // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("audit: declare exchange: %w", err)
	}

	p := &RabbitMQPublisher{
		conn:   conn,
		ch:     ch,
		queue:  make(chan Event, bufferSize),
		closed: make(chan struct{}),
	}

	p.wg.Add(1)
	go p.worker()

	return p, nil
}

// Publish enqueues an event for asynchronous publishing.
// If the buffer is full, it logs an error and drops the event rather than
// causing unbounded latency in the proxy.
func (p *RabbitMQPublisher) Publish(ctx context.Context, event Event) error {
	p.closeMu.Lock()
	if p.isDone {
		p.closeMu.Unlock()
		return fmt.Errorf("audit: publisher is closed")
	}
	p.closeMu.Unlock()

	select {
	case p.queue <- event:
		return nil
	default:
		slog.ErrorContext(ctx, "audit: buffer full; dropping audit event",
			"event_id", event.ID, "tool", event.ToolName, "outcome", event.Outcome)
		return fmt.Errorf("audit: publisher buffer full")
	}
}

// worker processes events from the buffer channel and publishes to RabbitMQ.
func (p *RabbitMQPublisher) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.closed:
			// Drain remaining events in channel
			for len(p.queue) > 0 {
				event := <-p.queue
				p.send(event)
			}
			return
		case event := <-p.queue:
			p.send(event)
		}
	}
}

func (p *RabbitMQPublisher) send(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("audit: failed to marshal event", "event_id", event.ID, "err", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	routingKey := RoutingKey(event.Outcome)
	err = p.ch.PublishWithContext(
		ctx,
		ExchangeAudit, // exchange
		routingKey,    // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.ID,
			Timestamp:    event.OccurredAt,
			Body:         data,
		},
	)
	if err != nil {
		slog.Error("audit: failed to publish event to rabbitmq",
			"event_id", event.ID, "routing_key", routingKey, "err", err)
	}
}

// Close gracefully flushes pending events and closes the AMQP channel & connection.
func (p *RabbitMQPublisher) Close() error {
	p.closeMu.Lock()
	if p.isDone {
		p.closeMu.Unlock()
		return nil
	}
	p.isDone = true
	close(p.closed)
	p.closeMu.Unlock()

	p.wg.Wait()

	var chErr, connErr error
	if p.ch != nil {
		chErr = p.ch.Close()
	}
	if p.conn != nil {
		connErr = p.conn.Close()
	}
	if chErr != nil {
		return chErr
	}
	return connErr
}

// ── In-Memory Publisher (for testing and offline dev) ──────────────────────────

// MemoryPublisher collects published events in a thread-safe slice.
type MemoryPublisher struct {
	mu     sync.RWMutex
	events []Event
}

// NewMemoryPublisher creates an in-memory publisher.
func NewMemoryPublisher() *MemoryPublisher {
	return &MemoryPublisher{events: make([]Event, 0)}
}

func (m *MemoryPublisher) Publish(_ context.Context, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *MemoryPublisher) Close() error {
	return nil
}

// Events returns a copy of all published events.
func (m *MemoryPublisher) Events() []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Event, len(m.events))
	copy(out, m.events)
	return out
}

// Reset clears the recorded events.
func (m *MemoryPublisher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = m.events[:0]
}

// ── No-op Publisher ───────────────────────────────────────────────────────────

// NopPublisher silently drops all events.
type NopPublisher struct{}

func (NopPublisher) Publish(context.Context, Event) error { return nil }
func (NopPublisher) Close() error                         { return nil }
