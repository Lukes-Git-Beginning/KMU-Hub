package caldav

import (
	"bytes"
	"context"
	"encoding/xml"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	// pushNotifyTimeout is the HTTP timeout for each push notification request.
	pushNotifyTimeout = 5 * time.Second
)

// pushMessage is the WebDAV-Push draft format notification body.
type pushMessage struct {
	XMLName xml.Name `xml:"DAV: push-message"`
	Topic   string   `xml:"DAV: topic"`
}

// PushNotifier sends push notifications to subscribed clients when
// calendar or address book data changes. Follows the WebDAV-Push draft
// specification for clients that support it (primarily macOS Calendar).
type PushNotifier struct {
	subscriptions *PushSubscriptionService
	httpClient    *http.Client
}

// NewPushNotifier creates a new push notifier.
func NewPushNotifier(subscriptions *PushSubscriptionService) *PushNotifier {
	return &PushNotifier{
		subscriptions: subscriptions,
		httpClient: &http.Client{
			Timeout: pushNotifyTimeout,
		},
	}
}

// NotifyCollectionChanged sends push notifications to all active subscribers
// of the given collection. Notifications are fire-and-forget: failures are
// logged but never block the CalDAV write operation.
//
// If a push endpoint returns 410 Gone, the subscription is auto-removed.
// Other errors are logged at warn level.
func (n *PushNotifier) NotifyCollectionChanged(ctx context.Context, collectionType string, collectionID uuid.UUID) error {
	if n == nil || n.subscriptions == nil {
		return nil
	}

	subs, err := n.subscriptions.GetSubscriptionsForCollection(ctx, collectionType, collectionID)
	if err != nil {
		slog.Warn("failed to get push subscriptions for notification",
			"collection_type", collectionType,
			"collection_id", collectionID,
			"error", err,
		)
		return nil // Don't fail the write
	}

	if len(subs) == 0 {
		return nil
	}

	// Clean up expired subscriptions opportunistically
	go func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, cleanupErr := n.subscriptions.CleanupExpired(cleanupCtx); cleanupErr != nil {
			slog.Warn("failed to cleanup expired push subscriptions", "error", cleanupErr)
		}
	}()

	// Send notifications in a goroutine to avoid blocking the write path
	go func() {
		notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for _, sub := range subs {
			n.sendNotification(notifyCtx, sub)
		}
	}()

	return nil
}

// sendNotification sends a single push notification to a subscriber.
func (n *PushNotifier) sendNotification(ctx context.Context, sub *PushSubscription) {
	topic := sub.PushTopic
	if topic == "" {
		topic = sub.CollectionID.String()
	}

	msg := pushMessage{Topic: topic}
	body, err := xml.Marshal(msg)
	if err != nil {
		slog.Warn("failed to marshal push message",
			"subscription_id", sub.ID,
			"error", err,
		)
		return
	}

	xmlBody := append([]byte(xml.Header), body...)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.PushURL, bytes.NewReader(xmlBody))
	if err != nil {
		slog.Warn("failed to create push notification request",
			"subscription_id", sub.ID,
			"push_url", sub.PushURL,
			"error", err,
		)
		return
	}
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		slog.Warn("push notification delivery failed",
			"subscription_id", sub.ID,
			"push_url", sub.PushURL,
			"error", err,
		)
		return
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusGone:
		// 410 Gone: client no longer wants push notifications -- auto-unsubscribe
		slog.Info("push endpoint returned 410 Gone, removing subscription",
			"subscription_id", sub.ID,
			"push_url", sub.PushURL,
		)
		removeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if removeErr := n.subscriptions.UnsubscribeByURL(removeCtx, sub.PushURL); removeErr != nil {
			slog.Warn("failed to remove subscription after 410 Gone",
				"subscription_id", sub.ID,
				"error", removeErr,
			)
		}

	case resp.StatusCode >= 400:
		slog.Warn("push notification returned error status",
			"subscription_id", sub.ID,
			"push_url", sub.PushURL,
			"status", resp.StatusCode,
		)

	default:
		slog.Info("push notification delivered",
			"subscription_id", sub.ID,
			"collection_type", sub.CollectionType,
			"collection_id", sub.CollectionID,
			"status", resp.StatusCode,
		)
	}
}
