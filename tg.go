package main

import (
	"fmt"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
)

const TgChannel = -1001704285488

var tgBot TgBot

type TgBot struct {
	*tg.BotAPI
}

func (bot *TgBot) init() {
	bot.BotAPI, _ = tg.NewBotAPI(os.Getenv("tg.token"))
}

func (bot *TgBot) newOrderOpened(pair string, price, quantity, stopLossPrice float64) {
	msg := tg.NewMessage(TgChannel, fmt.Sprintf("%s%s%s%s%s",
		listFormat("Операция", "BUY"),
		listFormat("Пара", pair),
		listFormat("Цена", f2s(price)),
		listFormat("Кол-во", f2s(quantity)),
		listFormat("SL", f2s(stopLossPrice)),
	))
	msg.ParseMode = tg.ModeHTML

	message, err := tgBot.Send(msg)
	fmt.Println(message, err)
}

func (bot *TgBot) orderClosed(pair string, price, quantity float64) {
	msg := tg.NewMessage(TgChannel, fmt.Sprintf("%s%s%s%s",
		listFormat("Операция", "SELL"),
		listFormat("Пара", pair),
		listFormat("Цена", f2s(price)),
		listFormat("Кол-во", f2s(quantity)),
	))
	msg.ParseMode = tg.ModeHTML

	message, err := tgBot.Send(msg)
	fmt.Println(message, err)
}

func listFormat(key, value string) string {
	return fmt.Sprintf("<b>%s</b>: %s\n", key, value)
}
