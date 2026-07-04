// Package breaker decorates notify.Notifier with a circuit breaker
// (sony/gobreaker), per docs/COVER_LETTER.md — the seam left open early
// specifically for this. Wired only in main.go; team.Service is unaware it
// exists, since both LogNotifier and Notifier satisfy notify.Notifier.
package breaker

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker/v2"

	"github.com/zhuk/team-task-service/internal/notify"
)

// NotifierSettings are the production gobreaker.Settings for the invite
// email breaker: trip after 3 consecutive failures, stay open for 30s,
// then allow one trial request. Exported (not hardcoded in NewNotifier) so
// tests can supply a lower threshold/timeout instead of waiting on real
// production values.
func NotifierSettings() gobreaker.Settings {
	return gobreaker.Settings{
		Name:        "notify.SendInvite",
		MaxRequests: 1,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	}
}

// Notifier decorates a notify.Notifier with a circuit breaker: once open,
// it fails fast without calling the underlying notifier until Timeout
// elapses, then allows a single trial call to test recovery. The caller
// (team.Service) already treats SendInvite failures as best-effort, so a
// breaker-open error surfaces the same way any other SendInvite failure
// does — logged, not fatal to the request.
type Notifier struct {
	next notify.Notifier
	cb   *gobreaker.CircuitBreaker[any]
}

func NewNotifier(next notify.Notifier, settings gobreaker.Settings) *Notifier {
	return &Notifier{next: next, cb: gobreaker.NewCircuitBreaker[any](settings)}
}

func (n *Notifier) SendInvite(ctx context.Context, email string, teamID int64) error {
	_, err := n.cb.Execute(func() (any, error) {
		return nil, n.next.SendInvite(ctx, email, teamID)
	})
	if err != nil {
		return fmt.Errorf("send invite via circuit breaker: %w", err)
	}
	return nil
}
