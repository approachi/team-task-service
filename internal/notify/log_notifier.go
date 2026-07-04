package notify

import (
	"context"
	"log/slog"
)

// LogNotifier is the Phase 1 mock "email service": it just logs the invite
// instead of sending a real email.
//
// TODO: заменить на реальный email-провайдер (SES/SendGrid/Postmark) и
// отправлять письмо асинхронно через очередь (см. docs/COVER_LETTER.md,
// раздел "Функциональность"), а не синхронно в хендлере Invite — иначе
// недоступность внешнего почтового API напрямую замедляет ответ API.
type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

func (n *LogNotifier) SendInvite(_ context.Context, email string, teamID int64) error {
	slog.Info("mock invite email sent", "email", email, "team_id", teamID)
	return nil
}
