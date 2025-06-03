package bot

import "context"

// Limiter is a typed channel for limiting concurrency.
type Limiter chan struct{}

// Release frees up a slot in the limiter.
func (lim Limiter) Release() { <-lim }

// Acquire attempts to occupy a limiter slot or returns if the context is cancelled.
func (lim Limiter) Acquire(ctx context.Context) error {
	if cap(lim) == 0 {
		return ctx.Err() // no limit set, so we can proceed without blocking
	}

	select {
	case lim <- struct{}{}: // acquire a limiter slot
		if err := ctx.Err(); err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
