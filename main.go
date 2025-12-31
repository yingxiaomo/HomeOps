package main

import (
	"github.com/yingxiaomo/homeops/config"
	"github.com/yingxiaomo/homeops/pkg/bot"
)

func main() {
	config.LoadConfig()
	b := bot.NewBot()
	b.Start()
}
