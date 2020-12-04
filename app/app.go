package app

import (
	"encoding/csv"
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
		auth         *Auth
		homeKeyboard tgbotapi.InlineKeyboardMarkup
		keyboards    map[string]tgbotapi.InlineKeyboardMarkup
		teams        map[string]Team
		pages        map[string]string
	}

	Auth struct {
		authorized map[int]struct{}
		mu         sync.RWMutex
	}

	Team struct {
		Name  string   `json:"name"`
		Rows  []string `json:"rows"`
		Users []User   `json:"members"`
	}

	User struct {
		Name    string `json:"name"`
		Surname string `json:"surname"`
		Skills  string `json:"skills"`
		Link    string `json:"link"`
		Phone   string `json:"phone"`
		Data    string
	}

	EventsGroup struct {
		Title  string  `json:"title"`
		Rows   []Item  `json:"rows"`
		Events []Event `json:"items"`
	}

	Event struct {
		Title string `json:"title"`
		Info  string `json:"info"`
		Date  string `json:"date"`
		Links []Item `json:"links"`
	}

	Record struct {
		Title string   `json:"title"`
		Rows  []string `json:"rows"`
	}

	Item struct {
		Title string `json:"title"`
		Value string `json:"value"`
	}

	Sprint struct {
		Date string `json:"date"`
		Goal string `json:"goal"`
		Rows []Item `json:"rows"`
	}

	Community struct {
		Name    string `json:"name"`
		Mentors []User `json:"mentors"`
		Rows    []Item `json:"rows"`
	}

	Message struct {
		ChatID   int64
		UserID   int
		UserName string
		Text     string
	}
)

