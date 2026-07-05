package breaker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/platform/breaker"
)

type fakeNotifier struct {
	calls int
	fn    func() error
}

func (f *fakeNotifier) SendInvite(_ context.Context, _ string, _ int64) error {
	f.calls++
	return f.fn()
}

func TestNotifier_TripsAfterConsecutiveFailuresAndRecovers(t *testing.T) {
	fake := &fakeNotifier{fn: func() error { return errors.New("smtp down") }}

	settings := gobreaker.Settings{
		Name:        "test",
		MaxRequests: 1,
		Timeout:     20 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	}
	n := breaker.NewNotifier(fake, settings)
	ctx := context.Background()

	require.Error(t, n.SendInvite(ctx, "a@example.com", 1))
	require.Error(t, n.SendInvite(ctx, "a@example.com", 1))
	require.Equal(t, 2, fake.calls, "closed circuit: both calls must reach the underlying notifier")

	err := n.SendInvite(ctx, "a@example.com", 1)
	require.Error(t, err)
	require.Equal(t, 2, fake.calls, "open circuit: must short-circuit without calling the underlying notifier")

	time.Sleep(30 * time.Millisecond) // let Timeout elapse -> half-open
	fake.fn = func() error { return nil }

	require.NoError(t, n.SendInvite(ctx, "a@example.com", 1))
	require.Equal(t, 3, fake.calls, "half-open trial call must reach the underlying notifier")

	require.NoError(t, n.SendInvite(ctx, "a@example.com", 1))
	require.Equal(t, 4, fake.calls, "closed again: subsequent calls must reach the underlying notifier")
}

func TestNotifierSettings_TripsAfterThreeConsecutiveFailures(t *testing.T) {
	fake := &fakeNotifier{fn: func() error { return errors.New("smtp down") }}
	n := breaker.NewNotifier(fake, breaker.NotifierSettings())
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.Error(t, n.SendInvite(ctx, "a@example.com", 1))
	}
	require.Equal(t, 3, fake.calls)

	require.Error(t, n.SendInvite(ctx, "a@example.com", 1))
	require.Equal(t, 3, fake.calls, "4th call must be short-circuited: production settings trip at 3 consecutive failures")
}
