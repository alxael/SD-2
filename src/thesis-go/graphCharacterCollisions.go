package main

import (
	"fmt"
	"image/color"
	"math"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func generateCharacterCollisionsGraph() {
	records, err := readCsvReportFile("test-character-collisions")
	if err != nil {
		return
	}

	values := make(plotter.Values, len(records)-1)
	labels := make([]string, len(records)-1)
	totalCollisions := 0.0
	for index, record := range records[1:] {
		labels[index] = record[0]
		v, _ := strconv.ParseFloat(record[1], 64)
		values[index] = v
		totalCollisions += v
	}
	outputSize := len(records) - 2 // equalBytes ranges from 0..outputSize

	p := plot.New()
	p.Title.Text = "Histograma coliziunilor de caractere"
	p.X.Label.Text = "Octeți egali"
	p.Y.Label.Text = "Număr de coliziuni"
	applyLargePlotStyle(p)

	bars, err := plotter.NewBarChart(values, vg.Points(20))
	if err != nil {
		fmt.Println("Could not create bar chart:", err)
		return
	}
	bars.Color = color.RGBA{R: 50, G: 120, B: 220, A: 255}
	p.Add(bars)
	p.NominalX(labels...)

	expected := make(plotter.XYs, outputSize+1)
	p256 := 1.0 / 256.0
	for k := 0; k <= outputSize; k++ {
		expected[k].X = float64(k)
		expected[k].Y = totalCollisions * binomialPMF(outputSize, k, p256)
	}
	expectedLine, err := plotter.NewLine(expected)
	if err != nil {
		fmt.Println("Could not create expected line:", err)
		return
	}
	expectedLine.Color = color.RGBA{R: 220, G: 50, B: 50, A: 255}
	expectedLine.Width = vg.Points(2)
	expectedPoints, err := plotter.NewScatter(expected)
	if err != nil {
		fmt.Println("Could not create expected scatter:", err)
		return
	}
	expectedPoints.GlyphStyle.Color = color.RGBA{R: 220, G: 50, B: 50, A: 255}
	expectedPoints.GlyphStyle.Radius = vg.Points(3)
	p.Add(expectedLine, expectedPoints)
	p.Legend.Add("Observat", bars)
	p.Legend.Add("Așteptat", expectedLine)
	p.Legend.Top = true

	err = saveGraphImage("test-character-collisions", p, 20*vg.Inch, 7*vg.Inch)
	if err != nil {
		return
	}
}

// binomialPMF returns C(n, k) * p^k * (1-p)^(n-k), evaluated stably in log
// space so that very small probabilities don't underflow for k >> n*p.
func binomialPMF(n, k int, p float64) float64 {
	if k < 0 || k > n {
		return 0
	}
	logC := lnFactorial(n) - lnFactorial(k) - lnFactorial(n-k)
	logP := float64(k)*math.Log(p) + float64(n-k)*math.Log(1-p)
	return math.Exp(logC + logP)
}

func lnFactorial(n int) float64 {
	s := 0.0
	for i := 2; i <= n; i++ {
		s += math.Log(float64(i))
	}
	return s
}
