package triangulate

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/osuushi/triangulate/dbg"
)

// This implements the data structures for Seidel 1991 for trapezoidizing a non-monotone polygon
// into multiple segments. It uses the same lexicographic convention as
// elsewhere which avoids equal y values by lexicographic rotation.

type XDirection int

const (
	Left XDirection = iota
	Right
)

type YDirection int

const (
	Down = iota
	Up
)

type Direction struct {
	X XDirection
	Y YDirection
}

// This is an arbitrary direction for when you don't really care (e.g. tests)
var DefaultDirection = Direction{X: Left, Y: Down}

type QueryGraph struct {
	Root *QueryNode
}

// A graph iterator lets you loop over the nodes in a graph exactly once.
// Traversal order is not defined. Behavior is also undefined if you modify the
// graph during iteration.
type GraphIterator struct {
	stack []*QueryNode
	seen  map[*QueryNode]struct{}
}

func IterateGraph(root *QueryNode) chan *QueryNode {
	iter := NewGraphIterator(root)
	return iter.MakeChan()
}

func IterateTrapezoids(root *QueryNode) chan *Trapezoid {
	ch := make(chan *Trapezoid)
	go func() {
		for node := range IterateGraph(root) {
			if sink, ok := node.Inner.(SinkNode); ok {
				ch <- sink.Trapezoid
			}
		}
		close(ch)
	}()
	return ch
}

func NewGraphIterator(root *QueryNode) *GraphIterator {
	return &GraphIterator{[]*QueryNode{root}, map[*QueryNode]struct{}{}}
}

// Create a channel using a go routine to iterate over the subgraph. This provides
// a nicer API for looping, and allows the graph juggling to happen in another
// thread when possible.
func (iter *GraphIterator) MakeChan() chan *QueryNode {
	ch := make(chan *QueryNode)
	go func() {
		for {
			node := iter.Next()
			if node == nil {
				break
			}
			ch <- node
		}
		close(ch)
	}()
	return ch
}

func (iter *GraphIterator) Next() *QueryNode {
	if len(iter.stack) == 0 {
		return nil
	}
	node := iter.stack[len(iter.stack)-1]
	iter.stack = iter.stack[:len(iter.stack)-1]
	// Skip if we've seen the node before
	if _, ok := iter.seen[node]; ok {
		return iter.Next()
	}

	iter.seen[node] = struct{}{}

	// Push the children onto the stack
	iter.stack = append(iter.stack, node.ChildNodes()...)

	return node
}

// Create a new graph from a single segment, and return the root node.
func NewQueryGraph(segment *Segment) *QueryGraph {

	a := segment.Top()
	b := segment.Bottom()

	// We create the following trapezoid graph:
	/*
		         top
		------a--------------
		 left  \  right
		--------b------------
		       bottom

		Where:
		  a = segment.Top()
			b = segment.Bottom()
		And top, right, bottom and left are trapezoids (currently with infinite width)
	*/

	top := &Trapezoid{
		Left:   nil,
		Right:  nil,
		Top:    nil,
		Bottom: a,
	}

	top.Sink = &QueryNode{SinkNode{Trapezoid: top}}

	left := &Trapezoid{
		Left:   nil,
		Right:  segment,
		Top:    a,
		Bottom: b,
	}
	left.Sink = &QueryNode{SinkNode{Trapezoid: left}}

	right := &Trapezoid{
		Left:   segment,
		Right:  nil,
		Top:    a,
		Bottom: b,
	}
	right.Sink = &QueryNode{SinkNode{Trapezoid: right}}

	bottom := &Trapezoid{
		Left:   nil,
		Right:  nil,
		Top:    b,
		Bottom: nil,
	}
	bottom.Sink = &QueryNode{SinkNode{Trapezoid: bottom}}

	// Set up the neighbor relationships
	top.TrapezoidsBelow[0] = left
	top.TrapezoidsBelow[1] = right
	left.TrapezoidsAbove[0] = top
	left.TrapezoidsBelow[0] = bottom
	right.TrapezoidsAbove[0] = top
	right.TrapezoidsBelow[0] = bottom
	bottom.TrapezoidsAbove[0] = left
	bottom.TrapezoidsAbove[1] = right

	// Build the initial query graph pointing at the sinks
	graph := &QueryNode{YNode{
		Key:   a,
		Above: top.Sink,
		Below: &QueryNode{YNode{
			Key:   b,
			Below: bottom.Sink,
			Above: &QueryNode{XNode{
				Key:   segment,
				Left:  left.Sink,
				Right: right.Sink,
			}},
		}},
	}}

	// Backlink all the trapezoid sinks to their initial parents

	for node := range IterateGraph(graph) {
		for _, child := range node.ChildNodes() {
			if sink, ok := child.Inner.(SinkNode); ok {
				sink.InitialParent = node
				child.Inner = sink
			}
		}
	}

	return &QueryGraph{Root: graph}
}

