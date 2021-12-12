package triangulate

import (
	"fmt"
	"math"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/osuushi/triangulate/dbg"
)

type Trapezoid struct {
	Left, Right *Segment
	// The top and bottom are points, although geometrically, you can think of
	// them as the y values of those points. There are two reasons that points
	// must be used instead of y values:
	//
	// 1. A critical assumption of the algorithm is that no two points lie on the
	// same horizontal. This is simulated by lexicographic ordering, but it means
	// that _every_ Y comparison must have an X value involved to break ties.
	//
	// 2. The time will come when we ask every trapezoid "what points on your
	// boundary are vertices of the polygon"? Because of the unique Y value
	// assumption, the answer is _always_ two points. Those two points are the top
	// and bottom fields. Note that in some cases, these will be an endpoint of a
	// segment, and in some cases, they'll lie on the top or bottom of the
	// trapezoid, away from the left and right sides.
	Top, Bottom                      *Point
	TrapezoidsAbove, TrapezoidsBelow TrapezoidNeighborList
	Sink                             *QueryNode
}

// Trapezoids can have up to two neighbors above and below them in the stable
// state, but while splitting, they can have up to three below. This should
// never be the case after splitting is complete.
type TrapezoidNeighborList [3]*Trapezoid

// Is the trapezoid inside the polygon?
func (t *Trapezoid) IsInside() bool {
	// A trapezoid is inside the polygon iff it has both a right and left segment,
	// and the left segment points down. Note that this implies, for any valid
	// polygon, that the right side points up. Note also that a right-to-left
	// horizontal segment "points down" because of the lexicographic rotation.
	return t.Left != nil && t.Right != nil && t.Left.PointsDown()
}

func (t *Trapezoid) SegmentForSide(side XDirection) *Segment {
	if side == Left {
		return t.Left
	}
	return t.Right
}

func (t *Trapezoid) xValueForDirection(dir Direction) float64 {
	segment := t.SegmentForSide(dir.X)
	// Nil side has infinite X value
	if segment == nil {
		if dir.X == Left {
			return math.Inf(-1)
		} else {
			return math.Inf(1)
		}
	}

	var boundaryPoint *Point
	if dir.Y == Up {
		boundaryPoint = t.Top
	} else {
		boundaryPoint = t.Bottom
	}
	if boundaryPoint == nil {
		panic("cannot get x value with no boundary point")
	}

	// In the horizontal case, there is no solving for Y. Horizontal segment edges can only be on one trapezoid
	if segment.IsHorizontal() {
		return boundaryPoint.X
	}
	return segment.SolveForX(boundaryPoint.Y)
}

// This is what decides if two trapezoids are neighbors.
func (bottomTrapezoid *Trapezoid) NonzeroOverlapWithTrapezoidAbove(topTrapezoid *Trapezoid) bool {
	// Get bottom extent for top trapezoid
	topMinX := topTrapezoid.xValueForDirection(Direction{Left, Down})
	topMaxX := topTrapezoid.xValueForDirection(Direction{Right, Down})
	// Get top extent for bottom trapezoid
	bottomMinX := bottomTrapezoid.xValueForDirection(Direction{Left, Up})
	bottomMaxX := bottomTrapezoid.xValueForDirection(Direction{Right, Up})

	// Find the overlapping range
	minX := math.Max(topMinX, bottomMinX)
	maxX := math.Min(topMaxX, bottomMaxX)

	// Determine if the size of the range is greater than zero
	return (maxX - minX) > Epsilon
}

// Check if a segment crosses the bottom edge of the trapezoid.
func (t *Trapezoid) BottomIntersectsSegment(segment *Segment) bool {
	if t.Bottom == nil { // Bottom is at infinity, nothing can intersect it
		return false
	}

	// Check the case where the bottom point of the trapezoid is an edge, and is
	// the endpoint of the segment.
	if t.Bottom == segment.Start || t.Bottom == segment.End {
		if (t.Left != nil && t.Left.Bottom() == t.Bottom) || (t.Right != nil && t.Right.Bottom() == t.Bottom) {
			return false
		}
	}

	if segment.IsHorizontal() {
		panic("tried to intersect horizontal segment with bottom")
	}

	// Find the x value for the segment at the bottom of the trapezoid
	x := segment.SolveForX(t.Bottom.Y)
	point := &Point{x, t.Bottom.Y}

	return t.Left.IsLeftOf(point) && t.Right.IsRightOf(point)
}

