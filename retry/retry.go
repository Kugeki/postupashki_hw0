package retry

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Config struct {
	MaxAttempts int

	SleepCoefficient float64
	StartSleep       time.Duration
}

func NewFunc(conf Config, fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		attempts := 0
		sleepTime := time.Duration(0)

		for {
			err := fn(ctx)
			if err == nil {
				return nil
			}
			log.Printf("error in retry: %v. attempt: %v\n", err, attempts)

			attempts++

			switch {
			case attempts == 1:
				sleepTime = conf.StartSleep
			case attempts >= conf.MaxAttempts:
				return fmt.Errorf("attempts is exceeded: curr=%v max=%v", attempts, conf.MaxAttempts)
			default:
				sleepTime = time.Duration(conf.SleepCoefficient * float64(sleepTime))
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("context: %v", ctx.Err())
			case <-time.After(sleepTime):
			}
		}
	}
}
