package triangulate

type Polygon struct {
	Points []*Point
}

type Point struct {
	X float64
	Y float64
}

// Note that all points involved with the triangulation are pointers. This means
// they can be used as keys. We should never modify a point value from the
// original polygon, since some applications require exact equality, and we
// cannot tolerate loss of precision.
type Segment struct {
	Start *Point
	End   *Point
}

type Triangle struct {
	A, B, C *Point
}

type PointStack []*Point

type PointSet map[*Point]struct{}