// Split a trapezoid vertically with a segment, returning the two trapezoids. It
// is assumed that the segment fully passes through the trapezoid. The resulting
// left and right trapezoids will not yet be in the query graph, and they will
// still point to the original trapezoid's sink. This must be fixed after
// trapezoids with agreeing edges are merged.
func (t *Trapezoid) SplitBySegment(segment *Segment) (left, right *Trapezoid) {
	fmt.Println("Split trapezoid:", t.String())
	// Make duplicates and adjust them
	left = new(Trapezoid)
	right = new(Trapezoid)
	*left = *t
	*right = *t
	left.Right = segment
	right.Left = segment

	// Clear neighbors
	left.TrapezoidsAbove = TrapezoidNeighborList{}
	left.TrapezoidsBelow = TrapezoidNeighborList{}
	right.TrapezoidsAbove = TrapezoidNeighborList{}
	right.TrapezoidsBelow = TrapezoidNeighborList{}

	// Adjust neighbors

	// First we need to know where the segment intersects the top and bottom of
	// the trapezoid we split.

	for _, neighbor := range t.TrapezoidsAbove {
		if neighbor == nil {
			continue
		}
		// Remove the old trapezoid as a neighbor
		neighbor.TrapezoidsBelow.Remove(t)

		if left.NonzeroOverlapWithTrapezoidAbove(neighbor) {
			left.TrapezoidsAbove.Add(neighbor)
			neighbor.TrapezoidsBelow.Add(left)
		}

		if right.NonzeroOverlapWithTrapezoidAbove(neighbor) {
			right.TrapezoidsAbove.Add(neighbor)
			neighbor.TrapezoidsBelow.Add(right)
		}
	}

	for _, neighbor := range t.TrapezoidsBelow {
		if neighbor == nil {
			continue
		}
		neighbor.TrapezoidsAbove.Remove(t)

		if neighbor.NonzeroOverlapWithTrapezoidAbove(left) {
			left.TrapezoidsBelow.Add(neighbor)
			neighbor.TrapezoidsAbove.Add(left)
		}

		if neighbor.NonzeroOverlapWithTrapezoidAbove(right) {
			right.TrapezoidsBelow.Add(neighbor)
			neighbor.TrapezoidsAbove.Add(right)
		}
	}
	fmt.Println("\tLeft trapezoid:", left.String())
	fmt.Println("\tRight trapezoid:", right.String())
	return left, right
}

func (t *Trapezoid) CanMergeWith(other *Trapezoid) bool {
	return t.Left == other.Left && t.Right == other.Right
}

// Check if the point is any of the (up to) six points involved with the
// trapezoid. If it is, then it's already a line segment in the graph.
func (t *Trapezoid) HasPoint(p *Point) bool {
	if t.Top == p || t.Bottom == p {
		return true
	}
	if t.Left != nil {
		if t.Left.Start == p || t.Left.End == p {
			return true
		}
	}
	if t.Right != nil {
		if t.Right.Start == p || t.Right.End == p {
			return true
		}
	}
	return false
}

// Check if the trapezoid has a degenerate side (is it a triangle). If either
// side is nil, then it's never degenerate. Otherwise, this holds when the
// corresponding segment endpoints are equal IFF the corresponding side of the
// trapezoid is that segment's start or end.
func (t *Trapezoid) IsDegenerateOnSide(side YDirection) bool {
	switch side {
	case Up:
		return t.Left != nil && t.Top == t.Left.Top() && t.Left.Top() == t.Right.Top()
	case Down:
		return t.Left != nil && t.Bottom == t.Left.Bottom() && t.Left.Bottom() == t.Right.Bottom()
	}
	panic("invalid side")
}

func (t *Trapezoid) String() string {
	return fmt.Sprintf("Trapezoid %s { ⬆ %s, ⬇ %s } <L: %s, R: %s, T: %s, B: %s>",
		t.DbgName(),
		t.TrapezoidsAbove.String(),
		t.TrapezoidsBelow.String(),
		dbg.Name(t.Left),
		dbg.Name(t.Right),
		dbg.Name(t.Top),
		dbg.Name(t.Bottom),
	)
}

func (t *Trapezoid) DbgName() string {
	// If the trapezoid is infinite, color it orange
	name := dbg.Name(t)
	if t.Top == nil || t.Bottom == nil || t.Left == nil || t.Right == nil { // Infinite in some direction
		name = aurora.Cyan(name).String()
	} else if Equal(t.Top.Y, t.Bottom.Y) { // Zero height
		name = aurora.Red(name).String()
	} else {
		name = aurora.Green(name).String()
	}
	return name
}

func (tl *TrapezoidNeighborList) String() string {
	var parts []string
	for _, neighbor := range *tl {
		if neighbor != nil {
			parts = append(parts, dbg.Name(neighbor))
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

// Append a trapezoid to the list, if it isn't already there
func (tl *TrapezoidNeighborList) Add(t *Trapezoid) {
	for i, neighbor := range *tl {
		if neighbor == t {
			return
		}
		if neighbor == nil {
			(*tl)[i] = t
			return
		}
	}
	panic("too many neighbors")
}

func (tl *TrapezoidNeighborList) Remove(t *Trapezoid) {
	for i, neighbor := range *tl {
		if neighbor == t {
			(*tl)[i] = nil
			return
		}
	}
}

// Replace a trapezoid with another, or append it if the original isn't there
func (tl *TrapezoidNeighborList) ReplaceOrAdd(orig *Trapezoid, replacement *Trapezoid) {
	for i, neighbor := range *tl {
		if neighbor == orig {
			(*tl)[i] = replacement
			return
		}
	}
	// We didn't replace, so we must add
	tl.Add(replacement)
}

func (tl *TrapezoidNeighborList) AnyNeighbor() *Trapezoid {
	for _, neighbor := range *tl {
		if neighbor != nil {
			return neighbor
		}
	}
	return nil
}
