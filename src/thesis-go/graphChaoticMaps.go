package main

import (
	"fmt"
	"image/color"
	"math"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	vgdraw "gonum.org/v1/plot/vg/draw"
)

// spectrumColor maps t in [0, 1] to a colour by sweeping the hue from pure blue
// (t = 0) through cyan, green and yellow to pure red (t = 1). Interpolating in
// hue rather than straight RGB keeps every step vivid and fully saturated,
// instead of passing through the dark purple-grey that a direct blue-to-red RGB
// blend produces, so subtle differences stay visible.
func spectrumColor(t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	// Blue sits at hue 240 degrees, red at hue 0; sweep the long way round
	// through cyan (180), green (120) and yellow (60).
	hue := 240 * (1 - t)
	return hsvToRGB(hue, 1, 1)
}

// hsvToRGB converts an HSV colour (hue in degrees, saturation and value in
// [0, 1]) into an opaque RGBA colour.
func hsvToRGB(hue, saturation, value float64) color.RGBA {
	c := value * saturation
	x := c * (1 - math.Abs(math.Mod(hue/60, 2)-1))
	m := value - c

	var r, g, b float64
	switch {
	case hue < 60:
		r, g, b = c, x, 0
	case hue < 120:
		r, g, b = x, c, 0
	case hue < 180:
		r, g, b = 0, c, x
	case hue < 240:
		r, g, b = 0, x, c
	case hue < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return color.RGBA{
		R: uint8(math.Round((r + m) * 255)),
		G: uint8(math.Round((g + m) * 255)),
		B: uint8(math.Round((b + m) * 255)),
		A: 255,
	}
}

// plotLyapunovField renders the initial conditions (x0, y0) as a scatter, with
// each point coloured by the supplied exponent value over a blue-to-red
// spectrum normalised linearly between the observed minimum and maximum.
func plotLyapunovField(reportName, title string, valueColumn int) *plot.Plot {
	records, err := readCsvReportFile(reportName)
	if err != nil {
		return nil
	}
	if len(records) < 2 {
		fmt.Println("No data rows in", reportName)
		return nil
	}

	rows := records[1:]
	points := make(plotter.XYs, len(rows))
	values := make([]float64, len(rows))

	minValue := math.Inf(1)
	maxValue := math.Inf(-1)
	for i, record := range rows {
		x, _ := strconv.ParseFloat(record[0], 64)
		y, _ := strconv.ParseFloat(record[1], 64)
		value, _ := strconv.ParseFloat(record[valueColumn], 64)

		points[i].X = x
		points[i].Y = y
		values[i] = value

		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}

	span := maxValue - minValue

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"
	applyLargePlotStyle(p)

	scatter, err := plotter.NewScatter(points)
	if err != nil {
		fmt.Println("Could not create scatter:", err)
		return nil
	}
	scatter.GlyphStyleFunc = func(i int) vgdraw.GlyphStyle {
		t := 0.0
		if span > 0 {
			t = (values[i] - minValue) / span
		}
		return vgdraw.GlyphStyle{
			Color:  spectrumColor(t),
			Radius: vg.Points(2),
			Shape:  vgdraw.CircleGlyph{},
		}
	}
	p.Add(scatter)

	return p
}

func generateChaoticMapsGraphs() {
	const reportName = "test-chaotic-maps-gingerbreadman"

	width := 10 * vg.Inch
	height := 10 * vg.Inch

	// Column 2 is lambda1, column 3 is lambda2 (after x0, y0).
	lambda1Plot := plotLyapunovField(reportName, "Gingerbreadman map: exponentul Lyapunov λ₁", 2)
	if lambda1Plot != nil {
		saveGraphImage("test-chaotic-maps-gingerbreadman-lambda1", lambda1Plot, width, height)
	}

	lambda2Plot := plotLyapunovField(reportName, "Gingerbreadman map: exponentul Lyapunov λ₂", 3)
	if lambda2Plot != nil {
		saveGraphImage("test-chaotic-maps-gingerbreadman-lambda2", lambda2Plot, width, height)
	}

	// Baker's map.
	const bakerReport = "test-chaotic-maps-baker"

	bakerLambda1Plot := plotLyapunovField(bakerReport, "Baker's map: exponentul Lyapunov λ₁", 2)
	if bakerLambda1Plot != nil {
		saveGraphImage("test-chaotic-maps-baker-lambda1", bakerLambda1Plot, width, height)
	}

	bakerLambda2Plot := plotLyapunovField(bakerReport, "Baker's map: exponentul Lyapunov λ₂", 3)
	if bakerLambda2Plot != nil {
		saveGraphImage("test-chaotic-maps-baker-lambda2", bakerLambda2Plot, width, height)
	}

	// Composed baker -> gingerbreadman round used by hash.go.
	const hashReport = "test-chaotic-maps-hash"

	hashLambda1Plot := plotLyapunovField(hashReport, "Baker's map ∘ Gingerbreadman map: exponentul Lyapunov λ₁", 2)
	if hashLambda1Plot != nil {
		saveGraphImage("test-chaotic-maps-hash-lambda1", hashLambda1Plot, width, height)
	}

	hashLambda2Plot := plotLyapunovField(hashReport, "Baker's map ∘ Gingerbreadman map: exponentul Lyapunov λ₂", 3)
	if hashLambda2Plot != nil {
		saveGraphImage("test-chaotic-maps-hash-lambda2", hashLambda2Plot, width, height)
	}
}
