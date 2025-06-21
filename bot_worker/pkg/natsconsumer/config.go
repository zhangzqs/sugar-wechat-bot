package natsconsumer

import (
	"errors"
	"time"
)

type Config struct {
	NatsURL      string        `yaml:"nats_url"`
	Concurrency  int           `yaml:"concurrency"`
	Subject      string        `yaml:"subject"`
	ConsumerName string        `yaml:"consumer_name"`
	PullMaxWait  time.Duration `yaml:"pull_max_wait"`
}

func (c *Config) Validate() error {
	if c.Concurrency <= 0 {
		c.Concurrency = 1
	}
	if c.PullMaxWait <= 0 {
		c.PullMaxWait = 10 * time.Second
	}

	if c.NatsURL == "" {
		return errors.New("nats_url is required")
	}
	if c.Subject == "" {
		return errors.New("subject is required")
	}
	if c.ConsumerName == "" {
		return errors.New("consumer_name is required")
	}
	if c.Concurrency <= 0 {
		return errors.New("concurrency must be greater than 0")
	}
	if c.PullMaxWait <= 0 {
		return errors.New("pull_max_wait must be greater than 0")
	}
	return nil
}
