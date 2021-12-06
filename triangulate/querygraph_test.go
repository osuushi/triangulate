package triangulate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueryGraph(t *testing.T) {
	// Variables for casting
	var ynode YNode
	var xnode XNode
	var sink SinkNode
	segment := &Segment{
		Start: &Point{X: 1, Y: 2},
		End:   &Point{X: 3, Y: 4},
	}
	graph := NewQueryGraph(segment)
	root := graph.Root
	require.NotNil(t, root)

	require.IsType(t, &QueryNode{}, root)

	// Test root node
	require.IsType(t, YNode{}, root.Inner)
	ynode = root.Inner.(YNode)
	assert.Equal(t, 3.0, ynode.Key.X)
	assert.Equal(t, 4.0, ynode.Key.Y)

	// Test top sink
	require.IsType(t, SinkNode{}, ynode.Above.Inner)
	sink = ynode.Above.Inner.(SinkNode)
	// Check parent relationship
	assert.Equal(t, ynode, sink.InitialParent.Inner)
	top := sink.Trapezoid

	// Get the YNode below the top trapezoid
	require.IsType(t, YNode{}, ynode.Below.Inner)
	ynode = ynode.Below.Inner.(YNode)
	assert.Equal(t, 1.0, ynode.Key.X)
	assert.Equal(t, 2.0, ynode.Key.Y)

	// Test bottom sink
	require.IsType(t, SinkNode{}, ynode.Below.Inner)
	sink = ynode.Below.Inner.(SinkNode)
	bottom := sink.Trapezoid
	// Check parent relationship
	assert.Equal(t, ynode, sink.InitialParent.Inner)

	// Get the xnode above the bottom trapezoid
	require.IsType(t, XNode{}, ynode.Above.Inner)
	xnode = ynode.Above.Inner.(XNode)
	assert.Equal(t, segment, xnode.Key)

	// Get the left sink
	require.IsType(t, SinkNode{}, xnode.Left.Inner)
	sink = xnode.Left.Inner.(SinkNode)
	left := sink.Trapezoid

	// Get the right sink
	require.IsType(t, SinkNode{}, xnode.Right.Inner)
	sink = xnode.Right.Inner.(SinkNode)
	right := sink.Trapezoid

	// Assert trapezoid neighbor relationships
	assert.ElementsMatch(t, [2]*Trapezoid{}, top.TrapezoidsAbove)
	assert.ElementsMatch(t, [2]*Trapezoid{left, right}, top.TrapezoidsBelow)
	assert.ElementsMatch(t, [2]*Trapezoid{}, bottom.TrapezoidsBelow)
	assert.ElementsMatch(t, [2]*Trapezoid{left, right}, bottom.TrapezoidsAbove)
	assert.ElementsMatch(t, [2]*Trapezoid{top}, left.TrapezoidsAbove)
	assert.ElementsMatch(t, [2]*Trapezoid{bottom}, left.TrapezoidsBelow)
	assert.ElementsMatch(t, [2]*Trapezoid{top}, right.TrapezoidsAbove)
	assert.ElementsMatch(t, [2]*Trapezoid{bottom}, right.TrapezoidsBelow)

	// Test some points
	trapNames := map[*Trapezoid]string{ // To make test failures easier to read
		top:    "top",
		bottom: "bottom",
		left:   "left",
		right:  "right",
	}
	assertTrapezoidForPoint := func(t *testing.T, trapezoid *Trapezoid, x, y float64) {
		sink := root.FindPoint(&Point{x, y}, Left)
		require.NotNil(t, sink)
		require.IsType(t, SinkNode{}, sink.Inner)
		assert.Equal(t, trapNames[trapezoid], trapNames[sink.Inner.(SinkNode).Trapezoid])
	}

	cases := []struct {
		x, y      float64
		trapezoid *Trapezoid
	}{
		{1, 100, top},
		{10, 3, right},
		{-10, 3, left},
		{1, -100, bottom},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("point %.0f %.0f is in %s trapezoid", c.x, c.y, trapNames[c.trapezoid]), func(t *testing.T) {
			assertTrapezoidForPoint(t, c.trapezoid, c.x, c.y)
		})
	}
}

// func TestAddSegment(t *testing.T) {
// 	g := NewQueryGraph(&Segment{
// 		Start: &Point{X: 1, Y: 2},
// 		End:   &Point{X: 10, Y: 10},
// 	})
// 	g.AddSegment(&Segment{
// 		Start: &Point{X: 8, Y: 3},
// 		End:   &Point{X: 9, Y: 8},
// 	})
// }

func TestSplitTrapezoidHorizontally(t *testing.T) {
	g := NewQueryGraph(&Segment{
		Start: &Point{X: 1, Y: 2},
		End:   &Point{X: 10, Y: 10},
	})
	validateNeighborGraph(t, g)
	p := &Point{X: 7, Y: 5}
	g.SplitTrapezoidHorizontally(g.Root.FindPoint(p, Left), p)
	validateNeighborGraph(t, g)

	p2 := &Point{X: 8, Y: 2}
	g.SplitTrapezoidHorizontally(g.Root.FindPoint(p2, Left), p2)
	validateNeighborGraph(t, g)
}

// Validate that all neighbor relationships make sense. Every neighbor
// relationship should be reflexive, and the set of trapezoids reachable by
// traversing the neighbor graph should be the same as the set of trapezoids in
// the graph.
func validateNeighborGraph(t *testing.T, graph *QueryGraph) {
	// Find all the trapezoids in the graph
	var trapezoids []*Trapezoid
	for node := range IterateGraph(graph.Root) {
		if node, ok := node.Inner.(SinkNode); ok {
			trapezoids = append(trapezoids, node.Trapezoid)
		}
	}

	for _, trapezoid := range trapezoids {
		for _, neighbor := range trapezoid.TrapezoidsAbove {
			if neighbor == nil {
				continue
			}
			// Check reflexivity
			assert.Contains(t, neighbor.TrapezoidsBelow, trapezoid, "above neighbor %s does not have %s as a below neighbor", neighbor, trapezoid)
			// Check graph connectivity
			assert.Contains(t, trapezoids, neighbor, "above neighbor %s is not in the graph", neighbor)
		}
		for _, neighbor := range trapezoid.TrapezoidsBelow {
			if neighbor == nil {
				continue
			}

			// Check reflexivity
			assert.Contains(t, neighbor.TrapezoidsAbove, trapezoid, "below neighbor %s does not have %s as an above neighbor", neighbor, trapezoid)
			// Check graph connectivity
			assert.Contains(t, trapezoids, neighbor, "below neighbor %s is not in the graph", neighbor)
		}
	}
}
