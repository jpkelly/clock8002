package matrix

import ()

func (matrix *matrix) Scroll(bitmask [][]bool, color [3]byte, y, x, offset, length int) {
	matrix.DrawBitmask(matrix.getScrollBitmask(bitmask, y, x, offset, length), color, y, x)
}

func (matrix *matrix) ScrollPlasma(bitmask [][]bool, y, x, offset, length int) {
	matrix.DrawBitmapMask(matrix.PlasmaBitmap(offset),
		matrix.getScrollBitmask(bitmask, y, x, offset, length), y, x)
}

func (matrix *matrix) getScrollBitmask(bitmask [][]bool, y, x, offset, length int) [][]bool {
	s := len(bitmask[0])
	if s < length {
		return bitmask
	}
	// Seamlessly loop the bitmap around
	offset = offset % s
	size := s - offset
	if size >= length {
		return matrix.OffsetBitmask(bitmask, offset, length)
	} else {
		ret := make([][]bool, len(bitmask))
		for r, row := range matrix.OffsetBitmask(bitmask, offset, length) {
			ret[r] = row
		}
		for r, row := range matrix.OffsetBitmask(bitmask, 0, length-size) {
			ret[r] = append(ret[r], row...)
		}
		return ret
	}
}
