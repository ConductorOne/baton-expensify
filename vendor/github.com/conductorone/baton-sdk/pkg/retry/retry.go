package retry

import (
	"context"
	"fmt"
	"math"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var tracer = otel.Tracer("baton-sdk/retry")

type Retryer struct {
	attempts     uint
	maxAttempts  uint
	initialDelay time.Duration
	maxDelay     time.Duration
}

type RetryConfig struct {
	MaxAttempts  uint          // 0 means no limit (which is also the default).
	InitialDelay time.Duration // Default is 1 second.
	MaxDelay     time.Duration // Default is 60 seconds. 0 means no limit.
}

func NewRetryer(ctx context.Context, config RetryConfig) *Retryer {
	r := &Retryer{
		attempts:     0,
		maxAttempts:  config.MaxAttempts,
		initialDelay: config.InitialDelay,
		maxDelay:     config.MaxDelay,
	}
	if r.initialDelay == 0 {
		r.initialDelay = time.Second
	}
	if r.maxDelay == 0 {
		r.maxDelay = 60 * time.Second
	}
	return r
}

func (r *Retryer) ShouldWaitAndRetry(ctx context.Context, err error) bool {
	ctx, span := tracer.Start(ctx, "retry.ShouldWaitAndRetry")
	defer span.End()

	if err == nil {
		r.attempts = 0
		return true
	}

	l := ctxzap.Extract(ctx)
	statusCode := status.Code(err)
	fmt.Printf("RETRY CHECK: error=%v, status=%s, attempts=%d\n", err, statusCode.String(), r.attempts)

	if statusCode != codes.Unavailable && statusCode != codes.DeadlineExceeded {
		fmt.Printf("RETRY DECISION: NOT retryable (status %s not in [Unavailable, DeadlineExceeded])\n", statusCode.String())
		return false
	}

	r.attempts++

	if r.maxAttempts > 0 && r.attempts > r.maxAttempts {
		fmt.Printf("RETRY DECISION: NOT retrying (max attempts %d reached)\n", r.maxAttempts)
		l.Warn("max attempts reached", zap.Error(err), zap.Uint("max_attempts", r.maxAttempts))
		return false
	}

	// use linear backoff by default
	var wait time.Duration
	if r.attempts > math.MaxInt64 {
		wait = r.maxDelay
	} else {
		wait = time.Duration(int64(r.attempts)) * r.initialDelay
	}

	// If error contains rate limit data, use that instead
	if st, ok := status.FromError(err); ok {
		details := st.Details()
		for _, detail := range details {
			if rlData, ok := detail.(*v2.RateLimitDescription); ok {
				waitResetAt := time.Until(rlData.ResetAt.AsTime())
				if waitResetAt <= 0 {
					continue
				}
				duration := time.Duration(rlData.Limit)
				if duration <= 0 {
					continue
				}
				waitResetAt /= duration
				// Round up to the nearest second to make sure we don't hit the rate limit again
				waitResetAt = time.Duration(math.Ceil(waitResetAt.Seconds())) * time.Second
				if waitResetAt > 0 {
					wait = waitResetAt
					break
				}
			}
		}
	}

	if wait > r.maxDelay {
		wait = r.maxDelay
	}

	fmt.Printf("RETRY DECISION: WILL retry after %v (attempt %d)\n", wait, r.attempts)
	l.Warn("retrying operation", zap.Error(err), zap.Duration("wait", wait))

	for {
		select {
		case <-time.After(wait):
			return true
		case <-ctx.Done():
			return false
		}
	}
}
