package internal

import (
	"image"
	"math"
	"os"

	"github.com/fogleman/gg"
	imgcat "github.com/martinlindhe/imgcat/lib"
	"github.com/osuushi/triangulate/internal/dbg"
)

// Padding around the shape to make infinite trapezoids obvious
const dbgDrawPadding = 100

var inverseMatrixForContext map[*gg.Context]gg.Matrix

func init() {
	inverseMatrixForContext = make(map[*gg.Context]gg.Matrix)
}

// Helper to draw and print a query graph in the terminal (iTerm only) for debugging.
func (g *QueryGraph) dbgDraw(scale float64) {
	var minX, minY, maxX, maxY float64
	minX = math.Inf(1)
	minY = math.Inf(1)
	maxX = math.Inf(-1)
	maxY = math.Inf(-1)
	for t := range IterateTrapezoids(g.Root) {
		for _, side := range []*Segment{t.Left, t.Right} {
			if side == nil {
				continue
			}
			for _, point := range []*Point{side.Start, side.End} {
				minX = math.Min(minX, point.X)
				minY = math.Min(minY, point.Y)
				maxX = math.Max(maxX, point.X)
				maxY = math.Max(maxY, point.Y)
			}
		}
	}

	width := int(scale*(maxX-minX)) + dbgDrawPadding*2
	height := int(scale*(maxY-minY)) + dbgDrawPadding*2
	c := gg.NewContext(width, height)
	c.SetRGB(0, 0, 0)
	c.DrawRectangle(0, 0, float64(width), float64(height))
	c.Fill()
	// Flip the context so the origin is at the bottom left
	c.Translate(0, float64(height))
	c.Scale(1, -1)

	// Translate for padding
	c.Translate(dbgDrawPadding, dbgDrawPadding)
	// Scale
	c.Scale(scale, scale)
	// Translate to min
	c.Translate(-minX, -minY)

	// Reverse the above operations to get the inverse matrix. The gg library has
	// no matrix inverse, or even a way to get to the context matrix, so it comes
	// to this. Whatever, it's debugging code.
	inverseMatrix := gg.Identity().
		Translate(minX, minY).
		Scale(1/scale, 1/scale).
		Translate(-dbgDrawPadding, -dbgDrawPadding).
		Scale(1, -1).
		Translate(0, -float64(height))
	inverseMatrixForContext[c] = inverseMatrix

	c.SetLineWidth(3)
	g.draw(c)

	// Save to temp file
	c.SavePNG("/tmp/querygraph.png")
	// Print to terminal
	imgcat.CatFile("/tmp/querygraph.png", os.Stdout)
}

func (g *QueryGraph) draw(c *gg.Context) {
	// Find all the trapezoids and fill them, then stroke them
	for t := range IterateTrapezoids(g.Root) {
		t.draw(c, false)
	}
	for t := range IterateTrapezoids(g.Root) {
		t.draw(c, true)
	}
}

// Draw the trapezoid
func (t *Trapezoid) draw(c *gg.Context, stroke bool) {
	// Find the bounds of the canvas, for points at infinity
	bounds := getCanvasBounds(c)
	left, right := t.Left, t.Right
	top := t.Top
	bottom := t.Bottom
	if top == nil {
		top = &Point{X: 0, Y: float64(bounds.Max.Y)}
	}
	if bottom == nil {
		bottom = &Point{X: 0, Y: float64(bounds.Min.Y)}
	}

	for _, side := range []**Segment{&left, &right} {
		if *side == nil {
			x := float64(bounds.Min.X)
			if side == &right {
				x = float64(bounds.Max.X)
			}
			// Just make a line off the side of the image
			*side = &Segment{
				Start: &Point{X: x, Y: top.Y},
				End:   &Point{X: x, Y: bottom.Y},
			}
		} else if !(*side).IsHorizontal() { // leave horizontal segments alone
			// Solve for x
			var topX, bottomX float64
			topX = (*side).SolveForX(top.Y)
			bottomX = (*side).SolveForX(bottom.Y)
			*side = &Segment{
				Start: &Point{X: topX, Y: top.Y},
				End:   &Point{X: bottomX, Y: bottom.Y},
			}
		}
	}

	// Add the lines
	c.MoveTo(left.Start.X, left.Start.Y)
	c.LineTo(left.End.X, left.End.Y)
	c.LineTo(right.End.X, right.End.Y)
	c.LineTo(right.Start.X, right.Start.Y)
	c.ClosePath()
	if stroke {
		// Stroke
		c.SetRGB(0, 1, 0)
		c.Stroke()
	} else {
		if t.IsInside() {
			c.SetRGBA(0.3, 0.2, 1, 0.5)
			c.Fill()
		} else {
			c.SetRGBA(1, 1, 0, 0.5)
			c.Fill()
		}
		// Write the name of the trapezoid
		c.SetRGB(1, 1, 1)
		centerX := (left.Start.X + right.End.X + right.Start.X + left.End.X) / 4
		centerY := (left.Start.Y + right.End.Y + right.Start.Y + left.End.Y) / 4
		// We have to go back to identity to draw the text, so get the point in native coordinates
		centerX, centerY = c.TransformPoint(centerX, centerY)
		c.Push()
		c.Identity()
		// Undo scaling we're about to do
		centerX, centerY = gg.Identity().Scale(.5, .5).TransformPoint(centerX, centerY)
		c.Scale(2, 2)
		c.DrawStringAnchored(dbg.Name(t), centerX, centerY, 0.5, 0.5)
		c.Pop()
	}
}

func getCanvasBounds(c *gg.Context) image.Rectangle {
	matrix := inverseMatrixForContext[c]
	bounds := image.Rect(-10, -10, c.Width()+20, c.Height()+20)
	minX, minY := matrix.TransformPoint(float64(bounds.Min.X), float64(bounds.Min.Y))
	maxX, maxY := matrix.TransformPoint(float64(bounds.Max.X), float64(bounds.Max.Y))
	return image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Floor(maxX)), int(math.Floor(maxY)))
}
