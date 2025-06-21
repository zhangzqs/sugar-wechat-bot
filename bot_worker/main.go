package main

import (
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/internal"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/autoconfig"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/runner"
)

func main() {
	cfg := autoconfig.MustLoadConfig[internal.Config]()
	r, err := internal.NewBotWorkerRunner(cfg)
	if err != nil {
		panic(err)
	}
	runner.Run(r)
}
