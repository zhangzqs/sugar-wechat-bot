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
	"github.com/rs/zerolog"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/pkg/natsconsumer"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/pkg/natsproducer"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/pkg/reactagent"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/pkg/runner"
)

type Config struct {
	Producer               natsproducer.Config `yaml:"producer"`                  // NATS 生产者配置
	Consumer               natsconsumer.Config `yaml:"consumer"`                  // NATS 消费者配置
	ReactAgent             reactagent.Config   `yaml:"react_agent"`               // React Agent 配置
	UserMessageTemplate    string              `yaml:"user_message_template"`     // 用户消息模板
	UserMessageReplyFilter string              `yaml:"user_message_reply_filter"` // 用户消息回复过滤器，使用 expr 语言编写的过滤规则
}

var _ runner.Runner = (*WxAutoRunner)(nil)

type WxAutoRunner struct {
	cfg                 *Config            // 配置
	userMessageTemplate *template.Template // 用户消息模板
	msgFilter           *vm.Program        // 消息过滤器，使用 expr 语言编写的过滤规则
}

func MustNew(cfg *Config) *WxAutoRunner {
	if err := cfg.Consumer.Validate(); err != nil {
		panic(err)
	}

	tpl, err := template.New("").Parse(cfg.UserMessageTemplate)
	if err != nil {
		panic(err)
	}

	// 编译消息过滤器
	msgFilter, err := expr.Compile(
		cfg.UserMessageReplyFilter,
		expr.Env(ReceivedMessage{}),
		expr.AsBool(),
	)
	if err != nil {
		panic(err)
	}
	return &WxAutoRunner{
		userMessageTemplate: tpl,
		msgFilter:           msgFilter,
		cfg:                 cfg,
	}
}
func (b *WxAutoRunner) Name() string {
	return "WxAutoRunner"
}

func (b *WxAutoRunner) Run(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)

	logger.Info().Msg("BotWorkerRunner is starting")
	defer logger.Info().Msg("BotWorkerRunner has stopped")

	producer, err := natsproducer.New(&b.cfg.Producer)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create NATS producer")
		return err
	}

	reactAgent, err := reactagent.New(ctx, &b.cfg.ReactAgent)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create ReactAgent")
		return err
	}

	consumer := natsconsumer.New(&b.cfg.Consumer)
	consumer.Run(ctx, func(ctx context.Context, msg *nats.Msg) natsconsumer.HandleResult {
		return b.handleMessage(&Context{
			Context:    ctx,
			reactAgent: reactAgent,
			producer:   producer,
		}, msg)
	})
	return nil
}

type Context struct {
	context.Context
	reactAgent *reactagent.ReactAgent // React Agent 实例
	producer   *natsproducer.Producer // NATS 生产者实例
}

func (b *WxAutoRunner) handleMessage(ctx *Context, natsMsg *nats.Msg) natsconsumer.HandleResult {
	logger := zerolog.Ctx(ctx)
	// 处理消息的逻辑
	logger.Info().Str("subject", natsMsg.Subject).Msg("Received message")
	var msg ReceivedMessage
	if err := json.Unmarshal(natsMsg.Data, &msg); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal message")
		return natsconsumer.HandleResultTerm
	}
	logger.Info().Any("wxauto_message", msg).Msg("Processed wxauto message")

	if msg.Attr != MessageAttrFriend {
		logger.Warn().Str("attr", string(msg.Attr)).Msg("Unsupported message attribute, skipping")
		return natsconsumer.HandleResultTerm
	}
	// if msg.Type != MessageTypeText {
	// 	logger.Warn().Str("type", string(msg.Type)).Msg("Unsupported message type, skipping")
	// 	return natsconsumer.HandleResultTerm
	// }

	// 消息过滤器
	ret, err := expr.Run(b.msgFilter, msg)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to run message filter")
		return natsconsumer.HandleResultTerm
	}
	if ret == nil || !ret.(bool) {
		logger.Info().Msg("Message does not match filter, skipping")
		return natsconsumer.HandleResultAck
	}

	// 执行用户消息模板
	buf := bytes.Buffer{}
	if err := b.userMessageTemplate.Execute(&buf, msg); err != nil {
		logger.Error().Err(err).Msg("Failed to execute template")
		return natsconsumer.HandleResultTerm
	}
	answer, err := ctx.reactAgent.Question(ctx, buf.String())
	if err != nil {
		if strings.Contains(err.Error(), "exceeded max steps") {
			logger.Warn().Err(err).Msg("ReactAgent exceeded max steps, skipping message")
			answer = "抱歉，我无法处理这个请求，当前问题过于复杂"
		} else {
			logger.Error().Err(err).Msg("Failed to get answer from ReactAgent")
			answer = "破防，遇到了一些无法处理的错误: " + err.Error()
		}
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
	if err := ctx.producer.Publish(answerData); err != nil {
		logger.Error().Err(err).Msg("Failed to publish message")
		return natsconsumer.HandleResultNak
	}
	return natsconsumer.HandleResultAck
}
