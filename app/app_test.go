package app

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func SetUp() *App {
	app := NewApp()
	err := os.Setenv("TOKEN", "test")
	if err != nil {
		fmt.Println(err)
	}
	return app
}

func SetUpReadyApp() *App {
	app := NewApp()
	err := os.Setenv("TOKEN", "test")
	if err != nil {
		fmt.Println(err)
	}
	app.init()
	return app
}

func CleanUp() {
	err := os.Setenv("TOKEN", "")
	if err != nil {
		fmt.Println(err)
	}
}

func TestWhenApp_Init_LoadTokenFromEnv(t *testing.T) {
	app := NewApp()

	err := os.Setenv("TOKEN", "test")
	if err != nil {
		fmt.Println(err)
	}
	app.init()
	// Clean up
	err = os.Setenv("TOKEN", "")
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, "test", app.token)
}

func TestWhenApp_Init_PanicsWithoutToken(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Error must be thrown when token is empty!")
		}
		if r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	app := NewApp()

	app.init()
}

func TestWhenApp_Init_LoadMessages(t *testing.T) {
	app := NewApp()

	err := os.Setenv("TOKEN", "test")
	if err != nil {
		fmt.Println(err)
	}
	app.init()
	// Clean up
	err = os.Setenv("TOKEN", "")
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, "Welcome", app.config.Welcome)
}

func TestWhenApp_GetStartMessage_ShowWelcomeMessage(t *testing.T) {
	app := NewApp()
	app.config.Welcome = "Welcome"

	msg := app.chooseMsg("/start")

	assert.Equal(t, "Welcome", msg)
}

func TestWhenApp_Launch_LoadAuthData(t *testing.T) {
	defer CleanUp()
	app := SetUp()

	app.init()

	assert.Equal(t, 1, len(app.users))
}

func TestWhenApp_AfterLoadAuthDataError_ShowError(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Error must be thrown when users are empty!")
		} else {
			fmt.Println("Recovered in f", r)
		}
		ReaderFile = ioutil.ReadFile
	}()
	ReaderFile = func(filename string) ([]byte, error) {
		return nil, errors.New("Read error")
	}
	app := NewApp()

	app.loadUsers()
}

func TestWhenApp_AfterLoadAuthDataEmpty_ShowError(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Error must be thrown when users are empty!")
		} else {
			fmt.Println("Recovered in f", r)
		}
		ReaderFile = ioutil.ReadFile
	}()
	ReaderFile = func(filename string) ([]byte, error) {
		return []byte{}, nil
	}
	app := NewApp()

	app.loadUsers()
}

func TestWhenUser_NotAuthorized_ShowWelcomeAuthMessage(t *testing.T) {
	defer CleanUp()
	app := SetUpReadyApp()

	input := &Message{}
	msg := app.handle(input)

	assert.Equal(t, "Hi! You are not authorized. Please send your Surname Name and auth data", msg)
}

func TestWhenUser_NotAuthorized_SendsAuthMessage(t *testing.T) {
	defer CleanUp()
	app := SetUpReadyApp()

	input := &Message{Text: "Smith John abc"}
	msg := app.handle(input)

	assert.Equal(t, "Hey! I know you!", msg)
}

func TestWhenUser_Authorized_SendHomeKeyboard(t *testing.T) {
	defer CleanUp()
	app := SetUpReadyApp()

	keyboard := app.chooseKeyboard(app.config.Authorized)

	assert.Equal(t, app.homeKeyboard, keyboard)
}
