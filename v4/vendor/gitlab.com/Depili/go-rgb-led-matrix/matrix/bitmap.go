package matrix

import ()

func (matrix *matrix) DrawBitmask(bitmask [][]bool, color [3]byte, r int, c int) {
	for y, row := range bitmask {
		for x, b := range row {
			if b {
				matrix.SetPixel(r+y, c+x, color)
			}
		}
	}
}

func (matrix *matrix) DrawBitmapMask(bitmap [][][3]byte, bitmask [][]bool, y int, x int) {
	for r, row := range bitmask {
		for c, b := range row {
			if b {
				matrix.SetPixel(r+y, c+x, bitmap[r+y][c+x])
			}
		}
	}
}

func (matrix *matrix) DrawBitmap(bitmap [][][3]byte, y, x int) {
	for r, row := range bitmap {
		for c, color := range row {
			matrix.SetPixel(r+y, c+x, color)
		}
	}
}

func (matrix *matrix) ColorBitmap(bitmask [][]bool, color [3]byte) [][][3]byte {
	bitmap := make([][][3]byte, len(bitmask))
	for r, row := range bitmask {
		bitmap[r] = make([][3]byte, len(row))
		for c, b := range row {
			if b {
				bitmap[r][c] = color
			} else {
				bitmap[r][c] = [3]byte{0, 0, 0}
			}
		}
	}
	return bitmap
}

func (matrix *matrix) OffsetBitmask(bitmask [][]bool, offset int, length int) [][]bool {
	ret := make([][]bool, len(bitmask))
	for r, row := range bitmask {
		end := offset + length
		if end > len(row) {
			end = len(row)
		}
		ret[r] = make([]bool, end-offset)
		for c, b := range row[offset:end] {
			ret[r][c] = b
		}
	}
	return ret
}
