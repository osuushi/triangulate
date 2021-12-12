package triangulate

type Polygon struct {
	Points []*Point
}

// A list of polygons which must not have crossing segments. For the purposes of
// Seidel triangulation, holes must run clockwise, and the outer polygons must
// run counter-clockwise.
type PolygonList []Polygon

// Winding rule point-in-polygon. This is provided primarily for testing of the
// Seidel algorithm. If you are checking many points inside the same large
// polygon, it can be more effficient to trapezoidize it and use the resulting
// QueryGraph.
//
// Note that this is winding direction agnostic, so it will give a different
// answer from the Seidel algorithm if you add counter-clockwise holes, or
// clockwise outer polygons.
func (poly Polygon) ContainsPointByEvenOdd(p *Point) bool {
	return poly.CrossingCount(p)%2 == 1
}

// Crossing count helper for even odd rule
func (poly Polygon) CrossingCount(p *Point) int {
	crossingCount := 0
	for i, vertex := range poly.Points {
		nextVertex := poly.Points[CircularIndex(i+1, len(poly.Points))]

		segment := Segment{vertex, nextVertex}
		if !segment.IsLeftOf(p) && vertex.Below(p) != nextVertex.Below(p) {
			crossingCount++
		}
	}
	return crossingCount
}

func (l PolygonList) ContainsPointByEvenOdd(p *Point) bool {
	return l.CrossingCount(p)%2 == 1
}

func (l PolygonList) CrossingCount(p *Point) int {
	crossingCount := 0
	for _, poly := range l {
		crossingCount += poly.CrossingCount(p)
	}
	return crossingCount
}

func (poly Polygon) Reverse() Polygon {
	newPoly := Polygon{}
	for i := len(poly.Points) - 1; i >= 0; i-- {
		newPoly.Points = append(newPoly.Points, poly.Points[i])
	}
	return newPoly
}