func (graph *QueryGraph) PrintAllTrapezoids() {
	var parts []string
	for node := range IterateGraph(graph.Root) {
		if node, ok := node.Inner.(SinkNode); ok {
			parts = append(parts, node.Trapezoid.String())
		}
	}

	fmt.Println(strings.Join(parts, "\n"))
}

func (graph *QueryGraph) FindPoint(p *Point, dir Direction) *QueryNode {
	fmt.Println("Finding point", p)
	return graph.Root.FindPoint(p, dir)
}

func (graph *QueryGraph) AddSegment(segment *Segment) {
	if segment == nil {
		panic("nil segment")
	}
	top := segment.Top()
	bottom := segment.Bottom()

	topToBottomDirection := Direction{
		X: segment.XDirection(),
		Y: Down,
	}

	// Find the node that contains the top point, coming from the bottom
	node := graph.FindPoint(top, topToBottomDirection)

	var topTrapezoid = node.Inner.(SinkNode).Trapezoid

	// Check if the top point is already in the graph. If so, no horizontal split is needed
	if !topTrapezoid.HasPoint(top) {
		fmt.Println("Splitting for top")
		fmt.Println("Top", top)
		graph.SplitTrapezoidHorizontally(node, top)
	}

	// Do the same process for the bottom point
	node = graph.FindPoint(bottom, topToBottomDirection.Opposite())
	var bottomTrapezoid = node.Inner.(SinkNode).Trapezoid

	// Same check
	if !bottomTrapezoid.HasPoint(bottom) {
		fmt.Println("Splitting for bottom")
		fmt.Println("Bottom", bottom)
		graph.SplitTrapezoidHorizontally(node, bottom)
		// We now want the top sink trapezoid, since the line segment crosses that.
		bottomTrapezoid = node.Inner.(YNode).Above.Inner.(SinkNode).Trapezoid
	}

	// Split the trapezoids that intersect the line segment. Note at this point
	// that `top` sits exactly on top of the top trapezoid, and `bottom` sits
	// exactly on the bottom of the bottom trapezoid.
	curTrapezoid := bottomTrapezoid
	graph.dbgDraw(50)
	var leftTrapezoids []*Trapezoid
	var rightTrapezoids []*Trapezoid
trapezoidLoop:
	for { // Loop over the trapezoids
		// Split this trapezoid horizontally
		leftTrapezoid, rightTrapezoid := curTrapezoid.SplitBySegment(segment)
		leftTrapezoids = append(leftTrapezoids, leftTrapezoid)
		rightTrapezoids = append(rightTrapezoids, rightTrapezoid)

		// Find the next trapezoid out of the up to two neighbors above this one. It
		// will be the one whose bottom the line segment intersects

		for _, neighbor := range curTrapezoid.TrapezoidsAbove {
			if neighbor != nil && neighbor.BottomIntersectsSegment(segment) {
				curTrapezoid = neighbor
				break
			}
			// If we don't find a neighbor, we're done splitting
			break trapezoidLoop
		}

		// We'll stop once we get to the trapezoid that our segment top is the
		// bottom of. That's the one we created by splitting horizontally.
		if top == curTrapezoid.Bottom {
			break
		}
	}

	// We now have left and right chains of triangles that were split by the line
	// segment, but some of them may share edges, so we need to merge them. All of
	// the left trapezoids have the segment as a right edge and vice versa, so we
	// can treat each chain of trapezoids separately

	for i, chain := range [2][]*Trapezoid{leftTrapezoids, rightTrapezoids} {
		side := XDirection(i)
		// Divide the chain into chunks of connected trapezoids. Trapezoids can only
		// be merged if they're consecutive in the chain
		var chunks [][]*Trapezoid
		curChunk := []*Trapezoid{chain[0]}
		for _, trapezoid := range chain[1:] {
			if curChunk[0].CanMergeWith(trapezoid) {
				curChunk = append(curChunk, trapezoid)
			} else {
				chunks = append(chunks, curChunk)
				curChunk = []*Trapezoid{trapezoid}
			}
		}
		// Add on the last chunk
		chunks = append(chunks, curChunk)

		// Merge each chunk
		for _, chunk := range chunks {
			mergedTrapezoid := new(Trapezoid)
			bottomTrapezoid := chunk[0]
			*mergedTrapezoid = *bottomTrapezoid
			topTrapezoid := chunk[len(chunk)-1]
			// Merge geometry
			mergedTrapezoid.Top = topTrapezoid.Top
			// Merge neighbors
			mergedTrapezoid.TrapezoidsAbove = topTrapezoid.TrapezoidsAbove
			// Make the neighbors agree
			for _, neighbor := range mergedTrapezoid.TrapezoidsAbove {
				if neighbor == nil {
					continue
				}
				neighbor.TrapezoidsBelow.ReplaceOrAdd(topTrapezoid, mergedTrapezoid)
			}

			for _, neighbor := range mergedTrapezoid.TrapezoidsBelow {
				if neighbor == nil {
					continue
				}
				neighbor.TrapezoidsAbove.ReplaceOrAdd(bottomTrapezoid, mergedTrapezoid)
			}

			// Note that we can't set an initial parent on the new sink, because
			// (assuming there's more than one trapezoid in the chunk), the node will
			// have multiple XNode parents.
			mergedTrapezoid.Sink = &QueryNode{SinkNode{Trapezoid: mergedTrapezoid}}

			// Change every SinkNode to XNode, or complete the XNode depending on direction
			for _, trapezoid := range chunk {
				// Get the sink off the trapezoid. Note that this is a trapezoid we
				// created by SplitBySegment, so its sink still points at the original
				// trapezoid
				node := trapezoid.Sink
				var xnode XNode
				if side == Left { // On left side, we're making a new XNode
					xnode = XNode{
						Key:  segment,
						Left: mergedTrapezoid.Sink,
					}
				} else { // On right side, we created the xnode when we did the left side, so we just need to update it
					xnode = node.Inner.(XNode)
					xnode.Right = mergedTrapezoid.Sink
				}
				// Update the node
				node.Inner = xnode
			}
		}
	}
}

