package main

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/jdavasligil/go-ecs/pkg/queue"
)

type Graph struct {
	List [][]int

	IsGrid   bool
	GridSize int
}

func NewGraph(vertMax int, edgePerVertMax int) *Graph {
	g := &Graph{
		List: make([][]int, vertMax),
	}

	for i := range g.List {
		g.List[i] = make([]int, 0, edgePerVertMax)
	}

	return g
}

// Randomly generate a connected nxn grid.
// For each vert, randomly connect adjacent verts (at least 1 connection)
//
//	0-1-2  0-1-2-3    0--------1--------(n-2)--(n-1)
//	| | |  | | | |    |        |          |      |
//	3-4-5  4-5-6-7    n--------5----------6---(2n-1)
//	| | |  | | | |    |        |          |      |
//	6-7-8  8-9-A-B  n(n-2)-----9----------A---(n(n-1)-1)
//	       | | | |    |        |          |      |
//	       C-D-E-F  n(n-1)-(n(n-1)+1)--(nn-2)-(nn-1)
//
// Corners
//
//	0      -> 1      or n
//	n-1    -> n-2    or 2n-1
//	n(n-1) -> n(n-2) or n(n-1)+1
//	n*n-1  -> n*n-2  or n(n-1)-1
//
// Top     i in [1, n-1): (i-1) or (i+1) or (i+n)
// Left    i in [n, n(n-1)) i += n: (i-n) or (i+n) or (i+1)
// Right   i in [(2n-1), (n*n-1)) i += n: (i-n) or (i+n) or (i-1)
// Bottom  i in [n(n-1), (n*n-1)): (i-1) or (i+1) or (i-n)
// Middle  i in [n+1, 2n-1) j in [1, n-1): i+jn +/- 1 or n
func RandomConnectedGrid(rng *rand.Rand, n int, p float32) *Graph {
	g := NewGraph(n*n, 4)
	if n < 2 {
		return g
	}
	g.IsGrid = true
	g.GridSize = n

	// Randomly make connections
	for row := range n {
		for col := range n {
			v := row*n + col
			// North
			if rng.Float32() < p && (row-1) > 0 {
				g.Connect(v, v-n)
			}
			// East
			if rng.Float32() < p && (col+1) < n {
				g.Connect(v, v+1)
			}
			// South
			if rng.Float32() < p && (row+1) < n {
				g.Connect(v, v+n)
			}
			// West
			if rng.Float32() < p && (col-1) > 0 {
				g.Connect(v, v-1)
			}
		}
	}

	// Ensure each vertex has at least one connection

	// Top Left
	if len(g.List[0]) == 0 {
		if rng.UintN(2) == 0 {
			g.Connect(0, 1)
		} else {
			g.Connect(0, n)
		}
	}

	// Top Right
	if len(g.List[n-1]) == 0 {
		if rng.UintN(2) == 0 {
			g.Connect(n-1, n-2)
		} else {
			g.Connect(n-1, 2*n-1)
		}
	}

	// Bottom Left
	if len(g.List[n*(n-1)]) == 0 {
		if rng.UintN(2) == 0 {
			g.Connect(n*(n-1), n*(n-2))
		} else {
			g.Connect(n*(n-1), n*(n-1)+1)
		}
	}

	// Bottom Right
	if len(g.List[n*n-1]) == 0 {
		if rng.UintN(2) == 0 {
			g.Connect(n*n-1, n*(n-1)-1)
		} else {
			g.Connect(n*n-1, n*n-2)
		}
	}

	// Top
	for i := 1; i < n-1; i++ {
		if len(g.List[i]) == 0 {
			selection := rng.UintN(3)
			switch selection {
			case 0:
				g.Connect(i, i-1)
			case 1:
				g.Connect(i, i+1)
			case 2:
				g.Connect(i, i+n)
			}
		}
	}

	// Left
	for i := n; i < n*(n-1); i += n {
		if len(g.List[i]) == 0 {
			selection := rng.UintN(3)
			switch selection {
			case 0:
				g.Connect(i, i-n)
			case 1:
				g.Connect(i, i+n)
			case 2:
				g.Connect(i, i+1)
			}
		}
	}

	// Right
	for i := 2*n - 1; i < n*n-1; i += n {
		if len(g.List[i]) == 0 {
			selection := rng.UintN(3)
			switch selection {
			case 0:
				g.Connect(i, i-n)
			case 1:
				g.Connect(i, i+n)
			case 2:
				g.Connect(i, i-1)
			}
		}
	}

	// Bottom
	for i := n * (n - 1); i < n*n-1; i++ {
		if len(g.List[i]) == 0 {
			selection := rng.UintN(3)
			switch selection {
			case 0:
				g.Connect(i, i-1)
			case 1:
				g.Connect(i, i+1)
			case 2:
				g.Connect(i, i-n)
			}
		}
	}

	// Middle Values
	for y := 1; y < n-1; y++ {
		for x := 1; x < n-1; x++ {
			idx := x + y*n
			//fmt.Printf("(%d, %d)\n", x, y)
			if len(g.List[idx]) == 0 {
				selection := rng.UintN(4)
				switch selection {
				case 0:
					g.Connect(idx, idx-n)
				case 1:
					g.Connect(idx, idx+n)
				case 2:
					g.Connect(idx, idx+1)
				case 3:
					g.Connect(idx, idx-1)
				}
			}
		}
	}

	// Ensure graph is connected with iterative DFS passes
	// We use the complementary logic for visitation
	root := 0
	unvisited := make(map[int]struct{}, len(g.List))
	for i := 1; i < len(g.List); i++ {
		unvisited[i] = struct{}{}
	}

	subgraph := make([]int, 0, len(g.List))
	vq := queue.NewRingBuffer[int](len(g.List))

	vq.Push(root)
	subgraph = append(subgraph, root)

	for !vq.IsEmpty() {
		v := vq.Pop()
		// For each connected vert w from v to w
		for _, w := range g.List[v] {
			_, notVisited := unvisited[w]
			if notVisited {
				delete(unvisited, w)
				subgraph = append(subgraph, w)
				vq.Push(w)
			}
		}
	}

	if len(unvisited) == 0 {
		return g
	}

	// Collect disjoint set of connected subgraphs
	disjoint := make([][]int, 0, n*n)

	//sort.Sort(sort.IntSlice(subgraph))
	disjoint = append(disjoint, subgraph)

	for len(unvisited) != 0 {
		// Get arbitrary key from list of unvisited verts
		for k := range unvisited {
			root = k
			break
		}
		subgraph = make([]int, 0, len(unvisited))
		vq.Push(root)

		for !vq.IsEmpty() {
			v := vq.Pop()
			// For each connected vert w from v to w
			for _, w := range g.List[v] {
				_, notVisited := unvisited[w]
				if notVisited {
					delete(unvisited, w)
					subgraph = append(subgraph, w)
					vq.Push(w)
				}
			}
		}
		disjoint = append(disjoint, subgraph)
	}

	// Connect adjacent subgraphs
	found := false
	idx := 0
	otheridx := 0
	v := 0
	for len(disjoint) > 1 {
		// Find connection for first set
		found = false
		idx = 0
		otheridx = 0
		// loop over first set
		for !found {
			v = disjoint[0][idx]
			// search for valid grid connection in another set
			for i := 1; i < len(disjoint); i++ {
				for _, w := range disjoint[i] {
					if g.IsValidGridConnection(v, w) {
						found = true
						otheridx = i
						g.Connect(v, w)
						break
					}
				}
				if found {
					break
				}
			}
			idx++
		}
		// Merge the first and newly connected set then delete the other set
		disjoint[0] = slices.Concat(disjoint[0], disjoint[otheridx])
		disjoint = slices.Delete(disjoint, otheridx, otheridx+1)
	}

	return g
}

