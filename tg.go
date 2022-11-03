package main

import (
	"fmt"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
)

var tgBot TgBot

type TgBot struct {
	*tg.BotAPI
	Channel int64
}

func (bot *TgBot) init() {
	bot.BotAPI, _ = tg.NewBotAPI(os.Getenv("tg.token"))
	bot.Channel = s2i(os.Getenv("tg.channel"))
	bot.Debug = false
}

func (bot *TgBot) newOrderOpened(pair string, price, stopLossPrice float64, screen string) {
	msg := tg.NewPhoto(tgBot.Channel, tg.FilePath(screen))
	msg.Caption = fmt.Sprintf("%s%s%s%s",
		listFormat("Операция", "#BUY"),
		listFormat("Пара", "#"+pair),
		listFormat("Цена", f2s(price)),
		listFormat("SL", f2s(stopLossPrice)),
	)
	msg.ParseMode = tg.ModeHTML

	_, _ = tgBot.Send(msg)
}

func (bot *TgBot) orderClosed(pair string, price float64) {
	msg := tg.NewMessage(bot.Channel, fmt.Sprintf("%s%s%s",
		listFormat("Операция", "#SELL"),
		listFormat("Пара", "#"+pair),
		listFormat("Цена", f2s(price)),
	))
	msg.ParseMode = tg.ModeHTML

	message, err := tgBot.Send(msg)
	fmt.Println(message, err)
}

func listFormat(key, value string) string {
	return fmt.Sprintf("<b>%s</b>: %s\n", key, value)
}
