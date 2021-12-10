package triangulate

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
	FindPoint(DirectionalPoint) *QueryNode

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

func (n *QueryNode) FindPoint(dp DirectionalPoint) *QueryNode {
	// If we found a sink node, we're done
	if _, ok := n.Inner.(SinkNode); ok {
		return n
	}

	// For other node types, ask the inner node to search its children
	return n.Inner.FindPoint(dp)
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

func (node SinkNode) FindPoint(_ DirectionalPoint) *QueryNode {
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

func (node YNode) FindPoint(dp DirectionalPoint) *QueryNode {
	var direction YDirection
	// For equal points, we must use the direction given
	// Note that this only applies when directly comparing vertices, so pointer
	// comparison is fine.
	if node.Key == dp.Point {
		// Find the direction from the direction vector
		if Equal(dp.Direction.Y, 0) { // If horizontal, we need the lexicographic tiebreak
			if dp.Direction.X > 0 { // Slopes up from left to right
				direction = Up
			} else { // Slopes down from right to left
				direction = Down
			}
		} else if dp.Direction.Y > 0 {
			direction = Up
		} else {
			direction = Down
		}
	} else if dp.Point.Below(node.Key) {
		direction = Down
	} else {
		direction = Up
	}

	switch direction {
	case Up:
		return node.Above.FindPoint(dp)
	case Down:
		return node.Below.FindPoint(dp)
	}
	panic("no direction found") // should be unreachable
}

func (node YNode) ChildNodes() []*QueryNode {
	return []*QueryNode{node.Above, node.Below}
}

// An X node
type XNode struct {
	Left, Right *QueryNode
	Key         *Segment
}

func (node XNode) FindPoint(dp DirectionalPoint) *QueryNode {
	var direction XDirection

	// First check if it's an endpoint. If so, we use the direction vector to
	// decide what happens. There's a subtle point here: We are not asking if the
	// direction vector slopes left or right, but if it slopes _more_ left or
	// right than the node's key.
	if node.Key.Start == dp.Point || node.Key.End == dp.Point {
		// Since IsLeftOf doesn't actually care about the bounds of the segment (it
		// only tests the line through them), we can just add the direction vector
		// to the point and check if it's left of the node's segment key.
		nudgedPoint := &Point{
			X: dp.Point.X + dp.Direction.X,
			Y: dp.Point.Y + dp.Direction.Y,
		}
		if node.Key.IsLeftOf(nudgedPoint) {
			direction = Right
		} else { // Note that there is no middle here; that would imply overlapping line segments.
			direction = Left
		}
	} else if node.Key.IsLeftOf(dp.Point) {
		direction = Right
	} else {
		direction = Left
	}

	switch direction {
	case Left:
		return node.Left.FindPoint(dp)
	case Right:
		return node.Right.FindPoint(dp)
	}
	panic("no direction found") // should be unreachable
}

func (node XNode) ChildNodes() []*QueryNode {
	return []*QueryNode{node.Left, node.Right}
}
