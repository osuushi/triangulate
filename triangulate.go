// An asymptotically fast triangulation package for Go.
//
// This package allows you to convert a set of simple polygons, which may be
// non-convex, may be disjoint, and may contain holes, and convert them into a
// set of triangles containing only the original points.
package triangulate

import "github.com/osuushi/triangulate/advanced"

type Point = advanced.Point
type Triangle = advanced.Triangle
type Polygon = advanced.Polygon

// Take a set of point lists and convert them into triangles.
//
// The polygons must be simple and non-intersecting. "Solid" polygons must give
// their points in counterclockwise order, while "holes" must be in clockwise
// order.
//
// The order of the polygons is irrelevant. See the readme for more details.
func Triangulate(polygonPoints ...[]*Point) (result []*Triangle, err error) {
	defer func() {
		recoveredErr := advanced.HandleTriangulatePanicRecover(recover())
		if recoveredErr != nil {
			result = nil
			err = recoveredErr
		}
	}()
	polygons := make(advanced.PolygonList, len(polygonPoints))
	for i, points := range polygonPoints {
		polygons[i] = advanced.Polygon{Points: points}
	}
	return []*Triangle(polygons.Triangulate()), nil
}
