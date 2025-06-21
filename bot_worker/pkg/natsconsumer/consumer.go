package natsconsumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

type Context struct {
	context.Context
	WorkerID int             // 当前处理消息的工作线程ID
	Logger   *zerolog.Logger // 日志记录器，包含请求ID等上下文信息
}

type HandlerFunc func(ctx *Context, msg *nats.Msg)

type Consumer struct {
	// New 时候一次性设置所有
	logger     *zerolog.Logger
	ctx        context.Context
	cancelFunc context.CancelFunc // 用于取消消费者的上下文
	cfg        *Config

	// handler 在 Start 之前调用 SetHandler 方法设置
	handler HandlerFunc

	// Start 后构造这两个
	nc *nats.Conn
	wg *sync.WaitGroup
}

func NewConsumer(cfg *Config, logger *zerolog.Logger) *Consumer {
	logger1 := logger.With().
		Str("consumer_name", cfg.ConsumerName).
		Str("subject", cfg.Subject).
		Logger()
	ctx, cancelFunc := context.WithCancel(logger1.WithContext(context.Background()))
	return &Consumer{
		logger:     &logger1,
		ctx:        ctx,
		cfg:        cfg,
		cancelFunc: cancelFunc,
	}
}

func (c *Consumer) Start() (err error) {
	if c.nc != nil {
		c.logger.Warn().Msg("Consumer is already started, skipping Start")
		return nil
	}

	// 连接到NATS服务器
	nc, err := nats.Connect(c.cfg.NatsURL)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to connect to NATS server")
		return
	}
	c.nc = nc

	// 创建JetStream上下文
	js, err := nc.JetStream()
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to create JetStream context")
		return
	}

	var wg sync.WaitGroup
	for i := range c.cfg.Concurrency {
		wg.Add(1)
		c.logger.Info().Msgf("Starting consumer worker %d", i)
		go func(i int) {
			defer wg.Done()
			logger := c.logger.With().Int("worker_id", i).Logger()
			ctx := logger.WithContext(c.ctx)
			c.consumerWorker(ctx, js, i)
		}(i)
	}
	c.wg = &wg

	c.logger.Info().Msgf("Started %d consumer workers for subject %s", c.cfg.Concurrency, c.cfg.Subject)
	return
}

func (c *Consumer) SetHandler(handler HandlerFunc) {
	c.handler = handler
	c.logger.Info().Msg("Handler function set for consumer")
}

func (c *Consumer) Close() {
	if c.nc == nil {
		c.logger.Warn().Msg("Consumer is not started, nothing to close")
		return
	}

	c.logger.Info().Msg("Closing consumer")
	c.cancelFunc() // 取消上下文，通知所有工作线程停止
	// 等待所有消费者工作完成
	c.wg.Wait()
	c.logger.Info().Msg("All consumer workers have stopped")

	// 关闭NATS连接
	if err := c.nc.Drain(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to drain NATS connection")
		c.nc.Close() // 确保连接被关闭
	}

	c.logger.Info().Msg("Consumer closed successfully")

	// 清理
	c.nc = nil
	c.wg = nil
}

// isConnectionError 检查错误是否是连接相关的错误
func isConnectionError(err error) bool {
	return errors.Is(err, nats.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrNoResponders) ||
		errors.Is(err, nats.ErrBadSubscription)
}

func (c *Consumer) consumerWorker(ctx context.Context, js nats.JetStreamContext, workerID int) {
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
				logger.Debug().Msg("No messages received, continuing")
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
			if c.handler == nil {
				logger.Warn().Msg("No handler set, message will be requeued")
				if err := msg.Nak(); err != nil {
					logger.Error().Err(err).Msg("Failed to Nak message")
				}
				continue
			}

			func() {
				defer func() {
					if r := recover(); r != nil { // 如果recover捕获到panic，则记录错误并Nak消息
						logger.Error().Interface("panic", r).Msgf("Recovered from panic")
						if err := msg.Nak(); err != nil {
							logger.Error().Err(err).Msg("Failed to Nak message after panic")
						}
					}
				}()
				c.handler(&Context{
					Context:  logger.WithContext(ctx),
					WorkerID: workerID,
					Logger:   logger,
				}, msg)
			}()
		}
	}
}
