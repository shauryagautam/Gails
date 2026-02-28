package queue

import "context"

type Job interface {
	Perform(ctx context.Context) error
}
