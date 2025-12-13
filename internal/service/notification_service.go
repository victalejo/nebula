package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/victalejo/nebula/internal/core/events"
	"github.com/victalejo/nebula/internal/core/logger"
)

const (
	whatsappAPIURL = "https://wapi.iaportafolio.com/api/sendText"
	whatsappAPIKey = "ZR1UZEUaANUd2UUke3ZTbdFtCrXEwQV7"
	whatsappChatID = "120363409327342712@g.us"
	whatsappSession = "notificaciones"
)

type NotificationService struct {
	eventBus   *events.EventBus
	httpClient *http.Client
	log        logger.Logger
}

type whatsappMessage struct {
	ChatID      string `json:"chatId"`
	Text        string `json:"text"`
	LinkPreview bool   `json:"linkPreview"`
	Session     string `json:"session"`
}

func NewNotificationService(eventBus *events.EventBus, log logger.Logger) *NotificationService {
	return &NotificationService{
		eventBus:   eventBus,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		log:        log,
	}
}

func (s *NotificationService) Start(ctx context.Context) {
	subscriber := s.eventBus.Subscribe("notification-service", "")
	s.log.Info("notification service started, listening for deployment events")

	go s.listenEvents(ctx, subscriber)
}

func (s *NotificationService) listenEvents(ctx context.Context, sub *events.Subscriber) {
	for {
		select {
		case <-ctx.Done():
			s.eventBus.Unsubscribe(sub.ID)
			s.log.Info("notification service stopped")
			return
		case event, ok := <-sub.Events:
			if !ok {
				return
			}
			// Only notify on deployment status changes for running and failed
			if event.Type == events.EventDeploymentStatus {
				if event.Status == "running" || event.Status == "failed" {
					s.sendNotification(event)
				}
			}
		}
	}
}

func (s *NotificationService) sendNotification(event events.StatusEvent) {
	message := s.formatMessage(event)

	payload := whatsappMessage{
		ChatID:      whatsappChatID,
		Text:        message,
		LinkPreview: true,
		Session:     whatsappSession,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("failed to marshal whatsapp message", "error", err)
		return
	}

	req, err := http.NewRequest("POST", whatsappAPIURL, bytes.NewBuffer(body))
	if err != nil {
		s.log.Error("failed to create whatsapp request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Key", whatsappAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.log.Error("failed to send whatsapp notification", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.log.Info("whatsapp notification sent", "status", event.Status, "deployment_id", event.DeploymentID)
	} else {
		s.log.Warn("whatsapp notification failed", "status_code", resp.StatusCode)
	}
}

func (s *NotificationService) formatMessage(event events.StatusEvent) string {
	switch event.Status {
	case "running":
		return "✅ *Despliegue exitoso*\n" +
			"📦 Deployment: " + event.DeploymentID + "\n" +
			"🕐 " + event.Timestamp
	case "failed":
		msg := "❌ *Despliegue fallido*\n" +
			"📦 Deployment: " + event.DeploymentID + "\n" +
			"🕐 " + event.Timestamp
		if event.ErrorMessage != "" {
			msg += "\n⚠️ Error: " + event.ErrorMessage
		}
		return msg
	default:
		return "📢 Deployment " + event.DeploymentID + ": " + event.Status
	}
}
