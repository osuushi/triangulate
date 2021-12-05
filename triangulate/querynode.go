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
