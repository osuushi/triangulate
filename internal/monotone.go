package internal

// Facilities for converting a Y-monotone polygon into triangles. A Y monotone
// polygon is a simple polygon such that any horizontal line intersects at most
// two edges.
//
// The lexicographic Point.Below() method is used to simulate a slightly rotated
// coordinate system that eliminates horizontal segments but note that this
// affects where horizontal segments are allowed while maintaining strict
// monotonicity. Specifically, on the left chain, a horizontal edge must sit
// _above_ the inside of the polygon, while on the right chain, it must sit
// _below_. Since this convention is consistent with the assumptions used in
// trapezoidation, this is not a problem.
//
// Note that the polygon must be counterclockwise.

func TriangulateMonotone(polygon *Polygon) []*Triangle {
	if len(polygon.Points) < 3 {
		fatalf("cannot triangulate degenerate polygon with point count: %d", len(polygon.Points))
	}
	if len(polygon.Points) == 3 {
		return []*Triangle{{polygon.Points[0], polygon.Points[1], polygon.Points[2]}}
	}

	triangles := make([]*Triangle, 0, len(polygon.Points)-2)

	// Sort points so top point is at the top of the array.
	sortedPoints := make([]*Point, 0, len(polygon.Points))

	// Map to find index by point
	pointMap := make(map[*Point]int)

	// Find the top point, and build the index lookup
	var topPointIndex int
	for i, point := range polygon.Points {
		pointMap[point] = i
		if point.Above(polygon.Points[topPointIndex]) {
			topPointIndex = i
		}
	}

	sortedPoints = append(sortedPoints, polygon.Points[topPointIndex])

	// Structure for determining which chain a point is on the left or right chain
	leftChain := map[*Point]struct{}{}
	var isLeft = func(p *Point) bool {
		_, ok := leftChain[p]
		return ok
	}

	// Merge sort points starting from top, noting which are on the left chain, and track the bottom point separately
	leftOffset := 1
	rightOffset := 1
	// Check which point is next for the buffer
	var bottomPoint *Point
	for {
		leftPoint := polygon.Points[CircularIndex(topPointIndex+leftOffset, len(polygon.Points))]
		rightPoint := polygon.Points[CircularIndex(topPointIndex-rightOffset, len(polygon.Points))]

		// If we've met up, we're done. We don't add the bottom point to the list,
		// as it's handled at the very end.
		if leftPoint == rightPoint {
			bottomPoint = leftPoint
			break
		}

		if leftPoint.Above(rightPoint) {
			leftChain[leftPoint] = struct{}{}
			sortedPoints = append(sortedPoints, leftPoint)
			leftOffset++
		} else {
			sortedPoints = append(sortedPoints, rightPoint)
			rightOffset++
		}
	}
	// Create the stack and populate it with the first two points
	stack := make(PointStack, 0)
	stack.Push(sortedPoints[0])
	stack.Push(sortedPoints[1])
	// Iterate over the remainder of the sorted points
	for i, p := range sortedPoints[2:] {
		// Adjust index to account for the offset
		i := i + 2

		left := isLeft(p)
		if left != isLeft(stack.Peek()) { // If switched to opposite side chain
			// If we've jumped to the other chain, monotonicity guarantees that all
			// stack points are visible from the current point. We can there for empty the entire stack, making new triangles
			for !stack.Empty() {
				a := stack.Pop()
				if !stack.Empty() {
					b := stack.Peek()
					if left {
						/*
						              b
						             /|
						 diagonal-> / |
						           p--a
						*/
						triangles = appendTriangle(triangles, &Triangle{p, a, b})
					} else {
						/*
							b
							|\ <- Diagonal
							| \
							a--p
						*/
						triangles = appendTriangle(triangles, &Triangle{a, p, b})
					}
				}
			}
			// Put the last two points on the stack
			stack.Push(sortedPoints[i-1])
			stack.Push(sortedPoints[i])
		} else { // Same side chain
			// Always pop the last point off. If we don't create any triangles this
			// time, we'll put it back
			v := stack.Pop()

			// Get the initial unitVectorY value
			for !stack.Empty() {
				topOfStack := stack.Peek()
				// The easiest way to see if the point "sees" the top of the stack is to
				// try creating the triangle, and see if it's CCW
				var potentialTriangle *Triangle
				if left {
					/*
						q
						|\
						v \
						  \\ <- diagonal
						    \
						     p
					*/
					potentialTriangle = &Triangle{p, topOfStack, v}
				} else {
					/*
						               q
						              /|
						             / v
						            / /
						diagonal-> //
						          /
						         p
					*/
					potentialTriangle = &Triangle{p, v, topOfStack}
				}
				if IsCCW(potentialTriangle) {
					v = stack.Pop()
					triangles = append(triangles, potentialTriangle)
				} else {
					// Stop looping if we can't see the next point
					break
				}
			}

			// Put the last v back on the stack, and then the current point
			stack.Push(v)
			stack.Push(p)
		}
	}

	// Finally, add triangles for all remaining points on the stack. Note that we
	// always have two points.
	l := stack.Pop()
	for !stack.Empty() {
		p := stack.Pop()
		// Note that if we were just creating diagonals, as you'll sometimes see
		// with this algorithm, we would stop at the last point. However, we need
		// to generate the final triangle. Observe, for example, that in a case
		// where only two points remained on the stack, stopping before the last
		// point would completely remove the bottom point from the final triangle
		// list.

		// Check if last point is on the left chain
		if isLeft(l) {
			/*
					 p
				 / |
				l  | <- diagonal
				 \ |
				   b
			*/
			triangles = appendTriangle(triangles, &Triangle{bottomPoint, p, l})
		} else {
			/*
				            p
				            | \
				diagonal -> |  l
				            | /
				            b
			*/
			triangles = appendTriangle(triangles, &Triangle{bottomPoint, l, p})
		}
		l = p
	}
	return triangles
}

// This is pulled out so that it's easy to add instrumentation.
func appendTriangle(triangles []*Triangle, tri *Triangle) []*Triangle {
	if IsCW(tri) {
		fatalf("triangle is clockwise: %v", tri)
	}

	return append(triangles, tri)
}
