package advanced

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToMonotones_Spiral(t *testing.T) {
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

	assert.Equal(t, len(poly.Points), len(pointSet), "expected same number of points in split monotones")

	validatePolygonsBySampling(t, list, PolygonList{*poly})
}

func TestConvertToMonotones_Star(t *testing.T) {
	star := SimpleStar()
	list := ConvertToMonotones(star)
	validatePolygonsBySampling(t, list, star)
}

func TestConvertToMonotones_SquareWithHole(t *testing.T) {
	shape := SquareWithHole()
	list := ConvertToMonotones(shape)
	validatePolygonsBySampling(t, list, shape)
}

func TestConvertToMonotones_StarOutline(t *testing.T) {
	shape := StarOutline()
	list := ConvertToMonotones(shape)
	validatePolygonsBySampling(t, list, shape)
}

func TestConvertToMonotones_StarStripes(t *testing.T) {
	shape := StarStripes()
	list := ConvertToMonotones(shape)
	validatePolygonsBySampling(t, list, shape)
}

func TestConvertToMonotones_MultiLayeredHoles(t *testing.T) {
	shape := MultiLayeredHoles()
	list := ConvertToMonotones(shape)
	validatePolygonsBySampling(t, list, shape)
}
