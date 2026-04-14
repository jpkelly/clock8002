package bdf

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type Bdf struct {
	Chars    int
	Width    int
	Height   int
	Baseline int
	X_offset int
	glyphs   map[rune]Glyph
}

type Glyph struct {
	Width    int
	Height   int
	Y_offset int
	Bitmap   [][]bool
}

func Parse(filename string) (*Bdf, error) {
	var font = Bdf{
		Chars:    -1,
		Width:    0,
		Height:   0,
		Baseline: 0,
		X_offset: 0,
		glyphs:   make(map[rune]Glyph),
	}

	if file, err := os.Open(filename); err == nil {
		defer file.Close()
		r := bufio.NewReader(file)
		for {
			line, _, err := r.ReadLine()
			if err == io.EOF {
				return nil, err
			}
			s := string(line[:])

			if n, _ := fmt.Sscanf(s, "FONTBOUNDINGBOX %d %d %d %d", &font.Width, &font.Height, &font.X_offset, &font.Baseline); n == 4 {
				// Parsed
			} else if n, _ := fmt.Sscanf(s, "CHARS %d", &font.Chars); n == 1 {
				// Parsed
			} else if strings.Contains(s, "STARTCHAR") {
				codepoint, glyph, _ := parseGlyph(r)
				font.glyphs[rune(codepoint)] = *glyph
			}
			if font.Chars == len(font.glyphs) {
				return &font, nil
			}
		}
	}
	return nil, nil
}

func parseGlyph(r *bufio.Reader) (int, *Glyph, error) {
	var codepoint int
	var x_offset int
	row := 0
	var glyph = Glyph{
		Height: 0,
		Width:  0,
	}
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return codepoint, &glyph, err
		}
		s := string(line[:])

		if n, _ := fmt.Sscanf(s, "ENCODING %d", &codepoint); n == 1 {
			// Parsed
		} else if n, _ := fmt.Sscanf(s, "BBX %d %d %d %d", &glyph.Width, &glyph.Height, &x_offset, &glyph.Y_offset); n == 4 {
			// Parsed
		} else if strings.Contains(s, "BITMAP") {
			// Parse the glyph bitmap
			var bits int
			for {
				line, _, err := r.ReadLine()
				s := string(line[:])
				if err != nil {
					return codepoint, &glyph, err
				}
				if row < glyph.Height {
					if _, err := fmt.Sscanf(s, "%X", &bits); err != nil {
						return codepoint, &glyph, err
					}
					bitmap_row := make([]bool, glyph.Width)
					first_bit := 8*((glyph.Width+7)/8) + x_offset
					for i := 0; i < glyph.Width; i++ {
						bitmap_row[i] = hasBit(bits, uint(first_bit-i))
					}
					glyph.Bitmap = append(glyph.Bitmap, bitmap_row)
					row++
				} else if strings.Contains(s, "ENDCHAR") && row == glyph.Height {
					return codepoint, &glyph, nil
				} else if row > glyph.Height {
					panic("Malformed glyph")
				}
			}
		}
	}
}

func (font *Bdf) TextBitmap(text string) [][]bool {
	bitmap := make([][]bool, font.Height)
	for _, c := range text {
		glyph := font.GetGlyph(c)
		for r, row := range glyph.Bitmap {
			bitmap[r] = append(bitmap[r], row...)
		}
	}
	return bitmap
}

func (font *Bdf) TextLength(text string) int {
	length := 0
	for _, c := range text {
		length += font.GetGlyph(c).Width
	}
	return length
}

func (font *Bdf) GetGlyph(c rune) Glyph {
	glyph, ok := font.glyphs[c]
	if !ok {
		return font.glyphs[rune(0)]
	}
	return glyph
}

func hasBit(n int, pos uint) bool {
	val := n & (1 << pos)
	return (val > 0)
}
