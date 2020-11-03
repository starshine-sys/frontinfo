package main

import "time"

type member struct {
	Name         string `json:"name"`
	ID           string `json:"id"`
	AvatarURL    string `json:"avatar_url"`
	Birthday     string `json:"birthday"`
	TimeBirthday time.Time
	Pronouns     string    `json:"pronouns"`
	Description  string    `json:"description"`
	Created      time.Time `json:"created"`
}

type system struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type front struct {
	Members []member  `json:"members"`
	Since   time.Time `json:"timestamp"`
}

type pageInfo struct {
	PageTitle string
}
