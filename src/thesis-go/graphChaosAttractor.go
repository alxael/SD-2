package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"image/color"
	mathrand "math/rand"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	vgdraw "gonum.org/v1/plot/vg/draw"
)

func generateChaosAttractorGraph() {
	const orbits = 1 << 13
	const recordPerOrbit = 1 << 6

	plotAttractor("test-chaos-attractor", "Chaotic Attractor", orbits, recordPerOrbit, chaosRound, false, 280)
	plotAttractor("test-chaos-attractor-baker", "Baker's Map Attractor", orbits, recordPerOrbit, bakerRound, false, 210)
	plotAttractor("test-chaos-attractor-gingerbread", "Gingerbreadman Map Attractor", orbits, recordPerOrbit, gingerbreadRound, true, 0)
}

func bakerRound(x, y uint32) (uint32, uint32) {
	if x < 1<<31 {
		return x << 1, y >> 1
	}
	return x << 1, y>>1 | 1<<31
}

func gingerbreadRound(x, y uint32) (uint32, uint32) {
	absX := x
	if x >= 1<<31 {
		absX = -x
	}
	return absX - y, x
}

func plotAttractor(name, title string, orbits, recordPerOrbit int, step func(uint32, uint32) (uint32, uint32), signed bool, hueCenter float64) {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"
	if signed {
		p.X.Min = -0.5
		p.X.Max = 0.5
		p.Y.Min = -0.5
		p.Y.Max = 0.5
	} else {
		p.X.Min = 0
		p.X.Max = 1
		p.Y.Min = 0
		p.Y.Max = 1
	}
	applyLargePlotStyle(p)

	seed := make([]byte, 8)
	for orbit := 0; orbit < orbits; orbit++ {
		rand.Read(seed)
		x := binary.BigEndian.Uint32(seed[0:4])
		y := binary.BigEndian.Uint32(seed[4:8])

		points := make(plotter.XYs, recordPerOrbit)
		for r := 0; r < recordPerOrbit; r++ {
			x, y = step(x, y)
			if signed {
				points[r].X = float64(int32(x)) / float64(1<<32)
				points[r].Y = float64(int32(y)) / float64(1<<32)
			} else {
				points[r].X = float64(x) / float64(1<<32)
				points[r].Y = float64(y) / float64(1<<32)
			}
		}

		scatter, err := plotter.NewScatter(points)
		if err != nil {
			fmt.Println("Could not create scatter:", err)
			return
		}
		// pick a hue within +/- 30 degrees of the family center; saturation
		// and value are then modulated per-point by the (x, y) coordinates so
		// each orbit fades through its hue as the trajectory moves around the
		// attractor.
		hue := hueCenter + (mathrand.Float64()*60 - 30)
		// per-point glyph style: same hue/shape/radius, but s/v derived from
		// the normalized coordinates of the point.
		pts := points
		var xMin, xMax, yMin, yMax float64
		if signed {
			xMin, xMax, yMin, yMax = -0.5, 0.5, -0.5, 0.5
		} else {
			xMin, xMax, yMin, yMax = 0, 1, 0, 1
		}
		scatter.GlyphStyleFunc = func(i int) vgdraw.GlyphStyle {
			pt := pts[i]
			sx := (pt.X - xMin) / (xMax - xMin)
			sy := (pt.Y - yMin) / (yMax - yMin)
			// saturation rides on x, value on y; clamp into [0.4, 1] so dark
			// points still register against a white background.
			sat := 0.4 + 0.6*sx
			val := 0.4 + 0.6*sy
			return vgdraw.GlyphStyle{
				Color:  hsvToRGBA(hue, sat, val),
				Radius: vg.Points(0.4),
				Shape:  vgdraw.CircleGlyph{},
			}
		}
		p.Add(scatter)
	}

	err := saveGraphImage(name, p, 12*vg.Inch, 12*vg.Inch)
	if err != nil {
		return
	}
}

// hsvToRGBA converts an HSV triple (h in degrees, s and v in [0,1]) to an
// opaque RGBA color. Hue wraps modulo 360.
func hsvToRGBA(h, s, v float64) color.RGBA {
	h = h - 360*float64(int(h)/360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - absFloat(modFloat(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func modFloat(x, m float64) float64 {
	r := x - m*float64(int(x/m))
	if r < 0 {
		r += m
	}
	return r
}
