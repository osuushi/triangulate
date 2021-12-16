package internal

import (
	"testing"
)

func TestTriangulate_Spiral(t *testing.T) {
	shape := LoadFixture("spiral")
	list := PolygonList{*shape}
	result := list.Triangulate()
	validatePolygonsBySampling(t, result.ToPolygonList(), list)
}

func TestTriangulate_Star(t *testing.T) {
	shape := SimpleStar()
	result := shape.Triangulate()
	validatePolygonsBySampling(t, result.ToPolygonList(), shape)
}

func TestTriangulate_SquareWithHole(t *testing.T) {
	shape := SquareWithHole()
	result := shape.Triangulate()
	validatePolygonsBySampling(t, result.ToPolygonList(), shape)
}

func TestTriangulate_StarOutline(t *testing.T) {
	shape := StarOutline()
	result := shape.Triangulate()
	validatePolygonsBySampling(t, result.ToPolygonList(), shape)
}

func TestTriangulate_StarStripes(t *testing.T) {
	shape := StarStripes()
	result := shape.Triangulate()
	validatePolygonsBySampling(t, result.ToPolygonList(), shape)
}

func TestTriangulate_MultiLayeredHoles(t *testing.T) {
	shape := MultiLayeredHoles()
	result := shape.Triangulate()
	result.dbgDraw(50)
	validatePolygonsBySampling(t, result.ToPolygonList(), shape)
}
