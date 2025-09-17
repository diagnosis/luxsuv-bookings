package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, data interface{}) error
	Close() error
}

type Subscriber interface {
	Subscribe(subject string, handler func(msg *Message)) error
	QueueSubscribe(subject, queue string, handler func(msg *Message)) error
	Close() error
}

type EventBus interface {
	Publisher
	Subscriber
}

type Message struct {
	Subject   string
	Data      []byte
	Timestamp time.Time
	ID        string
}

type NATSEventBus struct {
	conn *nats.Conn
}

func NewNATSEventBus(url string) (*NATSEventBus, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	
	return &NATSEventBus{conn: conn}, nil
}

func (n *NATSEventBus) Publish(ctx context.Context, subject string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}
	
	logger.DebugContext(ctx, "Publishing event", "subject", subject, "data", string(payload))
	
	return n.conn.Publish(subject, payload)
}

func (n *NATSEventBus) Subscribe(subject string, handler func(msg *Message)) error {
	_, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(&Message{
			Subject:   msg.Subject,
			Data:      msg.Data,
			Timestamp: time.Now(),
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		})
	})
	return err
}

func (n *NATSEventBus) QueueSubscribe(subject, queue string, handler func(msg *Message)) error {
	_, err := n.conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		handler(&Message{
			Subject:   msg.Subject,
			Data:      msg.Data,
			Timestamp: time.Now(),
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		})
	})
	return err
}

func (n *NATSEventBus) Close() error {
	n.conn.Close()
	return nil
}

// Event types and subjects
const (
	// Booking events
	BookingCreated  = "booking.created"
	BookingUpdated  = "booking.updated"
	BookingCanceled = "booking.canceled"
	
	// Dispatch events
	DispatchAssignRequested = "dispatch.assign.requested"
	DispatchAssigned        = "dispatch.assigned"
	DispatchReassigned      = "dispatch.reassigned"
	DispatchDeclined        = "dispatch.declined"
	
	// Driver events
	DriverAccepted   = "driver.accepted"
	DriverDeclined   = "driver.declined"
	TripStarted      = "trip.started"
	TripCompleted    = "trip.completed"
	
	// Payment events
	PaymentIntentCreated = "payment.intent.created"
	PaymentCaptured      = "payment.captured"
	PaymentRefunded      = "payment.refunded"
	PaymentFailed        = "payment.failed"
	
	// Notification events
	NotifySend = "notify.send"
)

// Event payloads
type BookingCreatedEvent struct {
	BookingID   int64     `json:"booking_id"`
	RiderEmail  string    `json:"rider_email"`
	RiderName   string    `json:"rider_name"`
	Pickup      string    `json:"pickup"`
	Dropoff     string    `json:"dropoff"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Passengers  int       `json:"passengers"`
	CreatedAt   time.Time `json:"created_at"`
}

type BookingUpdatedEvent struct {
	BookingID   int64     `json:"booking_id"`
	RiderEmail  string    `json:"rider_email"`
	Changes     []string  `json:"changes"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BookingCanceledEvent struct {
	BookingID  int64     `json:"booking_id"`
	RiderEmail string    `json:"rider_email"`
	Reason     string    `json:"reason"`
	CanceledAt time.Time `json:"canceled_at"`
}

type DispatchAssignRequestedEvent struct {
	BookingID int64 `json:"booking_id"`
	DriverID  int64 `json:"driver_id"`
}

type DispatchAssignedEvent struct {
	BookingID    int64     `json:"booking_id"`
	DriverID     int64     `json:"driver_id"`
	AssignedAt   time.Time `json:"assigned_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type DriverAcceptedEvent struct {
	BookingID  int64     `json:"booking_id"`
	DriverID   int64     `json:"driver_id"`
	AcceptedAt time.Time `json:"accepted_at"`
}

type TripStartedEvent struct {
	BookingID int64     `json:"booking_id"`
	DriverID  int64     `json:"driver_id"`
	StartedAt time.Time `json:"started_at"`
}

type TripCompletedEvent struct {
	BookingID   int64     `json:"booking_id"`
	DriverID    int64     `json:"driver_id"`
	CompletedAt time.Time `json:"completed_at"`
}

type PaymentIntentCreatedEvent struct {
	BookingID    int64  `json:"booking_id"`
	IntentID     string `json:"intent_id"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
	ClientSecret string `json:"client_secret"`
}

type NotificationEvent struct {
	Type      string                 `json:"type"`
	Recipient string                 `json:"recipient"`
	Subject   string                 `json:"subject"`
	Template  string                 `json:"template"`
	Data      map[string]interface{} `json:"data"`
}