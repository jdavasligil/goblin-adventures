package main

import (
	"testing"
)

func TestNewGameServer(t *testing.T) {
	gs := NewGameServer()
	if gs == nil {
		t.Fatal("Nil server. Failed to create game server.")
	}
}
