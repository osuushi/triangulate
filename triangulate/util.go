package triangulate

import "math"

const Epsilon = 1e-6

// To compensate for imprecision in floats, equality is tolerance based. If we
// don't account for this, we'll end up shaving off absurdly thin triangles on nearly
// horizontal segments.
func Equal(a, b float64) bool {
	return math.Abs(a-b) < Epsilon
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

// Several properties can be derived from any structure that can compute its
// signed area.
type HasSignedArea interface {
	// Enclosed area of the structure, positive if counterclockwise, negative if clockwise.
	SignedArea() float64
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
