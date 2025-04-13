package main

import "testing"

func TestInitDB(t *testing.T) {
	_, err := NewGameDB("test.db")
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitQuery(t *testing.T) {
	db, err := NewGameDB("test.db")
	if err != nil {
		t.Fatal(err)
	}
	q, err := InitQuery(db)
	if err != nil {
		t.Fatal(err)
	}
	defer CloseQuery(q)

	if q[QueryCommand] == nil {
		t.Fatal("QueryCommand is nil")
	}
}

func TestCreateDeleteCharacter(t *testing.T) {
	db, err := NewGameDB("test.db")
	if err != nil {
		t.Fatal(err)
	}
	CreateCharacter(db, "TestChar1")
	CreateCharacter(db, "TestChar2")
	DeleteCharacter(db, "TestChar1")
	CreateCharacter(db, "TestChar3")
	CreateCharacter(db, "TestChar4")
}
