package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type NotificationRepository interface {
	// WasSent returns true if this exact notification was already sent.
	WasSent(ctx context.Context, userID int64, notifType domain.NotificationType, refID string) (bool, error)
	// MarkSent records that the notification was sent. Silently ignores duplicates.
	MarkSent(ctx context.Context, userID int64, notifType domain.NotificationType, refID string) error
}

type notificationRepo struct{ db *sqlx.DB }

func NewNotificationRepository(db *sqlx.DB) NotificationRepository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) WasSent(ctx context.Context, userID int64, notifType domain.NotificationType, refID string) (bool, error) {
	const q = `SELECT COUNT(1) FROM notifications WHERE user_id=$1 AND type=$2 AND ref_id=$3`
	var count int
	if err := r.db.GetContext(ctx, &count, q, userID, string(notifType), refID); err != nil {
		return false, fmt.Errorf("notificationRepo.WasSent: %w", err)
	}
	return count > 0, nil
}

func (r *notificationRepo) MarkSent(ctx context.Context, userID int64, notifType domain.NotificationType, refID string) error {
	const q = `
		INSERT INTO notifications (user_id, type, ref_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, type, ref_id) DO NOTHING`
	if _, err := r.db.ExecContext(ctx, q, userID, string(notifType), refID); err != nil {
		return fmt.Errorf("notificationRepo.MarkSent: %w", err)
	}
	return nil
}
