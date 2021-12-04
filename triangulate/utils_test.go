package triangulate

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPointStack(t *testing.T) {
	var ps PointStack
	assert.True(t, ps.Empty())
	ps.Push(&Point{1, 2})
	assert.False(t, ps.Empty())
	assert.Equal(t, &Point{1, 2}, ps.Peek())
	assert.False(t, ps.Empty())
	assert.Equal(t, &Point{1, 2}, ps.Pop())
	assert.True(t, ps.Empty())
	ps.Push(&Point{1, 2})
	ps.Push(&Point{3, 4})
	assert.False(t, ps.Empty())
	assert.Equal(t, &Point{3, 4}, ps.Peek())
	assert.Equal(t, &Point{3, 4}, ps.Pop())
	assert.False(t, ps.Empty())
	assert.Equal(t, &Point{1, 2}, ps.Peek())
	assert.Equal(t, &Point{1, 2}, ps.Pop())
	assert.True(t, ps.Empty())
}

func TestCircularIndex(t *testing.T) {
	n := 3
	expectedIndexes := []int{0, 1, 2, 0, 1, 2, 0, 1, 2}
	for i := -3; i < 6; i++ {
		actualIndex := CircularIndex(i, n)
		expectedIndex := expectedIndexes[0]
		expectedIndexes = expectedIndexes[1:]
		assert.Equal(t, expectedIndex, actualIndex)
	}
}

func TestTriangleSignedArea(t *testing.T) {
	for cwI := 0; cwI < 2; cwI++ {
		cwI := cwI // import into inner scope
		t.Run(fmt.Sprintf("With %s triangles", []string{"CCW", "CW"}[cwI]), func(t *testing.T) {
			tri := new(Triangle)
			tri.A = &Point{0, -1}
			tri.B = &Point{1, 0}
			tri.C = &Point{0, 1}
			// Clockwise triangles will have negative area, so sign is -1 for CW = 1
			sign := 1 - 2*float64(cwI)
			assertArea := func(expected float64) {
				assert.InDelta(t, sign*expected, tri.SignedArea(), Epsilon)
			}
			if cwI == 1 {
				tri.A, tri.B = tri.B, tri.A
			}
			assertArea(1)
			// Stretch the triangle out
			tri.A.Y *= 2
			tri.B.Y *= 2
			tri.C.Y *= 2
			assertArea(2)

			// Rotate the triangle repeatedly by a weird angle
			angle := math.Pi / 7
			for i := 0; i < 14; i++ {
				// Multiply each point by the rotation matrix
				rotatePoint(tri.A, angle)
				rotatePoint(tri.B, angle)
				rotatePoint(tri.C, angle)
				assertArea(2)
			}

			// Translate the triangle and do the whole rotation thing again
			tri.A.X += 5
			tri.A.Y += 3
			tri.B.X += 5
			tri.B.Y += 3
			tri.C.X += 5
			tri.C.Y += 3

			for i := 0; i < 14; i++ {
				// Multiply each point by the rotation matrix
				rotatePoint(tri.A, angle)
				rotatePoint(tri.B, angle)
				rotatePoint(tri.C, angle)
				assertArea(2)
			}
		})
	}
}

func TestPolygonSignedArea(t *testing.T) {
	for cwI := 0; cwI < 2; cwI++ {
		cwI := cwI // import into inner scope
		t.Run(fmt.Sprintf("With %s polygons", []string{"CCW", "CW"}[cwI]), func(t *testing.T) {
			// Make a skewed polygon - an hourglass - which it is hopefully easy to see has area 64
			poly := Polygon{
				Points: []*Point{
					{2, 0},
					{6, 4},
					{-6, 4},
					{-2, 0},
					{-6, -4},
					{6, -4},
				},
			}
			// Skew the polygon by moving the right side points down. The principle here
			// is to avoid bugs being hidden by reflective symmetry, but this skew
			// (hopefully obviously), doesn't change the area. The result is a sort of stylized "Z"
			for _, point := range poly.Points {
				if point.X > 0 {
					point.Y -= 5
				}
			}

			// Clockwise triangles will have negative area, so sign is -1 for CW = 1
			sign := 1 - 2*float64(cwI)
			assertArea := func(expected float64) {
				assert.InDelta(t, sign*expected, poly.SignedArea(), Epsilon)
			}
			if cwI == 1 {
				poly = poly.Reverse()
			}
			assertArea(64)
			// Stretch the triangle out
			for _, p := range poly.Points {
				p.Y *= 2
			}
			assertArea(128)

			// Rotate the polygon repeatedly by a weird angle
			angle := math.Pi / 7
			for i := 0; i < 14; i++ {
				// Multiply each point by the rotation matrix
				for _, p := range poly.Points {
					rotatePoint(p, angle)
				}
				assertArea(128)
			}

			// Translate the polygon and do the whole rotation thing again
			for _, p := range poly.Points {
				p.X += 5
				p.Y += 3
			}

			for i := 0; i < 14; i++ {
				// Multiply each point by the rotation matrix
				for _, p := range poly.Points {
					rotatePoint(p, angle)
				}
				assertArea(128)
			}
		})
	}
}

// Test the lexicographically adjusted "below" method
func TestBelow(t *testing.T) {
	p := &Point{1, 1}
	// Below by normal standards
	assert.True(t, p.Below(&Point{1, 2}))
	// Above by normal standards
	assert.False(t, p.Below(&Point{1, 0}))

	// Below by lexicographic correction (other point is to the right, so it is
	// "above" p because of the tie-break)
	assert.True(t, p.Below(&Point{2, 1}))
	// Above by lexicographic correction (other point is to the left, so it is
	// "below" p because of the tie-break)
	assert.False(t, p.Below(&Point{0, 1}))
}

// Helpers

func rotatePoint(point *Point, angle float64) {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	x := point.X
	y := point.Y
	point.X = x*cos - y*sin
	point.Y = x*sin + y*cos
}
