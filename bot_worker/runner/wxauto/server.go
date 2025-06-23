package wxauto

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/nats-io/nats.go"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/natsconsumer"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/natsproducer"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/reactagent"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/runner"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/zerologger"
)

type Config struct {
	Logger                 zerologger.Config   `yaml:"logger"`                    // 日志配置
	Producer               natsproducer.Config `yaml:"producer"`                  // NATS 生产者配置
	Consumer               natsconsumer.Config `yaml:"consumer"`                  // NATS 消费者配置
	ReactAgent             reactagent.Config   `yaml:"react_agent"`               // React Agent 配置
	UserMessageTemplate    string              `yaml:"user_message_template"`     // 用户消息模板
	UserMessageReplyFilter string              `yaml:"user_message_reply_filter"` // 用户消息回复过滤器，使用 expr 语言编写的过滤规则
}

var _ runner.Runner = (*WxAutoRunner)(nil)

type WxAutoRunner struct {
	logger              *zerologger.Logger
	producer            *natsproducer.Producer
	consumer            *natsconsumer.Consumer
	reactAgent          *reactagent.ReactAgent
	userMessageTemplate *template.Template // 用户消息模板
	msgFilter           *vm.Program        // 消息过滤器，使用 expr 语言编写的过滤规则
}

func NewWxAutoRunner(cfg *Config) (*WxAutoRunner, error) {
	if err := cfg.Logger.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.Consumer.Validate(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	logger, err := zerologger.NewLogger(&cfg.Logger)
	if err != nil {
		return nil, err
	}
	ctx = logger.WithContext(ctx)

	producer, err := natsproducer.NewProducer(&cfg.Producer)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create NATS producer")
		return nil, err
	}

	consumer := natsconsumer.NewConsumer(ctx, &cfg.Consumer)

	reactAgent, err := reactagent.New(ctx, &cfg.ReactAgent)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create ReactAgent")
		return nil, err
	}

	tpl, err := template.New("").Parse(cfg.UserMessageTemplate)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse template")
		return nil, err
	}

	// 编译消息过滤器
	msgFilter, err := expr.Compile(
		cfg.UserMessageReplyFilter,
		expr.Env(ReceivedMessage{}),
		expr.AsBool(),
	)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to compile message filter")
		return nil, err
	}
	return &WxAutoRunner{
		logger:              logger,
		consumer:            consumer,
		producer:            producer,
		reactAgent:          reactAgent,
		userMessageTemplate: tpl,
		msgFilter:           msgFilter,
	}, nil
}

func (b *WxAutoRunner) Close() error {
	if b.consumer != nil {
		b.consumer.Close()
		b.consumer = nil
	}
	if b.logger != nil {
		if err := b.logger.Close(); err != nil {
			if !strings.Contains(err.Error(), "already closed") {
				panic(err)
			}
		}
		b.logger = nil
	}
	return nil
}

func (b *WxAutoRunner) Name() string {
	return "WxAutoRunner"
}

func (b *WxAutoRunner) Run() error {
	b.logger.Info().Msg("BotWorkerRunner is starting")
	defer b.logger.Info().Msg("BotWorkerRunner has stopped")

	// 启动消费者
	if err := b.consumer.Start(); err != nil {
		b.logger.Error().Err(err).Msg("Failed to start NATS consumer")
		return err
	}
	b.consumer.SetHandler(b.handleMessage)
	return nil
}

func (b *WxAutoRunner) handleMessage(ctx *natsconsumer.Context, natsMsg *nats.Msg) natsconsumer.HandleResult {
	// 处理消息的逻辑
	b.logger.Info().Str("subject", natsMsg.Subject).Msg("Received message")
	var msg ReceivedMessage
	if err := json.Unmarshal(natsMsg.Data, &msg); err != nil {
		b.logger.Error().Err(err).Msg("Failed to unmarshal message")
		return natsconsumer.HandleResultTerm
	}
	b.logger.Info().Any("wxauto_message", msg).Msg("Processed wxauto message")

	if msg.Attr != MessageAttrFriend {
		b.logger.Warn().Str("attr", string(msg.Attr)).Msg("Unsupported message attribute, skipping")
		return natsconsumer.HandleResultTerm
	}
	// if msg.Type != MessageTypeText {
	// 	b.logger.Warn().Str("type", string(msg.Type)).Msg("Unsupported message type, skipping")
	// 	return natsconsumer.HandleResultTerm
	// }

	// 消息过滤器
	ret, err := expr.Run(b.msgFilter, msg)
	if err != nil {
		b.logger.Error().Err(err).Msg("Failed to run message filter")
		return natsconsumer.HandleResultTerm
	}
	if ret == nil || !ret.(bool) {
		b.logger.Info().Msg("Message does not match filter, skipping")
		return natsconsumer.HandleResultAck
	}

	// 执行用户消息模板
	buf := bytes.Buffer{}
	if err := b.userMessageTemplate.Execute(&buf, msg); err != nil {
		b.logger.Error().Err(err).Msg("Failed to execute template")
		return natsconsumer.HandleResultTerm
	}
	answer, err := b.reactAgent.Question(ctx.Context, buf.String())
	if err != nil {
		b.logger.Error().Err(err).Msg("Failed to get answer from ReactAgent")
		return natsconsumer.HandleResultTerm
	}
	// 将回答转换为 JSON
	answerData, _ := json.Marshal(SendMessage{
		Content:      answer,
		ReplyToMsgID: msg.ID,               // 回复原消息
		SendToChat:   msg.Sender,           // 回复给发送者
		At:           []string{msg.Sender}, // 在群聊中 @ 发送者
		Exact:        true,                 // 精确匹配发送者名称
	})

	// 发送回答
	if err := b.producer.Publish(answerData); err != nil {
		b.logger.Error().Err(err).Msg("Failed to publish message")
		return natsconsumer.HandleResultNak
	}
	return natsconsumer.HandleResultAck
}
