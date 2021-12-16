package triangulate

import "github.com/osuushi/triangulate/internal"

type Point internal.Point
type Polygon internal.Polygon
type PolygonList internal.PolygonList
type Triangle internal.Triangle
type TriangleList internal.TriangleList

func (list PolygonList) Triangulate() TriangleList {
	return TriangleList(internal.PolygonList(list).Triangulate())
}
