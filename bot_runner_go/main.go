package main

import (
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/pkg/autoconfig"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/pkg/runner"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/pkg/zerologger"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/runner/helloworld"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/runner/wxauto"
)

type Config struct {
	Logger           zerologger.Config  `yaml:"logger"`             // 日志配置
	HelloWorldRunner *helloworld.Config `yaml:"hello_world_runner"` // HelloWorldRunner配置
	WxAutoRunner     *wxauto.Config     `yaml:"wxauto_runner"`      // 微信机器人runner配置
}

func main() {
	cfg := autoconfig.MustLoadConfig[Config]()

	logger := zerologger.MustNewLogger(&cfg.Logger)
	defer logger.Close()

	runner.Run(
		*logger.Logger,
		helloworld.New(cfg.HelloWorldRunner),
		wxauto.MustNew(cfg.WxAutoRunner),
	)
}
