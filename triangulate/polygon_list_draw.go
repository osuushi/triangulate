package triangulate

import (
	"math"
	"os"

	"github.com/fogleman/gg"
	imgcat "github.com/martinlindhe/imgcat/lib"
)

// This is for debugging purposes only

func (pl PolygonList) dbgDraw(scale float64) {
	var minX, minY, maxX, maxY float64
	minX = math.Inf(1)
	minY = math.Inf(1)
	maxX = math.Inf(-1)
	maxY = math.Inf(-1)
	for _, poly := range pl {
		for _, p := range poly.Points {
			minX = math.Min(minX, p.X)
			minY = math.Min(minY, p.Y)
			maxX = math.Max(maxX, p.X)
			maxY = math.Max(maxY, p.Y)
		}
	}

	// Set up the context
	width := int(scale*(maxX-minX)) + dbgDrawPadding*2
	height := int(scale*(maxY-minY)) + dbgDrawPadding*2
	c := gg.NewContext(width, height)
	c.SetRGB(0, 0, 0)
	c.DrawRectangle(0, 0, float64(width), float64(height))
	c.Fill()
	c.SetFillRuleEvenOdd()

	// Flip the context so the origin is at the bottom left
	c.Translate(0, float64(height))
	c.Scale(1, -1)

	// Translate for padding
	c.Translate(dbgDrawPadding, dbgDrawPadding)
	// Scale
	c.Scale(scale, scale)
	// Translate to min
	c.Translate(-minX, -minY)

	c.SetLineWidth(2)
	for _, poly := range pl {
		c.MoveTo(poly.Points[0].X, poly.Points[0].Y)
		for _, p := range poly.Points[1:] {
			c.LineTo(p.X, p.Y)
		}
		c.ClosePath()
	}
	c.SetRGB(0, 0.5, 0)
	c.FillPreserve()
	c.SetRGB(0, 1, 1)
	c.Stroke()

	c.SavePNG("/tmp/polygon_list.png")
	imgcat.CatFile("/tmp/polygon_list.png", os.Stdout)
}
