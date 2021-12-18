package triangulate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Smoke test. The internals are already tested.
func TestTriangulate(t *testing.T) {
	points := []Point{
		{1, -1},
		{1, 1},
		{-1, 1},
		{-1, -1},
	}

	triangles, err := Triangulate(points)
	assert.NoError(t, err)
	assert.Len(t, triangles, 2)
}
