package matrix

import (
	"github.com/husl-colors/husl-go"
	"math"
)

func (matrix *matrix) PlasmaBitmap(step int) [][][3]byte {
	if len(matrix.plasma_palette) != 360 {
		var color [3]byte
		matrix.plasma_palette = make([][3]byte, 360)
		for i, _ := range matrix.plasma_palette {
			r, g, b := hsluv.HsluvToRGB(float64(i), 100, 50)
			color[0] = byte(r * 255.0)
			color[1] = byte(g * 255.0)
			color[2] = byte(b * 255.0)
			matrix.plasma_palette[i] = color
		}
	}

	buffer := make([][][3]byte, matrix.rows)
	var value float64
	for r, _ := range buffer {
		buffer[r] = make([][3]byte, matrix.columns)
		for c, _ := range buffer[r] {
			value = math.Sin(dist(c+step, r, 32.0, 128.8) / 8.0)
			value += math.Sin(dist(c, r, 16.0, 64.0) / 8.0)
			value += math.Sin(dist(c, r+step/7.0, 0.0, 0.0) / 7.0)
			value += math.Sin(dist(c, r, 192.0, 100.0) / 16.0)
			buffer[r][c] = matrix.plasma_palette[(int((4+value)*45)+step)%360]
		}
	}
	return buffer
}

func (matrix *matrix) PlasmaFill(step int) {
	matrix.DrawBitmap(matrix.PlasmaBitmap(step), 0, 0)
}

func dist(y, x int, c, d float64) float64 {
	a := float64(y)
	b := float64(x)
	return math.Sqrt((a-c)*(a-c) + (b-d)*(b-d))
}
