package main

import (
	"go_bot/config"
	"go_bot/pkg/bot"
)

func main() {
	config.LoadConfig()
	b := bot.NewBot()
	b.Start()
}
