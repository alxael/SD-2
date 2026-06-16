package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	vgdraw "gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// spectrumColor maps t in [0, 1] to a colour by sweeping the hue across the
// full RGB spectrum, from pure blue (t = 0) through cyan, green and yellow to
// pure red (t = 1). Combined with the linear normalisation of the plotted
// values, the smallest value renders blue and the largest renders red, with
// every fully saturated hue in between keeping subtle differences visible.
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

// spectrumColorMap implements palette.ColorMap over the blue-to-red spectrum
// used by spectrumColor, mapping a value linearly from [min, max] onto the
// hue sweep. It lets the colour bar legend share the exact same colouring as
// the scatter points.
type spectrumColorMap struct {
	min, max, alpha float64
}

func (m *spectrumColorMap) At(value float64) (color.Color, error) {
	if value < m.min || value > m.max {
		return nil, fmt.Errorf("spectrumColorMap: value %g out of range [%g, %g]", value, m.min, m.max)
	}
	t := 0.0
	if m.max > m.min {
		t = (value - m.min) / (m.max - m.min)
	}
	c := spectrumColor(t)
	c.A = uint8(math.Round(m.alpha * 255))
	return c, nil
}

func (m *spectrumColorMap) Max() float64       { return m.max }
func (m *spectrumColorMap) SetMax(v float64)   { m.max = v }
func (m *spectrumColorMap) Min() float64       { return m.min }
func (m *spectrumColorMap) SetMin(v float64)   { m.min = v }
func (m *spectrumColorMap) Alpha() float64     { return m.alpha }
func (m *spectrumColorMap) SetAlpha(v float64) { m.alpha = v }

func (m *spectrumColorMap) Palette(colors int) palette.Palette {
	if colors < 1 {
		colors = 1
	}
	cols := make([]color.Color, colors)
	for i := range cols {
		t := 0.0
		if colors > 1 {
			t = float64(i) / float64(colors-1)
		}
		cols[i] = spectrumColor(t)
	}
	return spectrumPalette(cols)
}

// spectrumPalette is a trivial palette.Palette backed by a slice of colours.
type spectrumPalette []color.Color

func (p spectrumPalette) Colors() []color.Color { return p }

