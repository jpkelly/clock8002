package matrix

import (
	"github.com/husl-colors/husl-go"
	"math/rand"
)

func (matrix *matrix) InitFlame() {
	matrix.flame_palette = make([][3]byte, 256)
	matrix.flame_buffer = make([][]byte, matrix.rows)
	for r, _ := range matrix.flame_buffer {
		matrix.flame_buffer[r] = make([]byte, matrix.columns)
	}

	var color [3]byte
	for i, _ := range matrix.flame_palette {
		l := float64(i) / 256.0 * 100.0
		if l > 50 {
			l = 50
		}
		r, g, b := hsluv.HsluvToRGB(float64(i)/2.0, 100, l)
		color[0] = byte(r * 255.0)
		color[1] = byte(g * 255.0)
		color[2] = byte(b * 255.0)
		matrix.flame_palette[i] = color
	}
}

func (matrix *matrix) FlameSet(r, c int) {
	matrix.flame_buffer[r][c] = byte(rand.Float32() * 255.0)
}

func (matrix *matrix) FlameClear(r, c int) {
	matrix.flame_buffer[r][c] = byte(0)
}

func (matrix *matrix) FlameSeed() {
	// Seed the bottom row
	for c, _ := range matrix.flame_buffer[matrix.rows-1] {
		matrix.flame_buffer[matrix.rows-1][c] = byte(rand.Float32() * 255.0)
	}
}

func (matrix *matrix) FlameFill() {
	if len(matrix.flame_palette) == 0 {
		matrix.InitFlame()
	}
	for r, row := range matrix.flame_buffer[0 : matrix.rows-1] {
		value := 0
		for c, _ := range row {
			if y := (r + 1); y < matrix.rows {
				value = int(matrix.flame_buffer[y][c])
				if x := (c - 1); x >= 0 {
					value += int(matrix.flame_buffer[y][x])
				}
				if x := c + 1; x < matrix.columns {
					value += int(matrix.flame_buffer[y][x])
				}
			}
			value *= 40
			value /= 129
			matrix.flame_buffer[r][c] = byte(value)
		}
	}

	for r, row := range matrix.bitmap {
		for c, _ := range row {
			matrix.bitmap[r][c] = matrix.flame_palette[matrix.flame_buffer[r][c]]
		}
	}
}
