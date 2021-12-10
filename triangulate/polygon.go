package triangulate

type Polygon struct {
	Points []*Point
}

// Winding rule point-in-polygon. This is provided primarily for testing of the
// Seidel algorithm. If you are checking many points inside the same large
// polygon, it can be more effficient to trapezoidize it and use the resulting
// QueryGraph.
func (poly Polygon) ContainsPointByWinding(p *Point) bool {
	var winding int
	for i, vertex := range poly.Points {
		nextVertex := poly.Points[CircularIndex(i+1, len(poly.Points))]

		segment := Segment{vertex, nextVertex}
		if segment.IsHorizontal() {
			continue
		}
		x := segment.SolveForX(p.Y) // crossing point if applicable
		if p.X < x {                // Left side of downward crossing
			if vertex.Below(p) && nextVertex.Above(p) { // Upward crossing
				winding++
			} else if vertex.Above(p) && nextVertex.Below(p) { // Downward crossing
				winding--
			}
		}
	}
	return winding != 0
}

func (poly Polygon) Reverse() Polygon {
	newPoly := Polygon{}
	for i := len(poly.Points) - 1; i >= 0; i-- {
		newPoly.Points = append(newPoly.Points, poly.Points[i])
	}
	return newPoly
}
