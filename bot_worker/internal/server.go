package internal

import (
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/natsconsumer"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/runner"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/zerologger"
)

type Config struct {
	Logger   zerologger.Config   `yaml:"logger"`   // 日志配置
	Consumer natsconsumer.Config `yaml:"consumer"` // NATS 消费者配置
}

var _ runner.Runner = (*BotWorkerRunner)(nil)

type BotWorkerRunner struct {
	logger   *zerologger.Logger
	consumer *natsconsumer.Consumer
}

func NewBotWorkerRunner(cfg *Config) (*BotWorkerRunner, error) {
	if err := cfg.Logger.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.Consumer.Validate(); err != nil {
		return nil, err
	}

	logger, err := zerologger.NewLogger(&cfg.Logger)
	if err != nil {
		return nil, err
	}
	zerolog.DefaultContextLogger = logger.Logger

	consumer := natsconsumer.NewConsumer(&cfg.Consumer, logger.Logger)
	if err := consumer.Start(); err != nil {
		return nil, err
	}

	return &BotWorkerRunner{
		logger:   logger,
		consumer: consumer,
	}, nil
}

func (b *BotWorkerRunner) Close() error {
	if b.consumer != nil {
		b.consumer.Close()
		b.consumer = nil
	}
	if b.logger != nil {
		if err := b.logger.Close(); err != nil {
			return err
		}
		b.logger = nil
	}
	return nil
}

func (b *BotWorkerRunner) Name() string {
	return "BotWorkerRunner"
}

func (b *BotWorkerRunner) Run() error {
	b.logger.Info().Msg("BotWorkerRunner is starting")
	defer b.logger.Info().Msg("BotWorkerRunner has stopped")

	// 启动消费者
	if err := b.consumer.Start(); err != nil {
		return err
	}
	b.consumer.SetHandler(b.handleMessage)
	return nil
}

func (b *BotWorkerRunner) handleMessage(ctx *natsconsumer.Context, msg *nats.Msg) {
	// 处理消息的逻辑
	b.logger.Info().
		Str("subject", msg.Subject).
		Str("data", string(msg.Data)).
		Msg("Received message")
	msg.Ack() // 确认消息
}
