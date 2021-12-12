package triangulate

func (list PolygonList) Triangulate() TriangleList {
	monotones := ConvertToMonotones(list)
	var result TriangleList
	for _, monotone := range monotones {
		triangles := TriangulateMonotone(&monotone)
		result = append(result, triangles...)
	}
	return result
}