// Split a trapezoid horizontally, and replace its sink with a y node. node.Inner must be a sink
func (graph *QueryGraph) SplitTrapezoidHorizontally(node *QueryNode, point *Point) {
	sink := node.Inner.(SinkNode)
	fmt.Printf("Splitting trapezoid %s horizontally at %v\n", sink.Trapezoid.String(), point)
	top := new(Trapezoid)
	bottom := new(Trapezoid)
	origTop := sink.Trapezoid.Top
	origBottom := sink.Trapezoid.Bottom
	if origTop != nil && origTop.Below(point) {
		panic("cannot split on point above top")
	}
	if origBottom != nil && origBottom.Above(point) {
		panic("cannot split on point below bottom")
	}

	// Duplicate and adjust
	*top = *sink.Trapezoid
	*bottom = *sink.Trapezoid

	// Create the dividing line at the point's Y value
	top.Bottom = point
	bottom.Top = point

	// Set neighbors. The top trapezoid retains the upper neighbors, and the
	// bottom trapezoid retains the lower neighbors
	top.TrapezoidsBelow = TrapezoidNeighborList{bottom}
	bottom.TrapezoidsAbove = TrapezoidNeighborList{top}

	top.Sink = &QueryNode{SinkNode{Trapezoid: top, InitialParent: node}}
	bottom.Sink = &QueryNode{SinkNode{Trapezoid: bottom, InitialParent: node}}

	// Back link neighbors
	for _, neighbor := range top.TrapezoidsAbove {
		if neighbor != nil {
			neighbor.TrapezoidsBelow.ReplaceOrAdd(sink.Trapezoid, top)
		}
	}
	for _, neighbor := range bottom.TrapezoidsBelow {
		if neighbor != nil {
			neighbor.TrapezoidsAbove.ReplaceOrAdd(sink.Trapezoid, bottom)
		}
	}

	// Create the new sink nodes, replacing the original trapezoid's sink
	node.Inner = YNode{
		Key:   point,
		Above: top.Sink,
		Below: bottom.Sink,
	}
	fmt.Println("Split into:", top.DbgName(), bottom.DbgName())
}

