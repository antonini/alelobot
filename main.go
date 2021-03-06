package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/caarlos0/alelobot/internal/datastore"
	"github.com/caarlos0/alelogo"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	ds := datastore.NewRedis(os.Getenv("REDIS_URL"))
	defer ds.Close()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// without a port binded, heroku complains and eventually kills the process.
	go serve()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		update := update
		if update.Message == nil {
			continue
		}
		log.Println("Message from:", *update.Message.From)
		if update.Message.Command() == "login" {
			go login(ds, bot, update)
			continue
		}
		if update.Message.Command() == "balance" {
			go balance(ds, bot, update)
			continue
		}
		log.Println("Unknown command", update.Message.Text)
		bot.Send(tgbotapi.NewMessage(
			update.Message.Chat.ID,
			"Os únicos comandos suportados são /login e /balance",
		))
	}
}

func balance(ds datastore.Datastore, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	cpf, pwd, err := ds.Retrieve(update.Message.From.ID)
	if cpf == "" || pwd == "" || err != nil {
		bot.Send(tgbotapi.NewMessage(
			update.Message.Chat.ID,
			"Por favor, faça /login...",
		))
		return
	}
	client, err := alelogo.New(cpf, pwd)
	if err != nil {
		log.Println(err.Error())
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
		return
	}
	cards, err := client.Balance()
	if err != nil {
		log.Println(err.Error())
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
		return
	}
	for _, card := range cards {
		bot.Send(tgbotapi.NewMessage(
			update.Message.Chat.ID,
			"Saldo do cartão "+strings.TrimSpace(card.Title)+" é "+card.Balance,
		))
	}
}

func login(ds datastore.Datastore, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	parts := strings.Split(
		strings.TrimSpace(update.Message.CommandArguments()), " ",
	)
	if len(parts) != 2 {
		bot.Send(tgbotapi.NewMessage(
			update.Message.Chat.ID,
			"Para fazer login, diga\n\n/login CPF Senha",
		))
		return
	}
	cpf, pwd := parts[0], parts[1]
	if _, err := alelogo.New(cpf, pwd); err != nil {
		log.Println(err.Error())
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
		return
	}
	if err := ds.Save(update.Message.From.ID, cpf, pwd); err != nil {
		log.Println(err.Error())
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
		return
	}
	bot.Send(tgbotapi.NewMessage(
		update.Message.Chat.ID, "Sucesso, agora é só dizer /balance!",
	))
}

func serve() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
