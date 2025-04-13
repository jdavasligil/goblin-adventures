package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type DatabaseItem struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Value       int    `json:"value"`
	Attack      string `json:"attack"`
	Defense     int    `json:"defense"`
	Description string `json:"description"`
}

// Enum for queries
const QueryCount int = 2
const (
	// Params:  cmd string
	// Returns: name string, type string
	QueryCommand int = iota

	// Params:  twitch_id int
	// Returns: id int
	QueryUser
)

func InitQuery(db *sql.DB) ([]*sql.Stmt, error) {
	var err error

	query := make([]*sql.Stmt, QueryCount)

	query[QueryCommand], err = db.Prepare(`
	SELECT Command.command, CommandType.type
	FROM Command
	JOIN CommandType ON Command.type_id = CommandType.id
	WHERE command = ?
	`)
	if err != nil {
		return nil, err
	}

	query[QueryUser], err = db.Prepare(`
	SELECT twitch_id FROM User WHERE twitch_id = ? LIMIT 1
	`)
	if err != nil {
		return nil, err
	}

	return query, nil
}

// MUST be called BEFORE db.Close()
func CloseQuery(query []*sql.Stmt) {
	for _, stmt := range query {
		if stmt == nil {
			continue
		}
		err := stmt.Close()
		if err != nil {
			log.Println(err)
		}
	}
}

// TODO: Abstract effect system
func CreateItemTable(db *sql.DB) error {
	defaultItemTypes := map[string]int{
		"empty":      1,
		"armor":      2,
		"weapon":     3,
		"consumable": 4,
	}
	defaultItems := []DatabaseItem{}
	data, err := os.ReadFile(GetPathPrefix() + "data/default_items.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &defaultItems)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE ItemType (
		id INTEGER PRIMARY KEY,
		type TEXT UNIQUE NOT NULL
	) STRICT;
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE Item (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		type_id INTEGER NOT NULL REFERENCES ItemType (id),
		value INTEGER NOT NULL,
		attack TEXT NOT NULL,
		defense INTEGER NOT NULL,
		description TEXT NOT NULL
	) STRICT;
	`)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	itemTypeStmt, err := tx.Prepare("INSERT INTO ItemType (id, type) VALUES (?, ?)")
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	defer itemTypeStmt.Close()

	itemStmt, err := tx.Prepare("INSERT INTO Item (name, type_id, value, attack, defense, description) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	defer itemStmt.Close()

	for itemType, id := range defaultItemTypes {
		_, err = itemTypeStmt.Exec(id, itemType)
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}

	for _, item := range defaultItems {
		_, err = itemStmt.Exec(item.Name, defaultItemTypes[item.Type], item.Value, item.Attack, item.Defense, item.Description)
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// User <- Character <- Inventory
func CreateUserTable(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE User (
		id INTEGER PRIMARY KEY,
		twitch_id TEXT UNIQUE NOT NULL
	) STRICT;
	`)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Spells List / Magic
func CreateCharacterTable(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE Character (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,

		user_id INTEGER NOT NULL REFERENCES User (id) ON DELETE CASCADE,
		armor_id INTEGER NOT NULL REFERENCES Item (id),
		weapon_id INTEGER NOT NULL REFERENCES Item (id),

		level INTEGER NOT NULL,
		experience INTEGER NOT NULL,
		shinies INTEGER NOT NULL,

		might INTEGER NOT NULL, 
		agility INTEGER NOT NULL, 
		will INTEGER NOT NULL ,

		hp INTEGER NOT NULL
	) STRICT;
	`)
	if err != nil {
		return err
	}

	return nil
}

// Empty slots point to the "Empty" item.
// The number of slots here forms a hard limit for all inventories in game
func CreateInventoryTable(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE Inventory (
		id INTEGER PRIMARY KEY,
		parent_id INTEGER NOT NULL REFERENCES Character (id) ON DELETE CASCADE,
		item_id_1 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_2 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_3 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_4 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_5 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_6 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_7 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_8 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_9 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1,
		item_id_10 INTEGER NOT NULL REFERENCES Item (id) DEFAULT 1
	) STRICT;
	`)
	if err != nil {
		return err
	}

	return nil
}

func CreateCommandTables(db *sql.DB) error {
	defaultCommandTypes := map[string]int{
		"admin":   1,
		"global":  2,
		"combat":  3,
		"explore": 4,
	}
	defaultCommands := []struct {
		Command string
		Type    string
	}{
		{"inspect", "global"},
		{"join", "global"},
		{"sneak", "explore"},
		{"attack", "combat"},
	}

	_, err := db.Exec(`
	CREATE TABLE CommandType (
		id INTEGER PRIMARY KEY,
		type TEXT UNIQUE NOT NULL
	) STRICT;
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE Command (
		id INTEGER PRIMARY KEY,
		command TEXT UNIQUE NOT NULL,
		type_id INTEGER NOT NULL REFERENCES CommandType (id)
	) STRICT;
	`)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	cmdTypeStmt, err := tx.Prepare("INSERT INTO CommandType (id, type) VALUES (?, ?)")
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	defer cmdTypeStmt.Close()

	cmdStmt, err := tx.Prepare("INSERT INTO Command (command, type_id) VALUES (?, ?)")
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	defer cmdStmt.Close()

	for cmdType, id := range defaultCommandTypes {
		_, err = cmdTypeStmt.Exec(id, cmdType)
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}

	for _, cmd := range defaultCommands {
		_, err = cmdStmt.Exec(cmd.Command, defaultCommandTypes[cmd.Type])
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func CreateCharacter(db *sql.DB, twitch_id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// id INTEGER PRIMARY KEY,
	// twitch_id INTEGER UNIQUE NOT NULL,
	// character_id INTEGER NOT NULL REFERENCES Character (id)
	userStmt, err := tx.Prepare(`
		INSERT INTO User (
		twitch_id
		) VALUES (?)
		`)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	defer userStmt.Close()

	charStmt, err := tx.Prepare(`
		INSERT INTO Character (
		name,
		user_id,
		armor_id,
		weapon_id,
		level,
		experience,
		shinies,
		might,
		agility,
		will,
		hp
		) VALUES (?, last_insert_rowid(), ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	defer charStmt.Close()

	invStmt, err := tx.Prepare("INSERT INTO Inventory (parent_id) VALUES (last_insert_rowid())")
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	defer invStmt.Close()

	_, err = userStmt.Exec(twitch_id)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	// TODO: Randomly generate values
	_, err = charStmt.Exec(twitch_id, 1, 1, 1, 0, 0, 9, 9, 9, 4)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	_, err = invStmt.Exec()
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func DeleteCharacter(db *sql.DB, twitch_id string) error {
	_, err := db.Exec(`DELETE FROM User WHERE twitch_id = ?`, twitch_id)
	if err != nil {
		return err
	}
	return nil
}

func GetPathPrefix() string {
	if _, err := os.Stat("go.mod"); errors.Is(err, os.ErrNotExist) {
		return "../../"
	}

	return ""
}

func NewGameDB(filename string) (*sql.DB, error) {
	dbFound := false
	path := GetPathPrefix() + "db/" + filename
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		dbFound = true
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	} else if dbFound {
		return db, nil
	}

	err = CreateCommandTables(db)
	if err != nil {
		return nil, err
	}

	err = CreateItemTable(db)
	if err != nil {
		return nil, err
	}

	err = CreateInventoryTable(db)
	if err != nil {
		return nil, err
	}

	err = CreateCharacterTable(db)
	if err != nil {
		return nil, err
	}

	err = CreateUserTable(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}
