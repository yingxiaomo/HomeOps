package main

import (
	"github.com/yingxiaomo/HomeOps/config"
	"github.com/yingxiaomo/HomeOps/pkg/bot"
)

func main() {
	config.LoadConfig()
	b := bot.NewBot()
	b.Start()
}
