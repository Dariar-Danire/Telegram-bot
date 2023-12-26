package main

type Update struct {
	Updateid int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}

type Chat struct {
	ChatId int `json:"id"`
}

type BotResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type BotMessage struct {
	ChatId int    `json:"chat_id"`
	Text   string `json:"text"`
}

type User struct {
	Chat_id   int `json:"chat_id"`
	GitHub_id int `json:"GitHub_id"`
}
