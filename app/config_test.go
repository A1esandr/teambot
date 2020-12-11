package app

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWhenLoadConfigFile(t *testing.T) {
	config := &Config{}
	err := config.loadConfig()
	if err != nil {
		fmt.Println(err)
	}
	assert.Equal(t, "Welcome", config.Welcome)
}
