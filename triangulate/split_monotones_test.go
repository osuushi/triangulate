package triangulate

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToMonotones(t *testing.T) {
	// Splitting something already monotone
	poly := LoadFixture("spiral")
	list := ConvertToMonotones(PolygonList{*poly})
	assert.NotNil(t, list)
	assert.GreaterOrEqual(t, len(list), 1, "expected at least one polygon")

	pointSet := make(map[*Point]struct{})
	for _, poly := range list {
		for _, p := range poly.Points {
			pointSet[p] = struct{}{}
		}
	}

	fmt.Println("Ground truth:")
	PolygonList{*poly}.dbgDraw(50)

	// for i, poly := range list {
	// 	if IsCW(&poly) {
	// 		list[i] = poly.Reverse()
	// 	}
	// }
	assert.Equal(t, len(poly.Points), len(pointSet), "expected same number of points in split monotones")

	fmt.Println("Actual:")
	list.dbgDraw(50)
	// validatePolygonsBySampling(t, list, PolygonList{*poly})
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
