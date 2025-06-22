package main

import (
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/autoconfig"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/runner"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/runner/helloworld"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/runner/wxauto"
)

type Config struct {
	HelloWorldRunner *helloworld.Config `yaml:"hello_world_runner"` // HelloWorldRunner配置
	WxAutoRunner     *wxauto.Config     `yaml:"wxauto_runner"`      // 微信机器人runner配置
}

func main() {
	var runners []runner.Runner
	cfg := autoconfig.MustLoadConfig[Config]()

	if cfg.HelloWorldRunner != nil {
		helloWorldRunner, err := helloworld.NewHelloWorldRunner(cfg.HelloWorldRunner)
		if err != nil {
			panic(err)
		}
		runners = append(runners, helloWorldRunner)
	}
	if cfg.WxAutoRunner != nil {
		wxautoRunner, err := wxauto.NewWxAutoRunner(cfg.WxAutoRunner)
		if err != nil {
			panic(err)
		}
		runners = append(runners, wxautoRunner)
	}

	runner.Run(runners...)
}
