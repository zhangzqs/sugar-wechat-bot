package natsproducer

import (
	"errors"

	"github.com/nats-io/nats.go"
)

type Config struct {
	NatsURL string `yaml:"nats_url"` // NATS 服务器地址
	Subject string `yaml:"subject"`  // 发布主题
}

func (c *Config) Validate() error {
	if c.NatsURL == "" {
		return errors.New("nats_url is required")
	}
	if c.Subject == "" {
		return errors.New("subject is required")
	}
	return nil
}

type Producer struct {
	subject string     // 发布主题
	nc      *nats.Conn // NATS 连接
}

func New(cfg *Config) (*Producer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		return nil, err
	}
	producer := &Producer{
		subject: cfg.Subject,
		nc:      nc,
	}
	return producer, nil
}

func (p *Producer) Publish(msg []byte) error {
	if p.nc == nil {
		return errors.New("NATS connection is not initialized")
	}

	err := p.nc.Publish(p.subject, msg)
	if err != nil {
		return err
	}
	return nil
}

func (p *Producer) Close() {
	if p.nc != nil {
		p.nc.Close()
		p.nc = nil
	}
}
