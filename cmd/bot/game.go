package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TODO: Game Loop ->
//  1. New Game: Players !join to fill the dungeon party
//  2. Start:    After 10 seconds or when full, start crawl
//  3. Crawl:    Dungeon is generated procedurally one room at a time
//               Players may explore at the risk of deadly traps and combat
//  4. Combat:   In combat play proceeds round by round.

// Commonly modified values are cached in Character
type Character struct {
	Might   int
	Agility int
	Will    int

	AgilityMax int
	MightMax   int
	WillMax    int

	HP    int
	HPMax int

	AttackDie     int
	NumAttackDice int
	AttackBonus   int

	Defense int
}

type Party struct {
	PlayersMax int

	PlayerCharacters map[string]Character
}

type GameServer struct {
	CommandsIn chan string // For getting commands from bot
	Interrupt  chan os.Signal
	Shutdown   chan struct{}

	MessagesOut chan string // For sending messages to bot

	DB *sql.DB

	Query []*sql.Stmt

	Party

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

		Party: Party{
			PlayersMax: 10,
			PlayerCharacters: make(map[string]Character),
		},

		TickRate: 32 * time.Millisecond,
	}
}

func (g *GameServer) EnsureRegistered(uid string) error {
	_, err := g.Query[QueryUser].Query(uid)
	if err != nil {
		log.Println("USER", uid, "DOES NOT EXIST")
		CreateCharacter(g.DB, uid)
	} else {
		log.Println("USER", uid, "EXISTS")
	}

	return nil
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

		g.EnsureRegistered(uid)

		err := g.Query[QueryCommand].QueryRow(cmd[0]).Scan(&command, &commandType)
		if err != nil {
			log.Println("game:", err)
			continue
		}

		switch command {
		case "join":
			m := "game: " + uname + " joined the party!"
			log.Println(m)
			g.MessagesOut <- m
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
