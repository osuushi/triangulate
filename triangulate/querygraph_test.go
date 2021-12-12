package triangulate

import (
	"fmt"
	"math"
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
	assert.ElementsMatch(t, TrapezoidNeighborList{}, top.TrapezoidsAbove)
	assert.ElementsMatch(t, TrapezoidNeighborList{left, right}, top.TrapezoidsBelow)
	assert.ElementsMatch(t, TrapezoidNeighborList{}, bottom.TrapezoidsBelow)
	assert.ElementsMatch(t, TrapezoidNeighborList{left, right}, bottom.TrapezoidsAbove)
	assert.ElementsMatch(t, TrapezoidNeighborList{top}, left.TrapezoidsAbove)
	assert.ElementsMatch(t, TrapezoidNeighborList{bottom}, left.TrapezoidsBelow)
	assert.ElementsMatch(t, TrapezoidNeighborList{top}, right.TrapezoidsAbove)
	assert.ElementsMatch(t, TrapezoidNeighborList{bottom}, right.TrapezoidsBelow)

	// Test some points
	trapNames := map[*Trapezoid]string{ // To make test failures easier to read
		top:    "top",
		bottom: "bottom",
		left:   "left",
		right:  "right",
	}
	assertTrapezoidForPoint := func(t *testing.T, trapezoid *Trapezoid, x, y float64) {
		sink := graph.FindPoint(DefaultDirectionalPoint(x, y))
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

func TestAddSegment(t *testing.T) {
	firstSegment := &Segment{
		Start: &Point{X: 1, Y: 2},
		End:   &Point{X: 10, Y: 10},
	}
	g := NewQueryGraph(firstSegment)
	g.AddSegment(&Segment{
		Start: &Point{X: 8, Y: 3},
		End:   &Point{X: 9, Y: 8},
	})

	// // Add a segment below everything
	// g.AddSegment(&Segment{&Point{X: 5, Y: -30}, &Point{X: 1, Y: -20}})
	// validateNeighborGraph(t, g)

	g.PrintAllTrapezoids()

	// Add a segment that connects to the first one
	connectedSegment := &Segment{firstSegment.End, &Point{X: 20, Y: 4}}
	g.AddSegment(connectedSegment)
	validateNeighborGraph(t, g)

	// Find a point that lies between the two connected segments
	sink := g.FindPoint(DefaultDirectionalPoint(10, 9))
	require.NotNil(t, sink)
	require.IsType(t, SinkNode{}, sink.Inner)
	trapezoid := sink.Inner.(SinkNode).Trapezoid
	// Validate the sides of the trapezoid we found
	assert.Equal(t, firstSegment, trapezoid.Left)
	assert.Equal(t, connectedSegment, trapezoid.Right)
}

func TestSplitTrapezoidHorizontally(t *testing.T) {
	g := NewQueryGraph(&Segment{
		Start: &Point{X: 1, Y: 2},
		End:   &Point{X: 10, Y: 10},
	})
	validateNeighborGraph(t, g)
	p := &Point{X: 7, Y: 5}
	g.SplitTrapezoidHorizontally(g.FindPoint(p.PointingRight()), p)
	validateNeighborGraph(t, g)

	p2 := &Point{X: 8, Y: 2}
	g.SplitTrapezoidHorizontally(g.FindPoint(p2.PointingRight()), p2)
	validateNeighborGraph(t, g)
}

func TestAddPolygon_Triangle(t *testing.T) {
	// Create a graph for a simple triangle with no horizontal edges
	g := &QueryGraph{}
	poly := Polygon{[]*Point{
		{X: 1, Y: 0},
		{X: -1, Y: 1},
		{X: -1, Y: -1},
	}}
	g.AddPolygon(poly)

	// Validate the graph
	validateNeighborGraph(t, g)

	// Test points
	validateGraphBySampling(t, g, PolygonList{poly})
}

func TestAddPolygon_Circle(t *testing.T) {
	// Create a graph for a regular polygon with 100 sides
	g := &QueryGraph{}
	var points []*Point
	var radius float64 = 3
	n := 20
	for i := 0; i < n; i++ {
		angle := 2 * math.Pi * float64(i) / float64(n)
		points = append(points, &Point{X: radius * math.Cos(angle), Y: radius * math.Sin(angle)})
	}

	poly := Polygon{points}

	g.AddPolygon(poly)

	// Scan over the circle sampling points and comparing to the winding rule
	validateGraphBySampling(t, g, PolygonList{poly})
}

func TestAddPolygon_Spiral(t *testing.T) {
	poly := *LoadFixture("spiral")
	g := &QueryGraph{}

	// Testing: skew the points. This is a real stress test of vertical alignment
	// handling, and uncommenting the below code will eliminate the alignment by
	// skewing slightly.

	for _, p := range poly.Points {
		p.Y += p.X * 0.3
	}
	g.AddPolygon(poly)
	g.dbgDraw(70)
	validateGraphBySampling(t, g, PolygonList{poly})
}

func TestAddPolygon_Star(t *testing.T) {
	g := &QueryGraph{}
	star := SimpleStar()
	g.AddPolygons(star)
	validateNeighborGraph(t, g)
	validateGraphBySampling(t, g, star)
}

func TestAddPolygon_SquareWithHole(t *testing.T) {
	list := SquareWithHole()

	g := &QueryGraph{}
	g.AddPolygons(list)
	validateNeighborGraph(t, g)
	validateGraphBySampling(t, g, list)

	fmt.Println("All trapezoids:")
	g.PrintAllTrapezoids()
	fmt.Println("----")
	// Skew the points and draw the trapezoid set
	for _, poly := range list {
		for _, p := range poly.Points {
			p.Y += p.X * 0.3
		}
	}

	g.dbgDraw(70)
}

func TestAddPolygon_StarOutline(t *testing.T) {
	filledPoints := []*Point{}
	holePoints := []*Point{}
	const filledOuterRadius = 10
	const filledInnerRadius = 5
	const holeOuterRadius = filledOuterRadius - 2
	const holeInnerRadius = filledInnerRadius - 2
	for i := 0; i < 10; i++ {
		var (
			filledRadius float64
			holeRadius   float64
		)
		if i%2 == 0 {
			filledRadius = filledOuterRadius
			holeRadius = holeOuterRadius
		} else {
			filledRadius = filledInnerRadius
			holeRadius = holeInnerRadius
		}
		angle := 2 * math.Pi * float64(i) / 10
		filledPoints = append(filledPoints, &Point{X: filledRadius * math.Cos(angle), Y: filledRadius * math.Sin(angle)})
		holePoints = append(holePoints, &Point{X: holeRadius * math.Cos(angle), Y: holeRadius * math.Sin(angle)})
	}

	filledPoly := Polygon{filledPoints}
	holePoly := Polygon{holePoints}.Reverse()

	g := &QueryGraph{}
	g.AddPolygon(filledPoly)
	g.AddPolygon(holePoly)

	validateNeighborGraph(t, g)
	validateGraphBySampling(t, g, PolygonList{filledPoly, holePoly})
}

func TestAddPolygon_StarStripes(t *testing.T) {
	// Multiple inset stars with alternating winding
	var list PolygonList
	const outerRadius = 10
	const n = 20
	var scale float64 = 1
	const indentScale = 0.7
	const gapScale = 0.9

	for i := 0; i < n; i++ {
		var points []*Point
		for j := 0; j < 10; j++ {
			angle := 2 * math.Pi * float64(j) / 10
			r := outerRadius * scale
			if j%2 == 1 {
				r *= indentScale
			}
			points = append(points, &Point{X: r * math.Cos(angle), Y: r * math.Sin(angle)})
		}
		scale *= gapScale
		poly := Polygon{points}
		if i%2 == 1 {
			poly = poly.Reverse()
		}
		list = append(list, poly)
	}
	g := &QueryGraph{}
	for _, poly := range list {
		g.AddPolygon(poly)
	}
	validateNeighborGraph(t, g)
	validateGraphBySampling(t, g, list)
}

func TestAddPolygon_MultiLayeredHoles(t *testing.T) {
	// In this test, we want multiple holes which contain filled shapes inside.
	makeStar := func(x, y, outerRadius, innerRadius float64) Polygon {
		points := []*Point{}
		for i := 0; i < 10; i++ {
			angle := 2 * math.Pi * float64(i) / 10
			r := outerRadius
			if i%2 == 1 {
				r = innerRadius
			}
			points = append(points, &Point{X: x + r*math.Cos(angle), Y: y + r*math.Sin(angle)})
		}
		return Polygon{points}
	}
	list := PolygonList{
		// Outer star
		makeStar(0, 0, 10, 7),
		// Top hole
		makeStar(1.5, 5, 3, 2).Reverse(),
		// Top inner
		makeStar(1.5, 5, 2, 1),
		// Bottom hole
		makeStar(1.8, -5, 3, 2).Reverse(),
		// Bottom inner
		makeStar(1.8, -5, 2, 1),
		// Left hole
		makeStar(-3, 0, 4, 2).Reverse(),
		// Left inner
		makeStar(-3, 0, 3, 1),
	}
	g := &QueryGraph{}
	for _, poly := range list {
		g.AddPolygon(poly)
	}
	validateNeighborGraph(t, g)
	validateGraphBySampling(t, g, list)
}

func validateNeighborGraph(t *testing.T, graph *QueryGraph) {
	// Find all the trapezoids in the graph
	var trapezoids []*Trapezoid
	for node := range IterateGraph(graph.Root) {
		if node, ok := node.Inner.(SinkNode); ok {
			trapezoids = append(trapezoids, node.Trapezoid)
		}
	}

	for _, trapezoid := range trapezoids {
		var count int
		count = 0
		for _, neighbor := range trapezoid.TrapezoidsAbove {
			if neighbor == nil {
				continue
			}
			// Check reflexivity
			assert.Contains(t, neighbor.TrapezoidsBelow, trapezoid, "above neighbor %s does not have %s as a below neighbor", neighbor, trapezoid)
			// Check graph connectivity
			assert.Contains(t, trapezoids, neighbor, "above neighbor %s is not in the graph", neighbor)
			count++
		}
		assert.LessOrEqual(t, count, 2, "trapezoid %s has more than 2 above neighbors", trapezoid)

		count = 0
		for _, neighbor := range trapezoid.TrapezoidsBelow {
			if neighbor == nil {
				continue
			}

			// Check reflexivity
			assert.Contains(t, neighbor.TrapezoidsAbove, trapezoid, "below neighbor %s does not have %s as an above neighbor", neighbor, trapezoid)
			// Check graph connectivity
			assert.Contains(t, trapezoids, neighbor, "below neighbor %s is not in the graph", neighbor)
			count++
		}
		assert.LessOrEqual(t, count, 2, "trapezoid %s has more than 2 below neighbors", trapezoid)
	}
}

func validateGraphBySampling(t *testing.T, graph *QueryGraph, list PolygonList) {
	minX, minY, maxX, maxY, step := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1), 0.1

	for _, poly := range list {
		for _, p := range poly.Points {
			minX = math.Min(minX, p.X)
			minY = math.Min(minY, p.Y)
			maxX = math.Max(maxX, p.X)
			maxY = math.Max(maxY, p.Y)
			maxX = math.Max(maxX, p.X)
		}
	}

	// Pad the bounding box by 10%
	xPadding := (maxX - minX) * 0.1
	yPadding := (maxY - minY) * 0.1
	minX -= xPadding
	minY -= yPadding
	maxX += xPadding
	maxY += yPadding

	for y := minY; y <= maxY; y += step {
		for x := minX; x <= maxX; x += step {
			p := &Point{X: x, Y: y}
			actual := graph.ContainsPoint(p)
			if list.ContainsPointByEvenOdd(p) {
				assert.True(t, actual, "point %v should be in the polygon", p)
			} else {
				assert.False(t, actual, "point %v should not be in the polygon", p)
			}
		}
	}
}