func NewApp() *App {
	return &App{
		config:    &Config{},
		users:     map[string][]User{},
		auth:      &Auth{authorized: map[int]struct{}{}},
		keyboards: map[string]tgbotapi.InlineKeyboardMarkup{},
		pages:     map[string]string{}}
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
			tgbotapi.NewInlineKeyboardButtonData(a.config.SprintTitle, a.config.SprintTitle),
			tgbotapi.NewInlineKeyboardButtonData(a.config.EventsTitle, a.config.EventsTitle),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(a.config.TeamsTitle, a.config.TeamsTitle),
			tgbotapi.NewInlineKeyboardButtonData(a.config.CommunitiesTitle, a.config.CommunitiesTitle),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(a.config.ArtifactsTitle, a.config.ArtifactsTitle),
		),
	)
	// teams
	teams := [][]tgbotapi.InlineKeyboardButton{}
	for index, team := range a.config.Teams {
		i := index / 3
		if len(teams) == i {
			teams = append(teams, []tgbotapi.InlineKeyboardButton{})
		}
		teams[i] = append(teams[i], tgbotapi.NewInlineKeyboardButtonData(team.Name, team.Name))
	}
	a.keyboards[a.config.TeamsTitle] = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: teams}

	// communities
	communities := [][]tgbotapi.InlineKeyboardButton{}
	for index, community := range a.config.Communities {
		i := index / 2
		if len(communities) == i {
			communities = append(communities, []tgbotapi.InlineKeyboardButton{})
		}
		communities[i] = append(communities[i], tgbotapi.NewInlineKeyboardButtonData(community.Name, community.Name))
	}
	a.keyboards[a.config.CommunitiesTitle] = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: communities}

	// sprints
	sprints := [][]tgbotapi.InlineKeyboardButton{}
	for index, sprint := range a.config.Sprints {
		i := index / 2
		if len(sprints) == i {
			sprints = append(sprints, []tgbotapi.InlineKeyboardButton{})
		}
		sprints[i] = append(sprints[i], tgbotapi.NewInlineKeyboardButtonData(sprint.Date, sprint.Date))
	}
	a.keyboards[a.config.SprintTitle] = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: sprints}

	// events
	events := [][]tgbotapi.InlineKeyboardButton{}
	for index, event := range a.config.Events {
		i := index / 2
		if len(events) == i {
			events = append(events, []tgbotapi.InlineKeyboardButton{})
		}
		events[i] = append(events[i], tgbotapi.NewInlineKeyboardButtonData(event.Title, event.Title))

		innerEvents := [][]tgbotapi.InlineKeyboardButton{}
		for innerIndex, innerEvent := range event.Events {
			x := innerIndex / 2
			if len(innerEvents) == x {
				innerEvents = append(innerEvents, []tgbotapi.InlineKeyboardButton{})
			}
			eventKey := innerEvent.Title + " " + innerEvent.Date
			innerEvents[x] = append(innerEvents[x], tgbotapi.NewInlineKeyboardButtonData(eventKey, eventKey))
		}
		a.keyboards[event.Title] = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: innerEvents}
	}
	a.keyboards[a.config.EventsTitle] = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: events}

	// artifacts
	artifacts := [][]tgbotapi.InlineKeyboardButton{}
	for index, fact := range a.config.Artifacts {
		i := index / 2
		if len(artifacts) == i {
			artifacts = append(artifacts, []tgbotapi.InlineKeyboardButton{})
		}
		artifacts[i] = append(artifacts[i], tgbotapi.NewInlineKeyboardButtonData(fact.Title, fact.Title))
	}
	a.keyboards[a.config.ArtifactsTitle] = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: artifacts}

	// pages

	// events
	var sb strings.Builder
	sb.WriteString(a.config.EventsTitle)
	sb.WriteString("\n\n")
	sb.WriteString(a.config.EventsInfo)
	sb.WriteString("\n")
	a.pages[a.config.EventsTitle] = sb.String()
	sb.Reset()

	// sprints

	for _, sprint := range a.config.Sprints {
		sb.WriteString(sprint.Date)
		sb.WriteString("\n")
		sb.WriteString(sprint.Goal)
		sb.WriteString("\n\n")
		for _, row := range sprint.Rows {
			sb.WriteString(row.Title)
			sb.WriteString(" - ")
			sb.WriteString(row.Value)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		a.pages[sprint.Date] = sb.String()
		sb.Reset()
	}

	// communities
	for _, community := range a.config.Communities {
		sb.WriteString(community.Name)
		sb.WriteString("\n\n")
		for _, row := range community.Rows {
			sb.WriteString(row.Title)
			sb.WriteString(" ")
			sb.WriteString(row.Value)
			sb.WriteString("\n")
		}
		for _, mentor := range community.Mentors {
			sb.WriteString(mentor.Surname)
			sb.WriteString(" ")
			sb.WriteString(mentor.Name)
			sb.WriteString(" ")
			sb.WriteString(mentor.Link)
			sb.WriteString(" ")
			sb.WriteString(mentor.Phone)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		a.pages[community.Name] = sb.String()
		sb.Reset()
	}

	// teams
	for _, team := range a.config.Teams {
		sb.WriteString(team.Name)
		sb.WriteString("\n\n")
		for _, row := range team.Rows {
			sb.WriteString(row)
			sb.WriteString("\n")
		}
		for _, user := range team.Users {
			sb.WriteString(user.Surname)
			sb.WriteString(" ")
			sb.WriteString(user.Name)
			sb.WriteString(" ")
			sb.WriteString(user.Link)
			sb.WriteString("\n")
			sb.WriteString(user.Skills)
			sb.WriteString(" ")
			sb.WriteString(user.Phone)
			sb.WriteString("\n\n")
		}
		a.pages[team.Name] = sb.String()
		sb.Reset()
	}

	// events
	for _, group := range a.config.Events {
		sb.WriteString(group.Title)
		sb.WriteString("\n")
		for _, row := range group.Rows {
			sb.WriteString(row.Title)
			sb.WriteString(" ")
			sb.WriteString(row.Value)
			sb.WriteString("\n")
		}
		a.pages[group.Title] = sb.String()
		sb.Reset()
		for _, event := range group.Events {
			sb.WriteString(event.Title)
			sb.WriteString(" ")
			sb.WriteString(event.Date)
			sb.WriteString("\n\n")
			sb.WriteString(event.Info)
			sb.WriteString("\n")
			for _, link := range event.Links {
				sb.WriteString(link.Title)
				sb.WriteString(" ")
				sb.WriteString(link.Value)
				sb.WriteString("\n")
			}
			a.pages[event.Title+" "+event.Date] = sb.String()
			sb.Reset()
		}
	}

	// artifacts
	for _, artifact := range a.config.Artifacts {
		sb.WriteString(artifact.Title)
		sb.WriteString("\n")
		for _, row := range artifact.Rows {
			sb.WriteString(row)
			sb.WriteString("\n")
		}
		a.pages[artifact.Title] = sb.String()
		sb.Reset()
	}
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
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
			if page, ok := a.pages[update.CallbackQuery.Data]; ok {
				msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, page)
				msg.ReplyMarkup = a.homeKeyboard
			}
			if keyboard, ok := a.keyboards[update.CallbackQuery.Data]; ok {
				msg.ReplyMarkup = keyboard
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
