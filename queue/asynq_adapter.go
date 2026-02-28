package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/shaurya/gails/config"
)

type Queue interface {
	Enqueue(job Job) error
	EnqueueIn(delay time.Duration, job Job) error
	EnqueueAt(t time.Time, job Job) error
	EnqueueWithQueue(qName string, job Job) error
}

type AsynqAdapter struct {
	Client *asynq.Client
	Config config.RedisConfig
}

func NewAsynqAdapter(cfg config.RedisConfig) *AsynqAdapter {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.URL})
	return &AsynqAdapter{Client: client, Config: cfg}
}

func (a *AsynqAdapter) Enqueue(job Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	task := asynq.NewTask(fmt.Sprintf("%T", job), payload)
	_, err = a.Client.Enqueue(task)
	return err
}

func (a *AsynqAdapter) EnqueueIn(delay time.Duration, job Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	task := asynq.NewTask(fmt.Sprintf("%T", job), payload)
	_, err = a.Client.Enqueue(task, asynq.ProcessIn(delay))
	return err
}

func (a *AsynqAdapter) EnqueueAt(t time.Time, job Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	task := asynq.NewTask(fmt.Sprintf("%T", job), payload)
	_, err = a.Client.Enqueue(task, asynq.ProcessAt(t))
	return err
}

func (a *AsynqAdapter) EnqueueWithQueue(qName string, job Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	task := asynq.NewTask(fmt.Sprintf("%T", job), payload)
	_, err = a.Client.Enqueue(task, asynq.Queue(qName))
	return err
}
