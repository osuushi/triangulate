package triangulate

import "math"

const Tolerance = 1e-6

// To compensate for imprecision in floats, equality is tolerance based. If we
// don't account for this, we'll end up shaving off absurdly thin triangles on nearly
// horizontal segments.
func Equal(a, b float64) bool {
	return math.Abs(a-b) < Tolerance
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
