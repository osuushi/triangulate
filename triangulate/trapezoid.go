package triangulate

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

// Check if a segment crosses the bottom edge of the trapezoid. The segment must
// not be horizontal. Note that horizontal segments should never need to ask
// this question, since they never get past the first iteration when searching
// for trapezoids to split vertically.
//
// Furthermore, this assumes that that the segment does pass through the Y value
// of the bottom of the trapezoid. Again, this is always true during the scan.
func (t *Trapezoid) BottomIntersectsSegment(segment *Segment) bool {
	if segment.IsHorizontal() {
		panic("horizontal segments should never be tested for bottom intersection")
	}

	// Find the x value for the segment at the bottom of the trapezoid
	x := segment.SolveForX(t.BottomY)
	point := &Point{x, t.BottomY}

	return t.Left.IsLeftOf(point) && !t.Right.IsLeftOf(point)
}

func appendTrapezoidNeighbor(neighbors *[2]*Trapezoid, trapezoid *Trapezoid) {
	for i := 0; i < 2; i++ {
		if neighbors[i] == nil {
			neighbors[i] = trapezoid
			return
		}
	}
	panic("too many neighbors")
}

func replaceTrapezoidNeighborOrAppend(neighbors *[2]*Trapezoid, orig *Trapezoid, replacement *Trapezoid) {
	for i := 0; i < 2; i++ {
		if neighbors[i] == orig {
			neighbors[i] = replacement
			return
		}
	}
	// We didn't replace, so we must need to append
	appendTrapezoidNeighbor(neighbors, replacement)
}

// Split a trapezoid vertically with a segment, returning the two trapezoids. It
// is assumed that the segment fully passes through the trapezoid. The resulting
// left and right trapezoids will not yet be in the query graph, and they will
// still point to the original trapezoid's sink. This must be fixed after
// trapezoids with agreeing edges are merged.
func (t *Trapezoid) SplitBySegment(segment *Segment) (left, right *Trapezoid) {
	// Make duplicates and adjust them
	left = new(Trapezoid)
	right = new(Trapezoid)
	*left = *t
	*right = *t
	left.Right = segment
	right.Left = segment

	// Clear neighbors
	left.TrapezoidsAbove = [2]*Trapezoid{}
	left.TrapezoidsBelow = [2]*Trapezoid{}
	right.TrapezoidsAbove = [2]*Trapezoid{}
	right.TrapezoidsBelow = [2]*Trapezoid{}

	// Adjust neighbors
	top := segment.Top()
	bottom := segment.Bottom()
	for i := 0; i < 2; i++ {
		neighbor := t.TrapezoidsAbove[i]
		if neighbor != nil {
			// Check if the top of the segment is right of the neighbor's left edge.
			// If so, it is an above neighbor to the left split.
			if neighbor.Left.IsLeftOf(top) {
				appendTrapezoidNeighbor(&left.TrapezoidsAbove, neighbor)
				replaceTrapezoidNeighborOrAppend(&neighbor.TrapezoidsBelow, t, left)
			}

			// Check if the top of the segment is left of the neighbor's right edge.
			// If so, it's a neighbor of the right split.
			if !neighbor.Right.IsLeftOf(top) {
				appendTrapezoidNeighbor(&right.TrapezoidsAbove, neighbor)
				replaceTrapezoidNeighborOrAppend(&neighbor.TrapezoidsBelow, t, right)
			}
		}

		neighbor = t.TrapezoidsBelow[i]
		if neighbor != nil {
			// Check if the bottom of the segment is right of the neighbor's left
			// edge. If so, it's a below neighbor to the left split.
			if neighbor.Left.IsLeftOf(bottom) {
				appendTrapezoidNeighbor(&left.TrapezoidsBelow, neighbor)
				replaceTrapezoidNeighborOrAppend(&neighbor.TrapezoidsAbove, t, left)
			}

			// Check if the bottom of the segment is left of the neighbor's right edge.
			// If so, it's a neighbor of the right split.
			if !neighbor.Right.IsLeftOf(bottom) {
				appendTrapezoidNeighbor(&right.TrapezoidsBelow, neighbor)
				replaceTrapezoidNeighborOrAppend(&neighbor.TrapezoidsAbove, t, right)
			}
		}
	}
	return left, right
}

func (t *Trapezoid) CanMergeWith(other *Trapezoid) bool {
	return t.Left == other.Left && t.Right == other.Right
}
