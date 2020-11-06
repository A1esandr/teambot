package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	//ReaderFile define for test
	ReaderFile = ioutil.ReadFile
)

type (
	App struct {
		token        string
		config       *Config
		users        map[string][]User
		teams        []Team
		auth         *Auth
		homeKeyboard tgbotapi.InlineKeyboardMarkup
	}

	Config struct {
		Welcome    string `json:"welcome"`
		AuthMsg    string `json:"auth_msg"`
		Authorized string `json:"authorized"`
		TeamsTitle string `json:"teams_button_title"`
	}

	Auth struct {
		authorized map[int]struct{}
		mu         sync.RWMutex
	}

	Team struct {
		Name  string
		Users []User
	}

	User struct {
		Name    string
		Surname string
		Data    string
	}

	Message struct {
		UserID   int
		UserName string
		Text     string
	}
)

func NewApp() *App {
	return &App{config: &Config{}, users: map[string][]User{}, auth: &Auth{authorized: map[int]struct{}{}}}
}

func (a *App) init() {
	a.token = os.Getenv("TOKEN")
	if a.token == "" {
		log.Panic("token is empty!")
	}
	err := a.config.loadConfig()
	if err != nil {
		log.Panic(err.Error())
	}
	a.loadUsers()
	a.homeKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(a.config.TeamsTitle, a.config.TeamsTitle),
		),
	)
}

func (c *Config) loadConfig() error {
	fileName := "../config/custom.json"
	data, err := ReaderFile(fileName)
	if err != nil {
		data, err = ReaderFile("../config/config.json")
		if err != nil {
			data, err = ReaderFile("config/config.json")
			if err != nil {
				return err
			}
		}
	}
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("Error to parse config %s", err)
	}
	return nil
}

func (a *App) loadUsers() {
	data, err := ReaderFile("../config/users.csv")
	if err != nil {
		data, err = ReaderFile("config/users.csv")
		if err != nil {
			log.Panic(err)
		}
	}
	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		log.Panic(err)
	}
	if len(records) == 0 {
		log.Panic("Empty users")
	}

	for _, record := range records {
		user := User{
			Name:    strings.ToLower(record[1]),
			Surname: strings.ToLower(record[0]),
			Data:    strings.ToLower(record[2]),
		}
		if _, ok := a.users[user.Surname]; !ok {
			a.users[user.Surname] = []User{}
		}
		a.users[user.Surname] = append(a.users[user.Surname], user)
	}
}

func (a *App) Start() {
	a.init()
	bot, err := tgbotapi.NewBotAPI(a.token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.CallbackQuery != nil {
			fmt.Print(update)
		}

		userMsg := &Message{
			UserID:   update.Message.From.ID,
			UserName: update.Message.From.UserName,
			Text:     update.Message.Text,
		}
		text := a.handle(userMsg)
		keyboard := a.chooseKeyboard(text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
		msg.ReplyToMessageID = update.Message.MessageID

		if len(keyboard.InlineKeyboard) > 0 {
			msg.ReplyMarkup = keyboard
		}

		_, err := bot.Send(msg)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (a *App) chooseMsg(command string) string {
	switch command {
	case "/start":
		return a.config.Welcome
	default:
		return command
	}
}

func (a *App) chooseKeyboard(text string) tgbotapi.InlineKeyboardMarkup {
	switch text {
	case a.config.Authorized:
		return a.homeKeyboard
	default:
		return tgbotapi.InlineKeyboardMarkup{}
	}
}

func (a *App) handle(msg *Message) string {
	var authorized bool
	if !a.checkAuth(msg.UserID) {
		authorized = a.authorize(msg)
		if !authorized {
			return a.config.AuthMsg
		}
		return a.config.Authorized
	}
	return a.chooseMsg(msg.Text)
}

func (a *App) checkAuth(userID int) bool {
	a.auth.mu.RLock()
	_, ok := a.auth.authorized[userID]
	a.auth.mu.RUnlock()
	return ok
}

func (a *App) authorize(msg *Message) bool {
	data := strings.Split(msg.Text, " ")
	if len(data) < 3 {
		return false
	}
	if _, ok := a.users[strings.ToLower(data[0])]; !ok {
		return false
	}
	for _, user := range a.users[strings.ToLower(data[0])] {
		if strings.ToLower(data[0]) == user.Surname &&
			strings.ToLower(data[1]) == user.Name &&
			strings.ToLower(data[2]) == user.Data {
			a.auth.mu.Lock()
			a.auth.authorized[msg.UserID] = struct{}{}
			a.auth.mu.Unlock()
			return true
		}
	}
	return false
}