// plotLyapunovField renders the initial conditions (x, y) as a scatter, with
// each point coloured by the supplied exponent value over a blue-to-red
// spectrum normalised linearly between the observed minimum and maximum. It
// also returns a colour map describing that normalisation so a matching colour
// bar legend can be drawn alongside the plot.
func plotLyapunovField(reportName, title string, valueColumn int) (*plot.Plot, palette.ColorMap) {
	records, err := readCsvReportFile(reportName)
	if err != nil {
		return nil, nil
	}
	if len(records) < 2 {
		fmt.Println("No data rows in", reportName)
		return nil, nil
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

	// When every point shares the same exponent (e.g. baker's map), the field
	// has zero variance, so a min/max normalisation would collapse the single
	// value onto the middle of the spectrum (purple) regardless of its sign.
	// Normalise symmetrically around zero instead, so the constant value's sign
	// drives the colour: a positive exponent renders red, a negative one blue.
	if maxValue <= minValue {
		magnitude := math.Abs(maxValue)
		if magnitude == 0 {
			magnitude = 1
		}
		minValue = -magnitude
		maxValue = magnitude
	}
	span := maxValue - minValue

	colorMap := &spectrumColorMap{min: minValue, max: maxValue, alpha: 1}

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "x"
	p.Y.Label.Text = "y"
	applyLargePlotStyle(p)

	scatter, err := plotter.NewScatter(points)
	if err != nil {
		fmt.Println("Could not create scatter:", err)
		return nil, nil
	}
	scatter.GlyphStyleFunc = func(i int) vgdraw.GlyphStyle {
		t := (values[i] - minValue) / span
		return vgdraw.GlyphStyle{
			Color:  spectrumColor(t),
			Radius: vg.Points(2),
			Shape:  vgdraw.CircleGlyph{},
		}
	}
	p.Add(scatter)

	return p, colorMap
}

// saveChaoticMapGraph renders the Lyapunov field together with a vertical
// colour bar legend on its right-hand side and writes the combined image to
// graphs/<graphName>.png. The colour bar shares the field's colour map, so its
// gradient and value axis describe exactly the spectrum of the plotted points.
func saveChaoticMapGraph(graphName string, p *plot.Plot, colorMap palette.ColorMap, width, height vg.Length) error {
	if p == nil || colorMap == nil {
		return nil
	}

	if err := os.MkdirAll("graphs", 0755); err != nil {
		fmt.Println("Could not create graphs directory:", err)
		return err
	}

	// Build the colour bar as its own single-plotter plot.
	legend := plot.New()
	legend.Add(&plotter.ColorBar{ColorMap: colorMap, Vertical: true})
	legend.HideX()
	legend.Y.Padding = 0
	legend.Y.Tick.Label.Font.Size = vg.Points(14)

	img := vgimg.New(width, height)
	dc := vgdraw.New(img)

	// Reserve a strip on the right for the colour bar and its value axis.
	const barWidth = 1.4 * vg.Inch
	mainCanvas := vgdraw.Crop(dc, 0, -barWidth, 0, 0)
	// Inset the legend vertically to roughly line its bar up with the main
	// plot's data area, which is shrunk by the title and the x-axis labels.
	legendCanvas := vgdraw.Crop(dc, width-barWidth, 0, 0.6*vg.Inch, -0.6*vg.Inch)

	p.Draw(mainCanvas)
	legend.Draw(legendCanvas)

	file, err := os.Create("graphs/" + graphName + ".png")
	if err != nil {
		fmt.Println("Could not create graph file:", err)
		return err
	}
	defer file.Close()

	pngCanvas := vgimg.PngCanvas{Canvas: img}
	if _, err := pngCanvas.WriteTo(file); err != nil {
		fmt.Println("Could not save graph:", err)
		return err
	}

	return nil
}

func generateChaoticMapsGraphs() {
	const reportName = "test-chaotic-maps-gingerbreadman"

	width := 10 * vg.Inch
	height := 10 * vg.Inch

	// Column 2 is lambda1, column 3 is lambda2 (after x0, y0).
	lambda1Plot, lambda1Colors := plotLyapunovField(reportName, "Gingerbreadman map: exponentul Lyapunov λ₁", 2)
	if lambda1Plot != nil {
		saveChaoticMapGraph("test-chaotic-maps-gingerbreadman-lambda1", lambda1Plot, lambda1Colors, width, height)
	}

	lambda2Plot, lambda2Colors := plotLyapunovField(reportName, "Gingerbreadman map: exponentul Lyapunov λ₂", 3)
	if lambda2Plot != nil {
		saveChaoticMapGraph("test-chaotic-maps-gingerbreadman-lambda2", lambda2Plot, lambda2Colors, width, height)
	}

	// Baker's map.
	const bakerReport = "test-chaotic-maps-baker"

	bakerLambda1Plot, bakerLambda1Colors := plotLyapunovField(bakerReport, "Baker's map: exponentul Lyapunov λ₁", 2)
	if bakerLambda1Plot != nil {
		saveChaoticMapGraph("test-chaotic-maps-baker-lambda1", bakerLambda1Plot, bakerLambda1Colors, width, height)
	}

	bakerLambda2Plot, bakerLambda2Colors := plotLyapunovField(bakerReport, "Baker's map: exponentul Lyapunov λ₂", 3)
	if bakerLambda2Plot != nil {
		saveChaoticMapGraph("test-chaotic-maps-baker-lambda2", bakerLambda2Plot, bakerLambda2Colors, width, height)
	}

	// Composed baker -> gingerbreadman round used by hash.go.
	const hashReport = "test-chaotic-maps-hash"

	hashLambda1Plot, hashLambda1Colors := plotLyapunovField(hashReport, "Baker's map ∘ Gingerbreadman map: exponentul Lyapunov λ₁", 2)
	if hashLambda1Plot != nil {
		saveChaoticMapGraph("test-chaotic-maps-hash-lambda1", hashLambda1Plot, hashLambda1Colors, width, height)
	}

	hashLambda2Plot, hashLambda2Colors := plotLyapunovField(hashReport, "Baker's map ∘ Gingerbreadman map: exponentul Lyapunov λ₂", 3)
	if hashLambda2Plot != nil {
		saveChaoticMapGraph("test-chaotic-maps-hash-lambda2", hashLambda2Plot, hashLambda2Colors, width, height)
	}
}

// generateChaoticMapsConvergenceGraph plots the running Benettin estimate of
// both Lyapunov exponents against the iteration count (logarithmic axis), for
// the baker's map (limit +/- ln 2) and for the composed hash round (empirical
// limit estimated from the ensemble of random-start trials).
func generateChaoticMapsConvergenceGraph() {
	plotConvergenceGraph(
		"test-chaotic-maps-baker-convergence",
		"Convergența numerică a exponenților Lyapunov ai baker's map",
		math.Ln2,
		"±ln 2 (limită teoretică)",
	)

	hashLimit := math.Abs(meanLambda1("test-chaotic-maps-hash"))
	plotConvergenceGraph(
		"test-chaotic-maps-hash-convergence",
		"Convergența numerică a exponenților Lyapunov ai rundei compuse",
		hashLimit,
		"±λ* (limită numerică)",
	)
}

// meanLambda1 returns the mean of the lambda1 column over a Lyapunov-field
// report (columns x0, y0, lambda1, lambda2), used as the empirical limit of the
// composed round whose Lyapunov exponents have no simple closed form.
func meanLambda1(reportName string) float64 {
	records, err := readCsvReportFile(reportName)
	if err != nil || len(records) < 2 {
		return 0
	}

	var sum float64
	var count int
	for _, r := range records[1:] {
		if len(r) < 3 {
			continue
		}
		v, err := strconv.ParseFloat(r[2], 64)
		if err != nil {
			continue
		}
		sum += v
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// plotConvergenceGraph reads a running-estimate trace and plots both Lyapunov
// exponents against the iteration count on a logarithmic axis, together with
// dashed grey reference lines at +/- refValue.
func plotConvergenceGraph(reportName, title string, refValue float64, refLabel string) {
	records, err := readCsvReportFile(reportName)
	if err != nil {
		fmt.Println("could not read convergence report:", err)
		return
	}
	if len(records) < 2 {
		return
	}

	rows := records[1:]
	lambda1 := make(plotter.XYs, 0, len(rows))
	lambda2 := make(plotter.XYs, 0, len(rows))

	for _, r := range rows {
		if len(r) < 3 {
			continue
		}
		step, err1 := strconv.ParseFloat(r[0], 64)
		l1, err2 := strconv.ParseFloat(r[1], 64)
		l2, err3 := strconv.ParseFloat(r[2], 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		lambda1 = append(lambda1, plotter.XY{X: step, Y: l1})
		lambda2 = append(lambda2, plotter.XY{X: step, Y: l2})
	}

	if len(lambda1) == 0 {
		return
	}

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "număr de pași"
	p.Y.Label.Text = "exponent Lyapunov"
	applyLargePlotStyle(p)

	p.X.Scale = plot.LogScale{}
	p.X.Tick.Marker = plot.LogTicks{Prec: -1}

	minX := lambda1[0].X
	maxX := lambda1[len(lambda1)-1].X

	// Limit drawn as dashed grey reference lines at +/- refValue.
	referenceColor := color.RGBA{R: 120, G: 120, B: 120, A: 255}

	refPositive, err := plotter.NewLine(plotter.XYs{{X: minX, Y: refValue}, {X: maxX, Y: refValue}})
	if err == nil {
		refPositive.Color = referenceColor
		refPositive.Width = vg.Points(1.5)
		refPositive.Dashes = []vg.Length{vg.Points(6), vg.Points(4)}
	}

	refNegative, err := plotter.NewLine(plotter.XYs{{X: minX, Y: -refValue}, {X: maxX, Y: -refValue}})
	if err == nil {
		refNegative.Color = referenceColor
		refNegative.Width = vg.Points(1.5)
		refNegative.Dashes = []vg.Length{vg.Points(6), vg.Points(4)}
	}

	line1, err := plotter.NewLine(lambda1)
	if err != nil {
		return
	}
	line1.Color = color.RGBA{R: 215, G: 25, B: 28, A: 255}
	line1.Width = vg.Points(2)

	line2, err := plotter.NewLine(lambda2)
	if err != nil {
		return
	}
	line2.Color = color.RGBA{R: 44, G: 75, B: 215, A: 255}
	line2.Width = vg.Points(2)

	if refPositive != nil {
		p.Add(refPositive)
	}
	if refNegative != nil {
		p.Add(refNegative)
	}
	p.Add(line1, line2)

	p.Legend.Add("λ₁ (estimat)", line1)
	p.Legend.Add("λ₂ (estimat)", line2)
	p.Legend.Add(refLabel, refPositive)
	p.Legend.Top = true

	saveGraphImage(reportName, p, 10*vg.Inch, 7*vg.Inch)
}
