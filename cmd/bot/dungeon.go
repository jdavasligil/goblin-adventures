package main

import (
	"fmt"
	"math/rand"
)

// TODO: Dungeon Generation
// - A new seed offset is generated for each delve.
// - Rooms are generated procedurally (determinisitic) via seed by accepting
//   (hashing) the coordinate by position. Room edits are stored and replayed.
// - Data is kept in the game DB. Such data is queried using the Query slice

// Preventing paradoxes? One way doors, etc?
// We need to generate a "Chunk" of rooms based on set patterns.
// Each Chunk has at least one egress in each direction.
// It is guaranteed for chunks to tile correctly, and to be connected.
// A Chunk is a set of 9 (3x3) rooms.
// This is just for door patterns.

// Fully Connected
//  | | |
// -#-#-#-
//  | | |
// -#-#-#-
//  | | |
// -#-#-#-
//  | | |

// Minimally Connected (Tree)
//    |     |
//  # # #  -# #-#
//  | | |   | | |
// -#-#-#-  # # #
//  | | |   | | |
//  # # #   #-# #-
//    |         |

// Door Boundary Patterns (As observed from the center room facing outward)
// 0: | | |
// 1: | | _
// 2: | _ |
// 3: _ | |
// 4: | _ _
// 5: _ | _
// 6: _ _ |
// Adjacent chunks must mirror the boundary pattern of their neighbor

// Chunk type can be described by the quadruplet ABCD where A,B,C,D are in [0-6]
// and describe the N, S, E, and W boundary type.
// Thus, there are 7^3 = 343 possible chunks by door boundary alone.

// Connection Rules (opposites attract)
//    |     |
//  # # #- -# #-#
//  | | |   | | |
// -#-#-#- -# # #
//  | | |   | | |
//  # # #   #-# #-
//    |         |
// The mating between the two chunks in the example (E to W or C-D) has boundaries
// 1: | | _ (left) to 3: _ | |
// Note: Only patterns 1/3 and 4/6 have chirality; 0, 2, and 5 are symmetric.

// Note Doors connect 2 rooms together, so both rooms need to map to the door.

// Number of possible interior chunk connections:
//  #-#-#  #-#-#  #-#-#  #X#-#  #X#X# 
//  | | |  | | |  | | |  X | |  X   X 
//  #-#-#  #-#-#  #-#-#  # #-#  # #X# 
//  | | |  X | |  X X    X X    X X   
//  #-#-#  # #-#  # #X#  # #X#  # #X# 
//   12     10      7     4       1
// Difficult to count because different edits result in more or less options.
// Graph theory to the rescue!
// OEIS sequence A001187 for the number of distinct connected labeled graphs
// with n nodes produces the result of 66296291072! Hence there are
// 4 * 343 * 66296291072 = 90958511350784 possibilities.

type ChunkType uint32

// COORDINATES: World(X, Y) -> Chunk(U, V) -> Room(S, T)

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
	// 0 1 2
	// 3 4 5
	// 6 7 8
	Rooms [9]RoomData // [X + 3Y], (0,0) is top-left
}

type Position struct {
	X int
	Y int
}

type RoomData struct {
	Stairs StairState
	// 0 1 2 3 = N S E W
	Doors [4]DoorState
}

type Dungeon struct {
	Seed   int64
	Rand   rand.Rand
	Level  int
	Chunks [9]Chunk
}

func hash(p Position) int64 {
	var a int64 = int64(p.X)
	var b int64 = int64(p.Y)
	if a >= b {
		return a*a + a + b
	} else {
		fmt.Print("hi")
		return a + b*b
	}
}


func (d Dungeon) Update(newPos Position) {
	d.Rand.Seed(d.Seed + hash(newPos))
}
