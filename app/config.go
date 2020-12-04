package app

import (
	"encoding/json"
	"fmt"
)

type (
	Config struct {
		Welcome          string        `json:"welcome"`
		AuthMsg          string        `json:"auth_msg"`
		Authorized       string        `json:"authorized"`
		TeamsTitle       string        `json:"teams_button_title"`
		SprintTitle      string        `json:"sprint_button_title"`
		CommunitiesTitle string        `json:"communities_button_title"`
		EventsTitle      string        `json:"events_button_title"`
		ArtifactsTitle   string        `json:"artifacts_button_title"`
		EventsInfo       string        `json:"events_info"`
		Teams            []Team        `json:"teams"`
		Sprints          []Sprint      `json:"sprints"`
		Communities      []Community   `json:"communities"`
		Events           []EventsGroup `json:"events"`
		Artifacts        []Record      `json:"artifacts"`
	}
)

func (c *Config) loadConfig() error {
	data, err := ReaderFile("../config/config.json")
	if err != nil {
		data, err = ReaderFile("config/config.json")
		if err != nil {
			return err
		}
	}
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("Error to parse config %s", err)
	}
	return nil
}
