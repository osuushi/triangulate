package triangulate

import (
	"testing"
)

func TestTriangulateMonotone(t *testing.T) {
	// Triangles. These are currently special-cased, so these should be an no-op.
	// Included in case that implementation changes.
	t.Run("simple triangle", func(t *testing.T) {
		poly := &Polygon{[]*Point{{0, 0}, {1, 1}, {0, 2}}}

		triangles := TriangulateMonotone(poly)
		AssertValidTriangulation(t, poly, triangles)
	})

	t.Run("wacky triangle", func(t *testing.T) {
		poly := &Polygon{[]*Point{{-10, 0}, {43, 2}, {0, 2}}}

		triangles := TriangulateMonotone(poly)
		AssertValidTriangulation(t, poly, triangles)
	})

	t.Run("triangle with horizontal", func(t *testing.T) {
		// A horizontal segment is always acceptable in a triangle. It will only
		// affect which chain the segment is considered to be part of
		poly := &Polygon{[]*Point{{0, 0}, {1, 0}, {0, 1}}}
		triangles := TriangulateMonotone(poly)
		AssertValidTriangulation(t, poly, triangles)
	})

	// Quadrilaterals.
	t.Run("square", func(t *testing.T) {
		// A square has horizontal segments, but it is still strictly y-monotone
		// because of the lexiographic ordering.
		poly := &Polygon{[]*Point{{0, 0}, {1, 0}, {1, 1}, {0, 1}}}
		triangles := TriangulateMonotone(poly)
		AssertValidTriangulation(t, poly, triangles)
	})

	t.Run("diamond", func(t *testing.T) {
		poly := &Polygon{[]*Point{{0, 0}, {1, 1}, {0, 2}, {-1, 1}}}
		triangles := TriangulateMonotone(poly)
		AssertValidTriangulation(t, poly, triangles)
	})

	t.Run("quad chevron", func(t *testing.T) {
		// Our first non-convex quadrilateral, shaped like this:
		/*
			 C
			 \ \
			  \  \
			  D   B
			 /  /
			/ /
			A
		*/
		poly := &Polygon{[]*Point{
			{0, 0},
			{10, 10},
			{0, 20},
			{5, 10},
		}}
		triangles := TriangulateMonotone(poly)
		AssertValidTriangulation(t, poly, triangles)
	})
	// Fixtures
	fixtureNames := []string{
		"monotone_asteroid",
		"monotone_c",
		"monotone_diamond",
	}
	for _, fixtureName := range fixtureNames {
		t.Run(fixtureName+" (original)", func(t *testing.T) {
			poly := LoadFixture(fixtureName)
			triangles := TriangulateMonotone(poly)
			AssertValidTriangulation(t, poly, triangles)
		})
		t.Run(fixtureName+" (x reflected)", func(t *testing.T) {
			poly := LoadFixture(fixtureName)
			*poly = poly.Reverse()
			for _, p := range poly.Points {
				p.X = -p.X
			}
			triangles := TriangulateMonotone(poly)
			AssertValidTriangulation(t, poly, triangles)
		})

		t.Run(fixtureName+" (y reflected)", func(t *testing.T) {
			poly := LoadFixture(fixtureName)
			*poly = poly.Reverse()
			for _, p := range poly.Points {
				p.Y = -p.Y
			}
			triangles := TriangulateMonotone(poly)
			AssertValidTriangulation(t, poly, triangles)
		})

		t.Run(fixtureName+" (xy reflected)", func(t *testing.T) {
			poly := LoadFixture(fixtureName)
			for _, p := range poly.Points {
				p.X = -p.X
				p.Y = -p.Y
			}
			triangles := TriangulateMonotone(poly)
			AssertValidTriangulation(t, poly, triangles)
		})
	}
}
