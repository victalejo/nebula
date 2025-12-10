package events

import (
	"sync"
	"time"
)

// EventType defines the type of status event
type EventType string

const (
	EventDeploymentStatus EventType = "deployment_status"
	EventServiceStatus    EventType = "service_status"
)

// StatusEvent represents a status change event
type StatusEvent struct {
	Type         EventType `json:"type"`
	DeploymentID string    `json:"deployment_id,omitempty"`
	ServiceID    string    `json:"service_id,omitempty"`
	ProjectID    string    `json:"project_id"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    string    `json:"timestamp"`
}

// Subscriber represents a channel that receives events
type Subscriber struct {
	ID        string
	ProjectID string // Filter by project ID (empty = all projects)
	Events    chan StatusEvent
}

// EventBus manages pub/sub for status events
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]*Subscriber),
	}
}

// Subscribe creates a new subscriber for a project
func (eb *EventBus) Subscribe(id, projectID string) *Subscriber {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	sub := &Subscriber{
		ID:        id,
		ProjectID: projectID,
		Events:    make(chan StatusEvent, 100), // Buffered channel
	}
	eb.subscribers[id] = sub
	return sub
}

// Unsubscribe removes a subscriber
func (eb *EventBus) Unsubscribe(id string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if sub, ok := eb.subscribers[id]; ok {
		close(sub.Events)
		delete(eb.subscribers, id)
	}
}

// Publish sends an event to all relevant subscribers
func (eb *EventBus) Publish(event StatusEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Set timestamp if not provided
	if event.Timestamp == "" {
		event.Timestamp = time.Now().Format(time.RFC3339)
	}

	for _, sub := range eb.subscribers {
		// Filter by project ID (empty projectID = all projects)
		if sub.ProjectID == "" || sub.ProjectID == event.ProjectID {
			select {
			case sub.Events <- event:
			default:
				// Drop event if channel is full (subscriber too slow)
			}
		}
	}
}

// PublishDeploymentStatus is a convenience method for publishing deployment status changes
func (eb *EventBus) PublishDeploymentStatus(projectID, serviceID, deploymentID, status, errorMessage string) {
	eb.Publish(StatusEvent{
		Type:         EventDeploymentStatus,
		ProjectID:    projectID,
		ServiceID:    serviceID,
		DeploymentID: deploymentID,
		Status:       status,
		ErrorMessage: errorMessage,
	})
}

// PublishServiceStatus is a convenience method for publishing service status changes
func (eb *EventBus) PublishServiceStatus(projectID, serviceID, status string) {
	eb.Publish(StatusEvent{
		Type:      EventServiceStatus,
		ProjectID: projectID,
		ServiceID: serviceID,
		Status:    status,
	})
}
