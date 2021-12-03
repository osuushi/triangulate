package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/osuushi/triangulate/triangulate"
)

// Demo of triangulation by generating an SVG file triangulting a set of
// polygons. Input on stdin should be newline separated points in the form "x
// y", with each polygon separated by an extra newline.
//
// Polygons should be simple and wind counterclockwise. A clockwise polygon is a
// hole. A hole should be contained by only one outer polygon, and should not
// intersect its edges. None of these requirements are validated.
func main() {
	polygons := readPolygons(os.Stdin)
	fmt.Printf("Read %d polygons\n", len(polygons))
}

func readPolygons(in *os.File) []Polygon {
	polygons := []Polygon{}
	// Scan lines
	scanner := bufio.NewScanner(in)
	points := []*Point{}
	for scanner.Scan() {
		// Read the next line
		line := scanner.Text()

		// If it's empty, and we collected any points, this is the end of the polygon
		if line == "" {
			if len(points) > 0 {
				polygons = append(polygons, Polygon{Points: points})
				points = []*Point{}
			}
			continue
		}

		// Parse the point out of the line
		point := parsePoint(line)
		points = append(points, &point)
	}

	// Handle trailing polygon if any
	if len(points) > 0 {
		polygons = append(polygons, Polygon{Points: points})
	}
	return polygons
}

func parsePoint(line string) Point {
	parts := strings.Fields(line)
	x, _ := strconv.ParseFloat(parts[0], 64)
	y, _ := strconv.ParseFloat(parts[1], 64)
	return Point{X: x, Y: y}
}
