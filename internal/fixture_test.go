package internal

import (
	"embed"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/JoshVarga/svgparser"
)

// This file parses the svg pixtures and outputs polygons. This is not a full
// (or even correct) svg parser. Handler. It parses the SVG and then finds
// whatever the first polygon is, then converts that into a CCW *Polygon. If
// anything goes wrong, it panics.
//
// Fixtures are available by name in this fixtures/ directory, sans extension.

//go:embed fixtures
var fixtures embed.FS

func LoadFixture(name string) *Polygon {
	fixture, err := fixtures.Open("fixtures/" + name + ".svg")
	if err != nil {
		log.Fatalf("Could not load fixture %q: %v", name, err)
	}

	defer fixture.Close()
	rootEl, err := svgparser.Parse(fixture, true)
	if err != nil {
		log.Fatalf("Failed to parse fixture %q: %v", name, err)
	}

	// Find the first polygon
	polygons := rootEl.FindAll("polygon")
	if len(polygons) == 0 {
		log.Fatalf("No polygons found in fixture %q", name)
	}
	if len(polygons) > 1 {
		log.Fatalf("More than one polygon found in fixture %q", name)
	}
	polygonEl := polygons[0]

	pointString := polygonEl.Attributes["points"]
	pointStrings := strings.Split(pointString, " ")
	points := make([]*Point, 0, len(pointStrings))
	for _, pointString := range pointStrings {
		if pointString == "" {
			continue
		}

		pointStrings := strings.Split(pointString, ",")
		if len(pointStrings) != 2 {
			log.Fatalf("Invalid point string %q", pointString)
		}
		x, err := strconv.ParseFloat(pointStrings[0], 64)
		if err != nil {
			log.Fatalf("Invalid x value %q: %v", pointStrings[0], err)
		}
		y, err := strconv.ParseFloat(pointStrings[1], 64)
		if err != nil {
			log.Fatalf("Invalid y value %q: %v", pointStrings[1], err)
		}
		points = append(points, &Point{x, y})
	}
	result := Polygon{Points: points}

	// Ensure that the polygon is CCW
	if IsCW(&result) {
		result = result.Reverse()
	}
	return &result
}

// Some ad hoc code specified fixtures
func SimpleStar() PolygonList {
	var points []*Point
	const outerRadius = 5
	const innerRadius = 2
	for i := 0; i < 10; i++ {
		var radius float64
		if i%2 == 0 {
			radius = outerRadius
		} else {
			radius = innerRadius
		}
		angle := 2 * math.Pi * float64(i) / 10
		points = append(points, &Point{X: radius * math.Cos(angle), Y: radius * math.Sin(angle)})
	}
	poly := Polygon{points}
	return PolygonList{poly}
}

func SquareWithHole() PolygonList {
	outerPoints := []*Point{
		{X: -5, Y: -5},
		{X: 5, Y: -5},
		{X: 5, Y: 5},
		{X: -5, Y: 5},
	}

	holePoints := []*Point{
		{X: -2, Y: -2},
		{X: -2, Y: 2},
		{X: 2, Y: 2},
		{X: 2, Y: -2},
	}

	return PolygonList{
		Polygon{outerPoints},
		Polygon{holePoints},
	}
}

func StarOutline() PolygonList {
	filledPoints := []*Point{}
	holePoints := []*Point{}
	const filledOuterRadius = 10
	const filledInnerRadius = 5
	const holeOuterRadius = filledOuterRadius - 2
	const holeInnerRadius = filledInnerRadius - 2
	for i := 0; i < 10; i++ {
		var (
			filledRadius float64
			holeRadius   float64
		)
		if i%2 == 0 {
			filledRadius = filledOuterRadius
			holeRadius = holeOuterRadius
		} else {
			filledRadius = filledInnerRadius
			holeRadius = holeInnerRadius
		}
		angle := 2 * math.Pi * float64(i) / 10
		filledPoints = append(filledPoints, &Point{X: filledRadius * math.Cos(angle), Y: filledRadius * math.Sin(angle)})
		holePoints = append(holePoints, &Point{X: holeRadius * math.Cos(angle), Y: holeRadius * math.Sin(angle)})
	}

	return PolygonList{
		Polygon{filledPoints},
		Polygon{holePoints}.Reverse(),
	}
}

func StarStripes() PolygonList {
	// Multiple inset stars with alternating winding
	var list PolygonList
	const outerRadius = 10
	const n = 20
	var scale float64 = 1
	const indentScale = 0.7
	const gapScale = 0.9

	for i := 0; i < n; i++ {
		var points []*Point
		for j := 0; j < 10; j++ {
			angle := 2 * math.Pi * float64(j) / 10
			r := outerRadius * scale
			if j%2 == 1 {
				r *= indentScale
			}
			points = append(points, &Point{X: r * math.Cos(angle), Y: r * math.Sin(angle)})
		}
		scale *= gapScale
		poly := Polygon{points}
		if i%2 == 1 {
			poly = poly.Reverse()
		}
		list = append(list, poly)
	}
	return list
}

func MultiLayeredHoles() PolygonList {
	// In this test, we want multiple holes which contain filled shapes inside.
	makeStar := func(x, y, outerRadius, innerRadius float64) Polygon {
		points := []*Point{}
		for i := 0; i < 10; i++ {
			angle := 2 * math.Pi * float64(i) / 10
			r := outerRadius
			if i%2 == 1 {
				r = innerRadius
			}
			points = append(points, &Point{X: x + r*math.Cos(angle), Y: y + r*math.Sin(angle)})
		}
		return Polygon{points}
	}
	list := PolygonList{
		// Outer star
		makeStar(0, 0, 10, 7),
		// Top hole
		makeStar(1.5, 5, 3, 2).Reverse(),
		// Top inner
		makeStar(1.5, 5, 2, 1),
		// Bottom hole
		makeStar(1.8, -5, 3, 2).Reverse(),
		// Bottom inner
		makeStar(1.8, -5, 2, 1),
		// Left hole
		makeStar(-3, 0, 4, 2).Reverse(),
		// Left inner
		makeStar(-3, 0, 3, 1),
	}
	return list
}
