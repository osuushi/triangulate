package internal

func (list TriangleList) dbgDraw(scale float64) {
	// Just turn the triangle list into a polygon list and use its draw method
	list.ToPolygonList().dbgDraw(scale)
}
