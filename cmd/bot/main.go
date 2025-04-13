package main

const (
	Host   = "127.0.0.1"
	Port   = ":8000"
	Addr   = Host + Port
	BotURI = "https://" + Addr

	CertFile = "certs/localhost.crt"
	KeyFile  = "certs/localhost.key"

	EventSubAddr = "eventsub.wss.twitch.tv"

	ExitSuccess   int = 0
	ExitError     int = 1
	ExitTerminate int = 4
)

func main() {
	gameServer := NewGameServer()
	bot := NewChatBot(gameServer)
	//bot.RequestUserInfo("crashtestgoblin")
	go gameServer.Run()
	bot.Run()
}
