package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type GameServer struct {
	CommandsIn  chan string // For getting commands from bot
	MessagesOut chan string // For sending messages to bot
	Interrupt   chan os.Signal
	Shutdown    chan struct{}

	DB *sql.DB

	Query []*sql.Stmt

	Players map[string]bool

	TickRate time.Duration
}

func NewGameServer() *GameServer {
	db, err := NewGameDB("game.db") 
	if err != nil {
		log.Fatalln("db:", err)
	}
	q, err := InitQuery(db)
	if err != nil {
		log.Fatalln("db:", err)
	}
	return &GameServer{
		CommandsIn: make(chan string, 32),
		Interrupt:  make(chan os.Signal, 1),
		Shutdown:   make(chan struct{}),

		DB:    db,
		Query: q,

		Players: make(map[string]bool),

		TickRate: 32 * time.Millisecond,
	}
}
// ASSUME cmdString has shape "<UserID> <Username> <Cmd> <Cmd Options>..."
// ASSUME commands are authorized by the bot
// Returns true to continue, false to shutdown
func (g *GameServer) HandleCommands() bool {
	var command, commandType string

	for len(g.CommandsIn) > 0 {
		cmdString := <-g.CommandsIn

		if cmdString == "shutdown" {
			return false
		}

		parts := strings.Split(cmdString, " ")
		uid := parts[0]
		uname := parts[1]
		cmd := parts[2:]

		EnsureRegistered(uid, uname)

		err := g.Query[QueryCommand].QueryRow(cmd[0]).Scan(&command, &commandType)
		if err != nil {
			log.Println("game:", err)
			continue
		}

		switch command {
		case "join":
			log.Println("game:", uname, "joined the party!")
		case "inspect":
			opts := cmd[1:]
			if len(opts) < 1 {
				break
			}
			otheruser := opts[0]
			log.Println("game:", uname, "is inspecting", otheruser)
		}
	}

	return true
}

func (g *GameServer) Run() {
	defer close(g.Shutdown)
	defer g.DB.Close()
	defer CloseQuery(g.Query)

	gameTicker := time.NewTicker(g.TickRate)
	defer gameTicker.Stop()

	alive := true

	for alive {
		select {
		case <-gameTicker.C:
			alive = g.HandleCommands()
		case <-g.Interrupt:
			log.Println("game: Interrupt received.")
			alive = false
		}
	}

	// Note: Sync any async transactions if they occur
	log.Println("game: Server saved.")
}
