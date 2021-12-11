package triangulate

type Polygon struct {
	Points []*Point
}

// Winding rule point-in-polygon. This is provided primarily for testing of the
// Seidel algorithm. If you are checking many points inside the same large
// polygon, it can be more effficient to trapezoidize it and use the resulting
// QueryGraph.
func (poly Polygon) ContainsPointByEvenOdd(p *Point) bool {
	return poly.CrossingCount(p)%2 == 1
}

// Crossing count helper for even odd rule
func (poly Polygon) CrossingCount(p *Point) int {
	crossingCount := 0
	for i, vertex := range poly.Points {
		nextVertex := poly.Points[CircularIndex(i+1, len(poly.Points))]

		segment := Segment{vertex, nextVertex}
		if segment.IsRightOf(p) && vertex.Below(p) != nextVertex.Below(p) {
			crossingCount++
		}
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
