package main

import (
	"fmt"
	"math/rand/v2"
)

// TODO: Dungeon Generation
// - A new seed offset is generated for each delve.
// - Rooms are generated procedurally (determinisitic) via seed by accepting
//   (hashing) the coordinate by position. Room edits are stored and replayed.
// - Data is kept in the game DB. Such data is queried using the Query slice


const SEEDCONST uint64 = 3355278010277430012

// COORDINATES: World(X, Y) -> Chunk(U, V) -> Room(S, T)
// MUST be perfect squares
const (
	CHUNKSIZE int = 16
	CHUNKCOUNT    = 9
)

type DoorState byte

const (
	DoorNone DoorState = iota
	DoorOpen
	DoorClosed
	DoorStuck
	DoorLocked
)

type StairState byte

const (
	StairNone StairState = iota
	StairDown
	StairUp
)

const (
	North int = iota
	South
	East
	West
)

type Chunk struct {
    // Index by [X + CHUNKSIZE * Y], (0,0) is top-left
	Rooms [CHUNKSIZE]Room
}

type Position struct {
	X int
	Y int
}

type Room struct {
	Stairs StairState
	// 0 1 2 3 = N S E W
	Doors [4]DoorState
}

type Dungeon struct {
	Seed      uint64
	RandState rand.PCG
	Rand      rand.Rand

	RoomPos  Position
	ChunkPos Position
	Level     int

	Chunks    [CHUNKCOUNT]Chunk
}

func NewDungeon(seed uint64) *Dungeon {
	pcg := rand.NewPCG(seed, SEEDCONST ^ seed)
	d := &Dungeon{
		Seed:      seed,
		RandState: *pcg,
		Rand: *rand.New(pcg)
	}
	return d
}

func (p Position) Hash() uint64 {
	return uint64(p.X)<<32 | uint64(p.Y)
}

func (d *Dungeon) Update(newPos Position) {
	seed := newPos.Hash() ^ d.Seed
	d.RandState.Seed(seed, SEEDCONST ^ seed)
}
