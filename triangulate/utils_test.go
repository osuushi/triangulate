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

// Helpers

func rotatePoint(point *Point, angle float64) {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	x := point.X
	y := point.Y
	point.X = x*cos - y*sin
	point.Y = x*sin + y*cos
}