func (g Graph) IsValidGridConnection(v int, w int) bool {
	return g.RelativeGridDirection(v, w) >= 0
}

// func (g Graph) IsValidGridConnection(v int, w int) bool {
// 	n := g.GridSize
//
// 	// Corners
// 	switch v {
// 	case 0:
// 		return w == 1 || w == n
// 	case n - 1:
// 		return w == (n-2) || w == (2*n-1)
// 	case n * (n - 1):
// 		return w == n*(n-2) || w == n*(n-1)+1
// 	case n*n - 1:
// 		return w == n*n-2 || w == n*(n-1)-1
// 	}
//
// 	// Edges
// 	if v < n {
// 		return w == (v-1) || w == (v+1) || w == (v+n)
// 	} else if v > n*(n-1) {
// 		return w == (v-1) || w == (v+1) || w == (v-n)
// 	} else if v%n == 0 {
// 		return w == (v+1) || w == (v-n) || w == (v+n)
// 	} else if v%n == 1 {
// 		return w == (v-1) || w == (v-n) || w == (v+n)
// 	}
//
// 	return w == (v+1) || w == (v-1) || w == (v+n) || w == (v-n)
// }

// Returns
//
//	0: w is N of v
//	1: w is E of v
//	2: w is S of v
//	3: w is W of v
//
// -1: Not an adjacent vertex
func (g Graph) RelativeGridDirection(v int, w int) int {
	switch w - v {
	case -g.GridSize:
		return 0
	case 1:
		return 1
	case g.GridSize:
		return 2
	case -1:
		return 3
	}
	return -1
}

