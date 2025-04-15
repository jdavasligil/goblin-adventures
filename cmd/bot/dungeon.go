package main

import (
	"math/rand/v2"
)

// TODO: Dungeon Generation
// - A new seed offset is generated for each delve.
// - Rooms are generated procedurally (determinisitic) via seed by accepting
//   (hashing) the coordinate by position. Room edits are stored and replayed.
// - Data is kept in the game DB. Such data is queried using the Query slice

const SEEDCONST uint64 = 3355278010277430012

// MUST be a perfect square
const (
	CHUNKSIZE     int = 16
	CHUNKSIZEROOT     = 4
)

type DoorState byte

const (
	DoorNone DoorState = iota
	DoorOpen
	DoorClosed
	DoorStuck
	DoorLocked
)

var DoorCDF = []float32{
	0.00,
	0.20,
	0.70,
	0.90,
	1.00,
}

type StairState byte

const (
	StairNone StairState = iota
	StairDown
	StairUp
)

var StairCDF = []float32{
	0.80,
	0.90,
	1.00,
}

const (
	North int = iota
	East
	South
	West
)

// Cast result as state type.
func RandomState(rng *rand.Rand, cdf []float32) int {
	for i, p := range cdf {
		if rng.Float32() < p {
			return i
		}
	}
	return 0
}

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
	// 0 1 2 3 = N E S W
	Doors [4]DoorState
}

func (r *Room) Randomize(rng *rand.Rand) {
	r.Stairs = StairState(RandomState(rng, StairCDF))
}

type Dungeon struct {
	Seed      uint64
	RandState rand.PCG
	Rand      *rand.Rand

	ConnectProbability float32

	RoomPos  Position
	ChunkPos Position
	Level    int

	Chunk
}

func NewDungeon(seed uint64) *Dungeon {
	d := &Dungeon{
		Seed:      seed,
		RandState: *rand.NewPCG(seed, SEEDCONST^seed),

		ConnectProbability: 0.25,
	}
	d.Rand = rand.New(&d.RandState)
	d.RoomPos.X = CHUNKSIZE / 2
	d.RoomPos.Y = CHUNKSIZE / 2
	d.UpdateChunk()

	return d
}

func (d *Dungeon) UpdateChunk() {
	seed := d.ChunkPos.Hash() ^ d.Seed
	d.RandState.Seed(seed, SEEDCONST^seed)
	g := RandomConnectedGrid(d.Rand, CHUNKSIZEROOT, d.ConnectProbability)
	for _, r := range d.Chunk.Rooms {
		r.Randomize(d.Rand)
	}

	// Randomize doors with interior connections matching the graph
	for v, adj := range g.List {
		for _, w := range adj {
			dir := g.RelativeGridDirection(v, w)
			d.Chunk.Rooms[v].Doors[dir] = DoorState(RandomState(d.Rand, DoorCDF))
			d.Chunk.Rooms[w].Doors[OppositeDirection(dir)] = d.Chunk.Rooms[w].Doors[South]
		}
	}

	// TODO: Randomize outgoing doors for stitching chunks
}

func (p Position) Hash() uint64 {
	return uint64(p.X)<<32 | uint64(p.Y)
}

// N <-> S, E <-> W
// Proof:
//
//	N: (0+2)%4 = 2 = S
//	S: (2+2)%4 = 0 = N
//	E: (1+2)%4 = 3 = W
//	W: (3+2)%4 = 1 = E
func OppositeDirection(dir int) int {
	return int(uint(dir+2) & 0b00000011) // mod 4 in base 2 is a binary op
}
