package advanced

type TrapezoidSet map[*Trapezoid]struct{}

// Use a query graph to split a set of polygons into monotone polygons.
func ConvertToMonotones(list PolygonList) PolygonList {
	// TODO: QueryGraph should natively support adding all of the polygons at once
	graph := &QueryGraph{}
	for _, polygon := range list {
		graph.AddPolygon(polygon)
	}

	trapezoids := make(TrapezoidSet)
	for trapezoid := range graph.IterateTrapezoids() {
		// Skip trapezoids that aren't inside
		if !trapezoid.IsInside() {
			continue
		}
		trapezoids[trapezoid] = struct{}{}
	}

	// This step will turn all trapezoids that should have diagonals (trapezoids
	// who have two non-adjacent points on their boundary) into two trapezoids.
	// This has destroyed the query graph, but not the neighbor graph; the
	// neighbor graph is valid over _inside_ trapezoids, which is all we care
	// about. We also can no longer trust the output of IsInside(), because some
	// trapezoids have been split with segments that do not obey its winding rule.
	// We will use the trapezoid set instead to determine if a trapezoid is
	// inside.
	splitTrapezoidsOnDiagonals(trapezoids)

	var result PolygonList
	for trapezoid := range trapezoids {
		// Scan to the top trapezoid in the monotone. It will always be degenerate
		// on top, and therefore have zero neighbors
		for {
			aboveNeighbor := trapezoid.TrapezoidsAbove.AnyNeighbor()
			if aboveNeighbor == nil {
				break
			}
			if _, ok := trapezoids[aboveNeighbor]; !ok {
				break
			}
			trapezoid = aboveNeighbor
		}

		// The top point is on both chains. We arbitrarily put it on the left
		leftChain := []*Point{trapezoid.Top}
		var rightChain []*Point

		// Traverse the trapezoid chain, collecting the points on the trapezoid's boundary
		for {
			bottom := trapezoid.Bottom
			leftBottom := trapezoid.Left.Bottom()
			rightBottom := trapezoid.Right.Bottom()

			if bottom == leftBottom && bottom == rightBottom {
				// We converged, so just put it on the left chain and break
				leftChain = append(leftChain, bottom)
				delete(trapezoids, trapezoid)
				break
			}

			// Figure out which chain we're on
			if bottom == leftBottom {
				leftChain = append(leftChain, bottom)
			} else if bottom == rightBottom {
				rightChain = append(rightChain, bottom)
			} else {
				fatalf("bottom point was not on either chain")
			}

			delete(trapezoids, trapezoid) // Skip iterating this later
			belowNeighbor := trapezoid.TrapezoidsBelow.AnyNeighbor()
			if belowNeighbor == nil {
				break
			}
			if _, ok := trapezoids[belowNeighbor]; !ok {
				break
			}
			trapezoid = belowNeighbor
		}

		// Now concatenate all the points from the right chain onto the left in reverse order
		points := leftChain
		for i := len(rightChain) - 1; i >= 0; i-- {
			points = append(points, rightChain[i])
		}
		if len(points) < 3 {
			fatalf("polygon is degenerate: %#v", points)
		}

		// Add the polygon to the result
		result = append(result, Polygon{points})
	}
	return result
}

// Split all trapezoids with diagonals into two trapezoids, updating the
// neighbor relationships. Note that this invalidates the query graph, and it
// breaks the validity of IsInside(), so we cannot use either of those after
// this has been used.
func splitTrapezoidsOnDiagonals(trapezoids TrapezoidSet) {
	for trapezoid := range trapezoids {
		top := trapezoid.Top
		bottom := trapezoid.Bottom
		leftTop := trapezoid.Left.Top()
		leftBottom := trapezoid.Left.Bottom()
		rightTop := trapezoid.Right.Top()
		rightBottom := trapezoid.Right.Bottom()

		// Skip if the top and bottom are one of the trapezoid's sides. There's no diagonal in that case
		if top == leftTop && bottom == leftBottom {
			continue
		} else if top == rightTop && bottom == rightBottom {
			continue
		}

		// Split the trapezoid into two trapezoids
		segment := &Segment{top, bottom}
		leftTrapezoid, rightTrapezoid := trapezoid.SplitBySegment(segment)

		// Remove the old trapezoid
		delete(trapezoids, trapezoid)

		// Add the trapezoids to the map
		trapezoids[leftTrapezoid] = struct{}{}
		trapezoids[rightTrapezoid] = struct{}{}

	}
}

func dbgDrawTrapezoids(trapezoids TrapezoidSet, scale float64) {
	var list PolygonList
	// Convert the trapezoids into polygons
	for trapezoid := range trapezoids {
		var points []*Point
		topY := trapezoid.Top.Y
		bottomY := trapezoid.Bottom.Y
		if trapezoid.Left.IsHorizontal() || trapezoid.Right.IsHorizontal() {
			// The trapezoid is degenerate, so just draw a line
			points = []*Point{trapezoid.Top, trapezoid.Bottom}
		} else {
			leftTopX := trapezoid.Left.SolveForX(topY)
			leftBottomX := trapezoid.Left.SolveForX(bottomY)
			rightTopX := trapezoid.Right.SolveForX(topY)
			rightBottomX := trapezoid.Right.SolveForX(bottomY)

			points = append(points, &Point{leftTopX, topY})
			points = append(points, &Point{leftBottomX, bottomY})
			points = append(points, &Point{rightBottomX, bottomY})
			points = append(points, &Point{rightTopX, topY})
		}
		list = append(list, Polygon{points})
	}
	list.dbgDraw(scale)
}
