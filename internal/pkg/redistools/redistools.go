package redistools

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func Connect(ctx context.Context, rdb *redis.Client) error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		defaultDelay := time.Second

		for {
			if err := rdb.Ping(ctx).Err(); err != nil {
				time.Sleep(defaultDelay)
				defaultDelay += time.Second

				if defaultDelay > time.Second*10 {
					errCh <- fmt.Errorf("cannot ping redis db error: %w", err)

					return
				}

				continue
			}

			break
		}
	}()
	select {
	case <-ctx.Done():
		return fmt.Errorf("context error: %w", ctx.Err())
	case err := <-errCh:
		return err
	}
}
