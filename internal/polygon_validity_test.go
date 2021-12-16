package internal

// This contains no actual tests. It is just a helper for testing triangulation
// validity.

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to check that a triangulation is valid. The rules are:
// 1. The set of points in the triangles must equal the set of points in the polygon.
// 2. The set of line segments in the polygon is a subset of the set of line segments in the triangles.
// 3. Every triangle is counterclockwise
// 4. No triangle has zero area
// 5. The sum of the areas of all triangles is equal to the area of the polygon.

func AssertValidTriangulation(t *testing.T, polygon *Polygon, triangles []*Triangle) {
	if !IsCCW(polygon) {
		t.Fatal("Polygon is not counterclockwise")
	}

	polyPoints := make(PointSet)
	for _, p := range polygon.Points {
		polyPoints.Add(p)
	}
	trianglePoints := make(PointSet)
	for _, t := range triangles {
		trianglePoints.Add(t.A)
		trianglePoints.Add(t.B)
		trianglePoints.Add(t.C)
	}

	require.True(t, polyPoints.Equals(trianglePoints), "set of points in the triangles must equal the set of points in the polygon")

	var triangleArea float64
	triangleSegmentSet := make(normalizedSegmentSet)
	for _, tri := range triangles {
		// Check that the triangle is counterclockwise
		require.True(t, IsCCW(tri), "clockwise triangle: %s", tri)
		triangleArea += Area(tri)
		// Add all the segments to the set
		triangleSegmentSet.add(tri.A, tri.B)
		triangleSegmentSet.add(tri.B, tri.C)
		triangleSegmentSet.add(tri.C, tri.A)
	}

	// Check every segment in the polygon is in the set
	for i, p1 := range polygon.Points {
		p2 := polygon.Points[(i+1)%len(polygon.Points)]
		require.True(t, triangleSegmentSet.contains(p1, p2), "segment %v-%v of the is not in the set of segments in the triangles", p1, p2)
	}

	// Check that the sum of the areas of all triangles is equal to the area of the polygon
	require.InDelta(t, Area(polygon), triangleArea, Epsilon, "sum of the areas of all triangles is equal to the area of the polygon")
}

// Used in the helper above, this is a "normalized" line segment, where the
// "lower" point (accounting for lexicographic adjustment) is always second
type normalizedSegment struct {
	lower, upper *Point
}

func newNormalizedSegment(a, b *Point) normalizedSegment {
	if a.Below(b) {
		return normalizedSegment{a, b}
	}
	return normalizedSegment{b, a}
}

type normalizedSegmentSet map[normalizedSegment]struct{}

func (set normalizedSegmentSet) add(a, b *Point) {
	set[newNormalizedSegment(a, b)] = struct{}{}
}

func (set normalizedSegmentSet) contains(a, b *Point) bool {
	_, ok := set[newNormalizedSegment(a, b)]
	return ok
}

func validatePolygonsBySampling(t *testing.T, actualPolygons PolygonList, expectedPolygons PolygonList) {
	minX, minY, maxX, maxY, step := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1), 0.1
	for _, list := range []PolygonList{actualPolygons, expectedPolygons} {
		for _, poly := range list {
			for _, p := range poly.Points {
				minX = math.Min(minX, p.X)
				minY = math.Min(minY, p.Y)
				maxX = math.Max(maxX, p.X)
				maxY = math.Max(maxY, p.Y)
				maxX = math.Max(maxX, p.X)
			}
		}
	}

	// Pad the bounding box by 10%
	xPadding := (maxX - minX) * 0.1
	yPadding := (maxY - minY) * 0.1
	minX -= xPadding
	minY -= yPadding
	maxX += xPadding
	maxY += yPadding

	// Compute the step size
	step = math.Max(maxX-minX, maxY-minY) / 50

	for y := minY; y <= maxY; y += step {
		for x := minX; x <= maxX; x += step {
			p := &Point{X: x, Y: y}

			actual := actualPolygons.ContainsPointByEvenOdd(p)
			if expectedPolygons.ContainsPointByEvenOdd(p) {
				assert.True(t, actual, "point %v should be in the monotone set", p)
			} else {
				assert.False(t, actual, "point %v should not be in the monotone set", p)
			}
		}
	}
}
