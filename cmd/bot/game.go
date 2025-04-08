package main

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type GameServer struct {
	DB        *sql.DB
	Commands  chan string
	Interrupt chan os.Signal

	CommandPrefix byte

	GlobalCommands  map[string]bool
	CombatCommands  map[string]bool
	ExploreCommands map[string]bool

	Players map[string]bool

	SaveInterval time.Duration
}

func NewGameServer() *GameServer {
	return &GameServer{
		Commands: make(chan string, 32),

		CommandPrefix: '!',

		GlobalCommands: map[string]bool{
			"inspect": true,
		},
		CombatCommands: map[string]bool{
			"attack": true,
		},
		ExploreCommands: map[string]bool{
			"sneak": true,
		},

		Players: make(map[string]bool),

		SaveInterval: 5 * time.Second,
	}
}

// Loads the game server savestate. Returns new server upon failure.
func LoadGameServer() *GameServer {
	dat, err := os.ReadFile("saves/savestate.gob")
	if err != nil {
		log.Println("game:", err)
		log.Println("Creating a new server.")
		return NewGameServer()
	}
	gs := &GameServer{}
	buf := bytes.Buffer{}
	buf.Write(dat)
	dec := gob.NewDecoder(&buf)
	err = dec.Decode(gs)
	if err != nil {
		log.Println("game:", err)
		log.Println("Failed to load server. Creating a new server.")
		return NewGameServer()
	}
	return gs
}

func (g *GameServer) Save() {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(*g)
	if err != nil {
		log.Println("game:", err)
		log.Println("Failed to save server.")
		return
	}
	err = os.WriteFile("saves/savestate.gob", buf.Bytes(), os.ModePerm)
	if err != nil {
		log.Println("game:", err)
		return
	}
}

func (g *GameServer) InitDB() {
	var err error

	g.DB, err = sql.Open("sqlite3", "../../db/game.db")
	if err != nil {
		log.Println("game:", err)
		return
	}
	defer g.DB.Close()

	_, err = g.DB.Exec(`
	CREATE TABLE Foo (id integer not null primary key, name text);
	DELETE FROM Foo;
	`)
	if err != nil {
		log.Println("db:", err)
	}

	tx, err := g.DB.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("INSERT INTO Foo(id, name) values(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(1, "Amy")
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(2, "Bob")
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(3, "Cat")
	if err != nil {
		log.Fatal(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	rows, err := g.DB.Query("SELECT id, name FROM Foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}

func (g *GameServer) Run() {
	saveTicker := time.NewTicker(g.SaveInterval)
	defer saveTicker.Stop()

	alive := true

	for alive {
		select {
		case <-saveTicker.C:
			g.Save()
		case <-g.Interrupt:
			log.Println("game: Interrupt received.")
			alive = false
		}
	}

	g.Save()
	log.Println("game: Server Saved.")
	log.Println("game: Shutting Down.")
}

func (g *GameServer) IsValidCommand(s string) bool {
	parts := strings.Split(s, " ")
	if len(parts) == 0 {
		return false
	} else if parts[0][0] != g.CommandPrefix {
		return false
	}
	return g.GlobalCommands[s] || g.ExploreCommands[s] || g.CombatCommands[s]
}
