package main

import (
	"database/sql"
	"errors"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Enum for queries
const QueryCount int = 1
const (
	// Params:  cmd string
	// Returns: name string, type string
	QueryCommand int = iota
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
		return query, err
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

func CreateCommandTables(db *sql.DB) error {
	defaultCommandTypes := map[string]int{
		"admin": 1,
		"global": 2,
		"combat": 3,
		"explore": 4,
	}
	defaultCommands := []struct {
		Command string
		TypeID int
	}{
		{"inspect", defaultCommandTypes["global"]},
		{"join", defaultCommandTypes["global"]},
		{"sneak", defaultCommandTypes["explore"]},
		{"attack", defaultCommandTypes["combat"]},
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
		type_id INTEGER NOT NULL,
		FOREIGN KEY(type_id) REFERENCES CommandType
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
		_, err = cmdStmt.Exec(cmd.Command, cmd.TypeID)
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

func NewGameDB(filename string) (*sql.DB, error) {
	dbExists := true
	path := "../../db/" + filename
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		dbExists = false
		log.Println("DB does not exist")
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	} else if dbExists {
		return db, nil
	}

	err = CreateCommandTables(db)
	if err != nil {
		return nil, err
	}

	//rows, err := db.Query("SELECT id, name FROM Foo")
	//if err != nil {
	//	return nil, err
	//}
	//defer rows.Close()
	//for rows.Next() {
	//	var id int
	//	var name string
	//	err = rows.Scan(&id, &name)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	log.Println(id, name)
	//}
	//err = rows.Err()
	//if err != nil {
	//	log.Fatal(err)
	//}

	return db, nil
}

func EnsureRegistered(uid string, uname string) {
}