func (g Graph) PrintGrid() {
	if !g.IsGrid {
		return
	}
	var sb strings.Builder
	n := g.GridSize

	// o-o  o-o-o 0
	// | |  | | |
	// o-o  o-o-o 1
	//      | | |
	//      o-o-o
	// 0 1  0 1 2
	for y := range n - 1 {
		sb.WriteString("\n  o")
		for x := range n - 1 {
			idx := x + y*n
			if g.IsEdge(idx, idx+1) {
				sb.WriteString("-")
			} else {
				sb.WriteString(" ")
			}
			sb.WriteString("o")
		}
		sb.WriteString("\n  ")
		for x := range n {
			idx := x + y*n
			if g.IsEdge(idx, idx+n) {
				sb.WriteString("|")
			} else {
				sb.WriteString(" ")
			}
			sb.WriteString(" ")
		}
	}
	sb.WriteString("\n  o")
	for x := range n - 1 {
		idx := x + (n-1)*n
		if g.IsEdge(idx, idx+1) {
			sb.WriteString("-")
		} else {
			sb.WriteString(" ")
		}
		sb.WriteString("o")
	}
	sb.WriteString("\n\n")

	fmt.Print(sb.String())
}

// Create an undirected edge from v to w.
func (g Graph) Connect(v int, w int) {
	g.List[v] = append(g.List[v], w)
	g.List[w] = append(g.List[w], v)
}

// Remove an undirected edge from v to w.
func (g Graph) Disconnect(v int, w int) {
	for i := range g.List[v] {
		if g.List[v][i] == w {
			g.List[v] = slices.Delete(g.List[v], i, i+1)
			break
		}
	}
	for i := range g.List[w] {
		if g.List[w][i] == v {
			g.List[w] = slices.Delete(g.List[w], i, i+1)
			break
		}
	}
}

func (g Graph) IsEdge(v int, w int) bool {
	return slices.Contains(g.List[v], w)
}

func (g Graph) IsConnected() bool {
	visited := make([]bool, len(g.List))
	visited[0] = true

	vq := queue.NewRingBuffer[int](len(g.List))
	vq.Push(0)
	count := 1

	for !vq.IsEmpty() {
		v := vq.Pop()
		// For each connected vert w from v to w
		for _, w := range g.List[v] {
			if !visited[w] {
				visited[w] = true
				vq.Push(w)
				count++
			}
		}
	}

	return count == len(g.List)
}

// Breadth-First-Search for the target vertex.
//
// Note
//
//	Provide a target of -1 to search for all connected vertices.
//
// Params
//
//	root   The starting vertex
//	target The vertex target of interest
//
// Returns
//
//	[]int The search history of vertices as a list of integers
//	bool  True if target is found
func (g Graph) BFS(root int, target int) ([]int, bool) {
	visited := make([]bool, len(g.List))
	visited[root] = true

	history := make([]int, 0, len(g.List))
	history = append(history, root)

	vq := queue.NewRingBuffer[int](len(g.List))
	vq.Push(root)

	for !vq.IsEmpty() {
		v := vq.Pop()
		if v == target {
			return history, true
		}
		// For each connected vert w from v to w
		for _, w := range g.List[v] {
			if !visited[w] {
				visited[w] = true
				history = append(history, w)
				vq.Push(w)
			}
		}
	}

	return history, false
}

func (g Graph) Print() {
	for i, adj := range g.List {
		fmt.Printf("    %d: %v\n", i, adj)
	}
}
