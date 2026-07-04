// Package notify defines the outbound "email service" seam. Phase 1 ships
// only a logging mock (SendInvite is best-effort — see team_service.go); a
// later phase wraps Notifier with a circuit breaker without touching any
// caller, since callers only depend on this interface.
package notify

import "context"

type Notifier interface {
	SendInvite(ctx context.Context, email string, teamID int64) error
}
