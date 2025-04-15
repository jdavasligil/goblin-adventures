package main

import (
	"math/rand/v2"
	"sort"
	"testing"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph(9, 4)
	if g == nil || g.List == nil {
		t.Fatal("Nil graph.")
	}
	if g.List[0] == nil || g.List[8] == nil {
		t.Fatal("Nil lists.")
	}
}

func TestBFS(t *testing.T) {
	g := NewGraph(9, 4)
	g.Connect(0, 1)
	g.Connect(1, 2)
	g.Connect(2, 5)
	g.Connect(5, 8)
	g.Connect(5, 4)
	g.Connect(4, 3)
	g.Connect(3, 6)
	g.Connect(6, 7)

	history, found := g.BFS(0, -1)
	if found {
		t.Fatal("-1 vertex was found")
	}
	if len(history) != len(g.List) {
		t.Fatal("Not all vertices were found.")
	}
	sort.Sort(sort.IntSlice(history))
	for i, v := range history {
		if i != v {
			t.Fatal("Not all vertices were found.")
		}
	}

	_, found = g.BFS(0, 8)
	if !found {
		t.Fatal("Vertex 8 was not found")
	}

	if !g.IsConnected() {
		t.Fatal("Graph is not connected")
	}

	g.Disconnect(2, 5)

	_, found = g.BFS(0, 8)
	if found {
		t.Fatal("Vertex 8 was found after becoming disconnected!")
	}

	if g.IsConnected() {
		t.Fatal("Graph is connected")
	}
}

func TestRandomConnectedGrid(t *testing.T) {
	rng := rand.New(rand.NewPCG(16490829034, 2923842757))
	g := RandomConnectedGrid(rng, 4, 0.0)
	if !g.IsConnected() {
		t.Log("Random grid is not connected")
		t.Fail()
	}
	g.PrintGrid()
}
