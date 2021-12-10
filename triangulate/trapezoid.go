package triangulate

import (
	"fmt"
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

// Check if a segment crosses the bottom edge of the trapezoid. If the segment
// is horizontal, this always returns false.
func (t *Trapezoid) BottomIntersectsSegment(segment *Segment) bool {
	if segment.IsHorizontal() {
		// The only way a horizontal segment could intersect the bottom of a
		// trapezoid would be if segments crossed. This is assumed never to be the
		// case.
		return false
	}
	if t.Bottom == nil { // Bottom is at infinity, nothing can intersect it
		return false
	}

	// Find the x value for the segment at the bottom of the trapezoid
	x := segment.SolveForX(t.Bottom.Y)
	point := &Point{x, t.Bottom.Y}
	fmt.Println("intersection point:", *point)
	fmt.Println("Left segment", t.Left)
	fmt.Println("Right segment", t.Right)

	return t.Left.IsLeftOf(point) && t.Right.IsRightOf(point)
}

// Split a trapezoid vertically with a segment, returning the two trapezoids. It
// is assumed that the segment fully passes through the trapezoid. The resulting
// left and right trapezoids will not yet be in the query graph, and they will
// still point to the original trapezoid's sink. This must be fixed after
// trapezoids with agreeing edges are merged.
func (t *Trapezoid) SplitBySegment(segment *Segment) (left, right *Trapezoid) {
	fmt.Println("Splitting trapezoid on segment", t.String())
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

	var topX, bottomX float64

	if segment.IsHorizontal() {
		// If the segment is horizontal, then it has infinitesimally positive slope,
		// which means that we can only go by top or bottom according to the
		// lexicographic ordering. There is no meaninful interpolation between these
		// x values depending on the y value.
		topX = segment.Top().X
		bottomX = segment.Bottom().X
	} else {
		topX = segment.SolveForX(t.Top.Y)
		bottomX = segment.SolveForX(t.Bottom.Y)
	}

	top := &Point{
		X: topX,
		Y: t.Top.Y,
	}
	bottom := &Point{
		X: bottomX,
		Y: t.Bottom.Y,
	}

	for _, neighbor := range t.TrapezoidsAbove {
		if neighbor == nil {
			continue
		}
		// Remove the old trapezoid as a neighbor
		neighbor.TrapezoidsBelow.Remove(t)
		if !left.IsDegenerateOnSide(Up) {
			// Check if the top of the segment is right of the neighbor's left edge.
			// If so, it is an above neighbor to the left split. (left edge at infinity
			// is left of everything)
			if neighbor.Left == nil || neighbor.Left.IsLeftOf(top) {
				left.TrapezoidsAbove.Add(neighbor)
				neighbor.TrapezoidsBelow.Add(left)
			}
		} else {
			fmt.Println("top side of", left.String(), "is degenerate, no above neighbors added")
		}

		if !right.IsDegenerateOnSide(Up) {
			// Check if the top of the segment is left of the neighbor's right edge.
			// If so, it's a neighbor of the right split. (right edge at infinity is
			// right of everything)
			if neighbor.Right == nil || neighbor.Right.IsRightOf(top) {
				right.TrapezoidsAbove.Add(neighbor)
				neighbor.TrapezoidsBelow.Add(right)
			}
		} else {
			fmt.Println("top side of", right.String(), "is degenerate, no above neighbors added")
		}

	}

	for _, neighbor := range t.TrapezoidsBelow {
		if neighbor == nil {
			continue
		}
		neighbor.TrapezoidsAbove.Remove(t)
		if !left.IsDegenerateOnSide(Down) {
			// Check if the bottom of the segment is right of the neighbor's left
			// edge. If so, it's a below neighbor to the left split.
			if neighbor.Left == nil || neighbor.Left.IsLeftOf(bottom) {
				left.TrapezoidsBelow.Add(neighbor)
				neighbor.TrapezoidsAbove.Add(left)
			}
		} else {
			fmt.Println("bottom side of", left.String(), "is degenerate, no below neighbors added")
		}

		if !right.IsDegenerateOnSide(Down) {
			// Check if the bottom of the segment is left of the neighbor's right edge.
			// If so, it's a neighbor of the right split.
			if neighbor.Right == nil || neighbor.Right.IsRightOf(bottom) {
				right.TrapezoidsBelow.Add(neighbor)
				neighbor.TrapezoidsAbove.Add(right)
			}
		} else {
			fmt.Println("bottom side of", right.String(), "is degenerate, no below neighbors added")
		}
	}
	fmt.Println("Left:", left.String())
	fmt.Println("Right:", right.String())
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
// corresponding segment endpoints are equal IFF the corresponding side of the trapezoid is that segment's start or end.
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
