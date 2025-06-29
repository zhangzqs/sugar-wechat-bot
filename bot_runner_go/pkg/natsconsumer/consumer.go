package natsconsumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

type ctxKeyWorkerID struct{}

func MustGetWorkerID(ctx context.Context) int {
	if v, ok := ctx.Value(ctxKeyWorkerID{}).(int); ok {
		return v
	}
	panic("worker ID not found in context")
}

type HandleResult string

const (
	HandleResultAck  HandleResult = "ACK"  // 处理成功，消息已确认
	HandleResultNak  HandleResult = "NAK"  // 处理失败，消息重新入队
	HandleResultTerm HandleResult = "TERM" // 处理失败，消息丢弃
)

type HandlerFunc func(ctx context.Context, msg *nats.Msg) HandleResult

type Consumer struct {
	cfg *Config
}

func New(cfg *Config) *Consumer {
	return &Consumer{
		cfg: cfg,
	}
}

func (c *Consumer) Run(ctx context.Context, handler HandlerFunc) (err error) {
	logger := zerolog.Ctx(ctx).With().
		Str("consumer_name", c.cfg.ConsumerName).
		Str("subject", c.cfg.Subject).
		Logger()

	// 连接到NATS服务器
	nc, err := nats.Connect(c.cfg.NatsURL)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to NATS server")
		return
	}
	defer func() {
		if err := nc.Drain(); err != nil {
			logger.Error().Err(err).Msg("Failed to drain NATS connection")
		}
	}()
	// 创建JetStream上下文
	js, err := nc.JetStream()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create JetStream context")
		return
	}

	var wg sync.WaitGroup
	for i := range c.cfg.Concurrency {
		wg.Add(1)
		logger.Info().Msgf("Starting consumer worker %d", i)
		go func(i int) {
			defer wg.Done()
			logger := logger.With().Int("worker_id", i).Logger()
			ctx := logger.WithContext(ctx)
			c.consumerWorker(ctx, js, i, handler)
		}(i)
	}
	logger.Info().Msgf("Started %d consumer workers for subject %s", c.cfg.Concurrency, c.cfg.Subject)
	wg.Wait()
	return
}

// isConnectionError 检查错误是否是连接相关的错误
func isConnectionError(err error) bool {
	return errors.Is(err, nats.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrNoResponders) ||
		errors.Is(err, nats.ErrBadSubscription)
}

func (c *Consumer) consumerWorker(ctx context.Context, js nats.JetStreamContext, workerID int, handler HandlerFunc) {
	logger := zerolog.Ctx(ctx)

	// 创建订阅者
	var sub *nats.Subscription
	defer func() {
		if sub != nil {
			if err := sub.Unsubscribe(); err != nil {
				logger.Error().Err(err).Msg("Failed to unsubscribe from subject")
			} else {
				logger.Info().Msg("Unsubscribed from subject")
			}
		}
	}()

	// 开始消费
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Consumer worker stopping")
			return
		default:
		}

		if sub != nil && !sub.IsValid() { // 如果订阅者无效，则重新创建
			logger.Warn().Msg("Subscription is invalid, recreating")
			sub = nil
		}

		if sub == nil { // 如果订阅者不存在，则创建一个新的消息订阅者
			// 订阅消息
			var err error
			sub, err = js.PullSubscribe(c.cfg.Subject, c.cfg.ConsumerName, nats.ManualAck())
			if err != nil {
				logger.Error().Err(err).Msg("Failed to subscribe to subject")
				time.Sleep(2 * time.Second)
				continue
			}
			logger.Info().Msgf("Subscribed to subject %s with consumer %s", c.cfg.Subject, c.cfg.ConsumerName)
		}

		// 拉取消息
		msgs, err := sub.Fetch(1, nats.MaxWait(c.cfg.PullMaxWait))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			logger.Error().Err(err).Msg("Failed to fetch messages")

			if isConnectionError(err) {
				logger.Warn().Err(err).Msg("Bad subscription, will recreate")
				sub = nil // 重置订阅者
			}
			continue
		}

		// 消息处理
		for _, msg := range msgs {
			if handler == nil {
				logger.Warn().Msg("No handler set, message will be requeued")
				if err := msg.Nak(); err != nil {
					logger.Error().Err(err).Msg("Failed to Nak message")
				}
				continue
			}

			func() {
				ctx = context.WithValue(ctx, ctxKeyWorkerID{}, workerID)
				switch result := handler(ctx, msg); result {
				case HandleResultAck:
					if err := msg.Ack(); err != nil {
						logger.Error().Err(err).Msg("Failed to Ack message")
					}
				case HandleResultNak:
					if err := msg.Nak(); err != nil {
						logger.Error().Err(err).Msg("Failed to Nak message")
					}
				case HandleResultTerm:
					if err := msg.Term(); err != nil {
						logger.Error().Err(err).Msg("Failed to Term message")
					}
				}
			}()
		}
	}
}
