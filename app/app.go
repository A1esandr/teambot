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
		token  string
		config *Config
		users  map[string][]User

		auth          *Auth
		homeKeyboard  tgbotapi.InlineKeyboardMarkup
		teamsKeyboard tgbotapi.InlineKeyboardMarkup
		teams         map[string]Team
	}

	Config struct {
		Welcome    string `json:"welcome"`
		AuthMsg    string `json:"auth_msg"`
		Authorized string `json:"authorized"`
		TeamsTitle string `json:"teams_button_title"`
		Teams      []Team `json:"teams"`
	}

	Auth struct {
		authorized map[int]struct{}
		mu         sync.RWMutex
	}

	Team struct {
		Name  string `json:"name"`
		Users []User `json:"members"`
	}

	User struct {
		Name    string `json:"name"`
		Surname string `json:"surname"`
		Data    string
	}

	Message struct {
		ChatID   int64
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
	teams := []tgbotapi.InlineKeyboardButton{}
	tmap := map[string]Team{}
	for _, team := range a.config.Teams {
		teams = append(teams, tgbotapi.NewInlineKeyboardButtonData(team.Name, team.Name))
		tmap[team.Name] = team
	}
	a.teamsKeyboard = tgbotapi.NewInlineKeyboardMarkup(teams)
	a.teams = tmap
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
	if err != nil {
		fmt.Println(err)
	}

	a.Handle(bot, updates)
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

func (a *App) Handle(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	for update := range updates {

		if update.CallbackQuery != nil {
			fmt.Println(update)
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
			if update.CallbackQuery.Data == a.config.TeamsTitle {
				msg.ReplyMarkup = a.teamsKeyboard
			}
			if team, ok := a.teams[update.CallbackQuery.Data]; ok {
				var sb strings.Builder
				sb.WriteString(team.Name)
				sb.WriteString("\n\n")
				for _, user := range team.Users {
					sb.WriteString(user.Surname)
					sb.WriteString(" ")
					sb.WriteString(user.Name)
					sb.WriteString("\n")
				}

				msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, sb.String())
			}
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			_, err = bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			if err != nil {
				log.Println(err)
			}
			continue
		}

		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		userMsg := &Message{}
		if update.Message != nil {
			userMsg.UserID = update.Message.From.ID
			userMsg.UserName = update.Message.From.UserName
			userMsg.Text = update.Message.Text
			userMsg.ChatID = update.Message.Chat.ID
		}

		guest, text := a.checkAuthorized(userMsg)
		if guest && userMsg.ChatID > 0 {
			msg := tgbotapi.NewMessage(userMsg.ChatID, text)
			if text == a.config.Authorized {
				msg.ReplyMarkup = a.homeKeyboard
			}
			_, err := bot.Send(msg)
			if err != nil {
				fmt.Println(err)
			}
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		text = a.chooseMsg(userMsg.Text)
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

func (a *App) checkAuthorized(msg *Message) (bool, string) {
	if !a.checkAuth(msg.UserID) {
		if !a.authorize(msg) {
			return true, a.config.AuthMsg
		}
		return true, a.config.Authorized
	}
	return false, ""
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
