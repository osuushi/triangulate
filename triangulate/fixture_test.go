package triangulate

import (
	"embed"
	"log"
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
