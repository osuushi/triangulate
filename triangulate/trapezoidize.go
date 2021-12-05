package triangulate

import (
	"math"
)

// This implements the data structures for Seidel 1991 for trapezoidizing a non-monotone polygon
// into multiple segments. It uses the same lexicographic convention as
// elsewhere which avoids equal y values by lexicographic rotation.

type Trapezoid struct {
	Left, Right                      *Segment
	TopY, BottomY                    float64       // y-coordinates of top and bottom of trapezoid
	TrapezoidsAbove, TrapezoidsBelow [2]*Trapezoid // Up to two neighbors in each direction
	Sink                             *QueryNode
}

// Is the trapezoid inside the polygon?
func (t *Trapezoid) IsInside() bool {
	// A trapezoid is inside the polygon iff it has both a right and left segment,
	// and the left segment points down. Note that this implies, for any valid
	// polygon, that the right side points up. Note also that a right-to-left
	// horizontal segment "points down" because of the lexicographic rotation.
	return t.Left != nil && t.Right != nil && t.Left.PointsDown()
}

type Direction int

const (
	Left Direction = iota
	Right
)

// Node for the query structure. The query structure allows us to navigate the
// trapezoid set efficiently, and can be built in O(nlog(n)) time. (TODO: There
// is a preprocessing loop you can use to get this to O(nlog*n) time. Implement
// this once tests are passing).
//
// This algorithm has been chosen because it has good asymptotic performance,
// and handles holes without special casing. In fact, it is rare in that you can
// do the entire process of splitting multiple discontinuous polygons with holes
// without even providing the polygons as a connected set. All you need are line
// segments and a consistent winding rule. This makes it prefect for processing
// 3D meshes where you might just have a pile of line segments that lie on a
// plane.

// Query nodes are polymorphic, and we need to be able to replace the content
// with a different node type in O(1) time. Therefore, we use this interface to
// provide a union between the different types of query node.
type QueryNodeInner interface {
	// Traverse the graph to find the sink whose trapezoid contains the point. The
	// direction argument is required to disambiguate when the point is an XNode
	// segment's endpoint.
	FindPoint(*Point, Direction) *QueryNode

	// Child nodes is useful for iterating over a graph
	ChildNodes() []*QueryNode

	// This is a dummy method that ensures that *QueryNode is not a
	// QueryNodeInner. The method is unused, but is a hint to the type system that
	// will prevent accidental double-wrapping, or otherwise using *QueryNode
	// where it doesn't belong.
	queryModeInnerTypeHint()
}

// QueryModeInner types enumerated here with type hint
func (SinkNode) queryModeInnerTypeHint() {}
func (YNode) queryModeInnerTypeHint()    {}
func (XNode) queryModeInnerTypeHint()    {}

type QueryNode struct {
	Inner QueryNodeInner
}

func (n *QueryNode) FindPoint(p *Point, dir Direction) *QueryNode {
	// If we found a sink node, we're done
	if _, ok := n.Inner.(SinkNode); ok {
		return n
	}

	// For other node types, ask the inner node to search its children
	return n.Inner.FindPoint(p, dir)
}

func (n *QueryNode) ChildNodes() []*QueryNode {
	return n.Inner.ChildNodes()
}

type SinkNode struct {
	Trapezoid *Trapezoid
	// Before a sink has been merged, it will always have a single parent, which
	// this points to. After a merge, we no longer need to know the parent, and
	// this will be nil.
	InitialParent *QueryNode
}

func (node SinkNode) FindPoint(point *Point, _ Direction) *QueryNode {
	// If we're at a sink, we can't traverse any further.
	panic("Should not try to find point from a sink")
}

func (node SinkNode) ChildNodes() []*QueryNode {
	return nil
}

// A Y Node is a node which lets us navigate up or down
type YNode struct {
	Above, Below *QueryNode
	Key          *Point // Point so that we can do the lexicographic thing
}

func (node YNode) FindPoint(point *Point, dir Direction) *QueryNode {
	if point.Below(node.Key) {
		return node.Below.FindPoint(point, dir)
	} else {
		return node.Above.FindPoint(point, dir)
	}
}

func (node YNode) ChildNodes() []*QueryNode {
	return []*QueryNode{node.Above, node.Below}
}

// An X node
type XNode struct {
	Left, Right *QueryNode
	Key         *Segment
}

func (node XNode) FindPoint(point *Point, dir Direction) *QueryNode {
	// First check if it's an endpoint. If so, we use dir to determine which way to go.
	if node.Key.Start == point || node.Key.End == point {
		switch dir {
		case Left:
			return node.Left.FindPoint(point, dir)
		case Right:
			return node.Right.FindPoint(point, dir)
		}
	}

	if node.Key.IsLeftOf(point) {
		return node.Right.FindPoint(point, dir)
	} else {
		return node.Left.FindPoint(point, dir)
	}
}

func (node XNode) ChildNodes() []*QueryNode {
	return []*QueryNode{node.Left, node.Right}
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

// Create a channel using a go routine to iterate over the graph. This provides
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
func NewQueryGraph(segment *Segment) *QueryNode {

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
		Left:    nil,
		Right:   nil,
		TopY:    math.Inf(1),
		BottomY: a.Y,
	}

	top.Sink = &QueryNode{SinkNode{Trapezoid: top}}

	left := &Trapezoid{
		Left:    nil,
		Right:   segment,
		TopY:    a.Y,
		BottomY: b.Y,
	}
	left.Sink = &QueryNode{SinkNode{Trapezoid: left}}

	right := &Trapezoid{
		Left:    segment,
		Right:   nil,
		TopY:    a.Y,
		BottomY: b.Y,
	}
	right.Sink = &QueryNode{SinkNode{Trapezoid: right}}

	bottom := &Trapezoid{
		Left:    nil,
		Right:   nil,
		TopY:    b.Y,
		BottomY: math.Inf(-1),
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
	return graph
}

// func (g *QueryGraph) AddSegment(segment *Segment) {
// 	top := segment.Top()
// 	bottom := segment.Bottom()
// 	direction := segment.Direction()
// 	// Find the node that contains the top point, coming from the bottom
// 	node := g.Root.FindPoint(top, direction.Opposite())
// 	switch node := node.(type) {
// 	case *SinkNode:
// 		// This means we landed inside an existing trapezoid. We will split it and replace its sink with a y node.

// 	case *XNode:
// 	default:
// 		panic("Unexpected node type in AddSegment")
// 	}
// }