// Add a polygon to the graph. If the polygon winds clockwise, this will end up
// producing a hole. Otherwise, it will be filled. The polygon must not
// intersect any existing segments in the graph.
//
// By default, this process is pseudorandom, but deterministic. This is because
// predictable results are easier to debug. However, this raises the potential
// for adversarial inputs. If you are using untrusted input, you should pass
// "true" for proper randomization.
func (graph *QueryGraph) AddPolygon(poly Polygon, nondeterministic ...bool) {
	var seed int64
	if len(nondeterministic) > 0 && nondeterministic[0] {
		// TODO: We should make an adapter for crypto/random, and secure random
		// numbers when nondeterministic mode is selected. Low priority, as it would
		// be quite difficult to construct an input on the fly that would cause
		// pathological performance based on a time based seed.
		seed = time.Now().UnixNano()
	}
	source := rand.NewSource(seed)
	r := rand.New(source)
	// Create the segments
	segments := make([]*Segment, 0, len(poly.Points))
	for i := 0; i < len(poly.Points); i++ {
		segments = append(segments, &Segment{poly.Points[i], poly.Points[(i+1)%len(poly.Points)]})
	}

	// Shuffle the segments. This is what gives us expected O(nlogn) time
	r.Shuffle(len(segments), func(i, j int) {
		segments[i], segments[j] = segments[j], segments[i]
	})

	// If this is an empty graph, initialize with the first segment
	if graph.Root == nil {
		fmt.Println("Adding segment", *segments[0], dbg.Name(segments[0]))
		newGraph := NewQueryGraph(segments[0])
		segments = segments[1:]
		*graph = *newGraph
	}

	// Add the segments
	//
	// TODO: Add the preprocessing step which finds new search roots for every
	// point. That step will make the algorithm O(nlog*n)
	for _, segment := range segments {
		graph.dbgDraw(100)
		fmt.Println("Adding segment", *segment, dbg.Name(segment))
		graph.AddSegment(segment)
	}
	graph.dbgDraw(100)

}

// Fast test for point-in-polygon using the trapezoid graph. Output is not
// defined for points exactly on the edge of the graph.
func (g *QueryGraph) ContainsPoint(point *Point) bool {
	// Find the trapezoid containing the point
	containingTrapezoid := g.FindPoint(point, DefaultDirection)
	if containingTrapezoid == nil {
		return false
	}

	// Check if the trapezoid is inside
	return containingTrapezoid.Inner.(SinkNode).Trapezoid.IsInside()
}
