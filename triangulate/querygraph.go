package triangulate

import (
	"fmt"
	"strings"
)

// This implements the data structures for Seidel 1991 for trapezoidizing a non-monotone polygon
// into multiple segments. It uses the same lexicographic convention as
// elsewhere which avoids equal y values by lexicographic rotation.

type Direction int

const (
	Left Direction = iota
	Right
)

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

func (graph *QueryGraph) AddSegment(segment *Segment) {
	top := segment.Top()
	bottom := segment.Bottom()

	// bottom := segment.Bottom()
	direction := segment.Direction()

	// Find the node that contains the top point, coming from the bottom
	node := graph.Root.FindPoint(top, direction.Opposite())

	var topTrapezoid = node.Inner.(SinkNode).Trapezoid
	// Check if we're not a bottom endpoint of the trapezoid segment. Note that we
	// can't be a top endpoint, since line segments don't overlap. // TODO: Is this true? What about  |  /|
	if topTrapezoid.Left.Bottom() != top && topTrapezoid.Right.Bottom() != top {
		graph.SplitTrapezoidHorizontally(node, top)
	}

	// Do the same process for the bottom point
	node = node.FindPoint(bottom, direction)
	var bottomTrapezoid = node.Inner.(SinkNode).Trapezoid
	fmt.Println("Splitting bottom trapezoid:", bottomTrapezoid.String())

	// This time, we need to check if we're the top endpoint of the trapezoid's segments.
	if bottomTrapezoid.Left.Top() != bottom && bottomTrapezoid.Right.Top() != bottom {
		graph.SplitTrapezoidHorizontally(node, bottom)
		// We now want the top sink trapezoid, since the line segment crosses that.
		bottomTrapezoid = node.Inner.(YNode).Above.Inner.(SinkNode).Trapezoid
	}

	fmt.Println("Trapezoids after adding segment bottom:")
	graph.PrintAllTrapezoids()
	fmt.Println()

	// Split the trapezoids that intersect the line segment. Note at this point
	// that `top` sits exactly on top of the top trapezoid, and `bottom` sits
	// exactly on the bottom of the bottom trapezoid.
	curTrapezoid := bottomTrapezoid

	fmt.Println("Bottom trapezoid", bottomTrapezoid.String())

	var leftTrapezoids []*Trapezoid
	var rightTrapezoids []*Trapezoid
	for { // Loop over the trapezoids
		// Split this trapezoid horizontally
		leftTrapezoid, rightTrapezoid := curTrapezoid.SplitBySegment(segment)
		leftTrapezoids = append(leftTrapezoids, leftTrapezoid)
		rightTrapezoids = append(rightTrapezoids, rightTrapezoid)

		// Find the next trapezoid out of the up to two neighbors above this one. It
		// will be the one whose bottom the line segment intersects

		// Check single neighbor cases
		var singleNeighbor bool
		for i, neighbor := range curTrapezoid.TrapezoidsAbove {
			if neighbor == nil {
				singleNeighbor = true
				curTrapezoid = curTrapezoid.TrapezoidsAbove[i^1] // choose other neighbor
				break
			}
		}

		if !singleNeighbor {
			// Check the first neighbor to see if it intersects the line segment. Only
			// one of them can
			neighbor := curTrapezoid.TrapezoidsAbove[0] // we pick the first neighbor arbitrarily

			// Note that we only have to check that the bottom intersects, since we're scanning from below
			if neighbor.BottomIntersectsSegment(segment) {
				curTrapezoid = neighbor
			} else {
				curTrapezoid = curTrapezoid.TrapezoidsAbove[1]
			}
		}

		// All of the above assumed we actually pass the trapezoid's bottom in the
		// vertical direction. If we didn't, we break here.
		if top.Below(curTrapezoid.Bottom) {
			break
		}
	}

	// We now have left and right chains of triangles that were split by the line
	// segment, but some of them may share edges, so we need to merge them. All of
	// the left trapezoids have the segment as a right edge and vice versa, so we
	// can treat each chain of trapezoids separately

	for i, chain := range [2][]*Trapezoid{leftTrapezoids, rightTrapezoids} {
		side := Direction(i)
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
			*mergedTrapezoid = *chunk[0]
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

			// Note that we can't set an initial parent on the new sink, because
			// (assuming there's more than one trapezoid in the chunk), the node will
			// have multiple XNode parents.
			mergedTrapezoid.Sink = &QueryNode{SinkNode{Trapezoid: mergedTrapezoid}}

			// Change every SinkNode to XNode, or complete the XNode depending on direction
			for _, trapezoid := range chunk {
				node := trapezoid.Sink
				var xnode XNode
				fmt.Printf("Updating sink: %T\n", node.Inner)
				if side == Left { // On left side, we're making a new XNode
					xnode = XNode{
						Left: mergedTrapezoid.Sink,
					}
				} else { // On right side, we created the xnode, so we just need to pull it off
					xnode = node.Inner.(XNode)
					xnode.Right = mergedTrapezoid.Sink
				}
				// Update the node
				node.Inner = xnode
			}
		}
	}

	fmt.Println("Trapezoids after merging trapezoids:")
	graph.PrintAllTrapezoids()
}

// Split a trapezoid horizontally, and replace its sink with a y node. node.Inner must be a sink
func (graph *QueryGraph) SplitTrapezoidHorizontally(node *QueryNode, point *Point) {
	sink := node.Inner.(SinkNode)
	top := new(Trapezoid)
	bottom := new(Trapezoid)

	// Duplicate and adjust
	*top = *sink.Trapezoid
	*bottom = *sink.Trapezoid

	// Create the dividing line at the point's Y value
	top.Bottom = point
	bottom.Top = point

	// Set neighbors. The top trapezoid retains the upper neighbors, and the
	// bottom trapezoid retains the lower neighbors
	top.TrapezoidsBelow = [2]*Trapezoid{bottom}
	bottom.TrapezoidsAbove = [2]*Trapezoid{top}

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

	// Create the new sink nodes
	node.Inner = YNode{
		Key:   point,
		Above: top.Sink,
		Below: bottom.Sink,
	}
}
