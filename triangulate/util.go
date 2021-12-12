package triangulate

import (
	"fmt"
	"math"
)

const Epsilon = 1e-6

// To compensate for imprecision in floats, equality is tolerance based. If we
// don't account for this, we'll end up shaving off absurdly thin triangles on nearly
// horizontal segments.
func Equal(a, b float64) bool {
	return math.Abs(a-b) < Epsilon
}

func GreaterThan(a, b float64) bool {
	return a-b > Epsilon
}

func LessThan(a, b float64) bool {
	return b-a > Epsilon
}

// A common convention in our geometry is that if two points have the same Y
// value, the one with the smallex X value is "lower". This simulates a slightly
// rotated coordinate system, allowing us to assume Y values are never equal.
func (p *Point) Below(otherPoint *Point) bool {
	if Equal(p.Y, otherPoint.Y) {
		return p.X < otherPoint.X
	}
	return p.Y < otherPoint.Y
}

func (p *Point) Above(otherPoint *Point) bool {
	return !p.Below(otherPoint)
}

// Create a directional point pointing at another point
func (p *Point) PointingAt(other *Point) DirectionalPoint {
	dir := Vector{
		X: other.X - p.X,
		Y: other.Y - p.Y,
	}.Normalize()

	return DirectionalPoint{
		Point:     p,
		Direction: dir,
	}
}

// Convenience function setting the convention for finding points when you don't care about the direction.
func (p *Point) PointingRight() DirectionalPoint {
	return DirectionalPoint{
		Point:     p,
		Direction: Vector{X: 1, Y: 0},
	}
}

// Another convenience function, same concept as PointingRight
func DefaultDirectionalPoint(x, y float64) DirectionalPoint {
	return DirectionalPoint{
		Point:     &Point{X: x, Y: y},
		Direction: Vector{X: 1, Y: 0},
	}
}

// Often we want to treat an array as a circular buffer. This gives the modular
// index given length n, but unlike the raw modulo operator, it only gives positive values
func CircularIndex(i, n int) int {
	return (i%n + n) % n
}

func (s *PointStack) Push(p *Point) {
	*s = append(*s, p)
}

func (s *PointStack) Pop() *Point {
	if len(*s) == 0 {
		return nil
	}
	p := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return p
}

func (s *PointStack) Peek() *Point {
	if len(*s) == 0 {
		return nil
	}
	return (*s)[len(*s)-1]
}

func (s *PointStack) Empty() bool {
	return len(*s) == 0
}

// Several properties can be derived from any structure that can compute its
// signed area.
type HasSignedArea interface {
	// Enclosed area of the structure, positive if counterclockwise, negative if clockwise.
	SignedArea() float64
}

func (t *Triangle) SignedArea() float64 {
	return ((t.A.X*t.B.Y - t.B.X*t.A.Y) +
		(t.B.X*t.C.Y - t.C.X*t.B.Y) +
		(t.C.X*t.A.Y - t.A.X*t.C.Y)) / 2
}

func (poly *Polygon) SignedArea() float64 {
	area := 0.0
	n := len(poly.Points)
	for i := 0; i < n; i++ {
		nextI := (i + 1) % n
		area += poly.Points[i].X*poly.Points[nextI].Y - poly.Points[nextI].X*poly.Points[i].Y
	}
	return area / 2
}

func Area(s HasSignedArea) float64 {
	return math.Abs(s.SignedArea())
}

func IsCCW(s HasSignedArea) bool {
	return s.SignedArea() > 0
}

func IsCW(s HasSignedArea) bool {
	return s.SignedArea() < 0
}

func (ps PointSet) Contains(p *Point) bool {
	_, ok := ps[p]
	return ok
}

func (ps PointSet) Add(p *Point) {
	ps[p] = struct{}{}
}

func (ps PointSet) Equals(otherSet PointSet) bool {
	if len(ps) != len(otherSet) {
		return false
	}
	for p := range ps {
		if !otherSet.Contains(p) {
			return false
		}
	}
	return true
}

// String functions
func (p *Point) String() string {
	return fmt.Sprintf("{%0.2f, %0.2f}", p.X, p.Y)
}

// A segment points down if its start point is above its endpoint
func (s *Segment) PointsDown() bool {
	return s.End.Below(s.Start)
}

// Is the line segment left of p. This assumes that P is vertically between the start and end of the segment
func (s *Segment) IsLeftOf(p *Point) bool {
	if s == nil {
		return true
	}
	// Handle horizontal case
	if Equal(s.Start.Y, s.End.Y) {
		return LessThan(s.Bottom().X, p.X)
	}

	if s.Start == p || s.End == p {
		return false
	}

	x := s.SolveForX(p.Y)
	return LessThan(x, p.X)
}

func (s *Segment) IsRightOf(p *Point) bool {
	if s == nil {
		return true
	}
	// Handle horizontal case
	if Equal(s.Start.Y, s.End.Y) {
		return GreaterThan(s.Start.X, p.X) && GreaterThan(p.X, s.End.X)
	}

	if s.Start == p || s.End == p {
		return false
	}

	x := s.SolveForX(p.Y)
	return GreaterThan(x, p.X)
}

// Determine which direction the segment points from top to bottom
/*
      o
    /   <- Left
  o

	o
	 \  <- Right
	  o
*/
func (s *Segment) XDirection() XDirection {
	top := s.Top()
	bottom := s.Bottom()
	if top.X > bottom.X {
		return Left
	} else {
		return Right
	}
}

func (s *Segment) Top() *Point {
	if s == nil {
		return nil
	}

	if s.PointsDown() {
		return s.Start
	}
	return s.End
}

func (s *Segment) Bottom() *Point {
	if s == nil {
		return nil
	}

	if s.PointsDown() {
		return s.End
	}
	return s.Start
}

func (dir XDirection) Opposite() XDirection {
	return dir ^ 1
}

func (dir YDirection) Opposite() YDirection {
	return dir ^ 1
}

func (dir Direction) Opposite() Direction {
	return Direction{dir.X.Opposite(), dir.Y.Opposite()}
}

func (s *Segment) IsHorizontal() bool {
	return Equal(s.Start.Y, s.End.Y)
}

func (s *Segment) IsVertical() bool {
	return Equal(s.Start.X, s.End.X)
}

// Solve the line (ignoring the bounds) for the given y value
func (s *Segment) SolveForX(y float64) float64 {
	if s.IsHorizontal() {
		panic("Cannot solve for X on a horizontal segment")
	}
	if s.IsVertical() { // Special case; no slope
		return s.Start.X
	}

	m := (s.End.Y - s.Start.Y) / (s.End.X - s.Start.X)
	b := s.Start.Y - m*s.Start.X
	return (y - b) / m
}

func (v Vector) Normalize() Vector {
	return Vector{
		X: v.X / v.Length(),
		Y: v.Y / v.Length(),
	}
}

func (v Vector) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}
