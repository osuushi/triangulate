// An asymptotically fast triangulation package for Go.
//
// This package allows you to convert a set of simple polygons, which may be
// non-convex, may be disjoint, and may contain holes, and convert them into a
// set of triangles containing only the original points.
package triangulate

import "github.com/osuushi/triangulate/internal"

type Point struct {
	X, Y float64
}

type Triangle struct {
	A, B, C Point
}

// Take a set of point lists and convert them into triangles.
//
// The polygons must be simple and non-intersecting. "Solid" polygons must give
// their points in counterclockwise order, while "holes" must be in clockwise
// order.
//
// The order of the polygons is irrelevant. See the readme for more details.
func Triangulate(polygons ...[]Point) (result []Triangle, err error) {
	defer func() {
		recoveredErr := internal.HandleTriangulatePanicRecover(recover())
		if recoveredErr != nil {
			result = nil
			err = recoveredErr
		}
	}()
	// To avoid having to export everything into the top level package, while
	// making the internal package still largely exported for advanced usage, we
	// copy all the points into the internal types. This keeps the external API
	// nice and tidy, but it's unfortunate that it requires an O(n) copy both in
	// and out.
	//
	// Another alternative would be to put the types into their own package, but
	// that means users have to import two packages, which is annoying. Might
	// decide to take a different approach here in the future.
	var internalPolygons = make(internal.PolygonList, len(polygons))
	for i, poly := range polygons {
		var polygon = internal.Polygon{}
		for _, point := range poly {
			polygon.Points = append(polygon.Points, &internal.Point{X: point.X, Y: point.Y})
		}
		internalPolygons[i] = polygon
	}
	triangleList := internalPolygons.Triangulate()
	var triangles = make([]Triangle, len(triangleList))
	for i, triangle := range triangleList {
		triangles[i] = Triangle{
			A: Point{triangle.A.X, triangle.A.Y},
			B: Point{triangle.B.X, triangle.B.Y},
			C: Point{triangle.C.X, triangle.C.Y},
		}
	}
	return triangles, nil
}
