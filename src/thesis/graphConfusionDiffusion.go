package main

import (
	"fmt"
	"image/color"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func generateConfusionDiffusionSpreadGraph(outputSize int) {
	records, err := readCsvReportFile("test-confusion-diffusion-spread")
	if err != nil {
		return
	}

	points := make(plotter.XYs, len(records)-1)
	for i, record := range records[1:] {
		x, _ := strconv.ParseFloat(record[0], 64)
		y, _ := strconv.ParseFloat(record[1], 64)
		points[i].X = x
		points[i].Y = y
	}

	p := plot.New()
	p.Title.Text = "Confusion & Diffusion Spread"
	p.X.Label.Text = "Iteration"
	p.Y.Label.Text = "Changed Bits"
	p.Y.Min = 0
	p.Y.Max = float64(outputSize * 8)
	applyLargePlotStyle(p)

	line, err := plotter.NewLine(points)
	if err != nil {
		fmt.Println("Could not create line:", err)
		return
	}
	line.Color = color.RGBA{R: 50, G: 120, B: 220, A: 255}
	line.Width = vg.Points(0.5)
	p.Add(line)

	half := float64(outputSize * 4)
	halfLine, err := plotter.NewLine(plotter.XYs{
		{X: 0, Y: half},
		{X: float64(len(records) - 2), Y: half},
	})
	if err != nil {
		fmt.Println("Could not create halfway line:", err)
		return
	}
	halfLine.Color = color.RGBA{R: 220, G: 50, B: 50, A: 255}
	halfLine.Width = vg.Points(1)
	halfLine.Dashes = []vg.Length{vg.Points(6), vg.Points(4)}
	p.Add(halfLine)

	err = saveGraphImage("test-confusion-diffusion-spread", p, 20*vg.Inch, 7*vg.Inch)
	if err != nil {
		return
	}
}

func generateConfusionDiffusionHistogramGraph() {
	records, err := readCsvReportFile("test-confusion-diffusion-spread-histogram")
	if err != nil {
		return
	}

	values := make(plotter.Values, len(records)-1)
	labels := make([]string, len(records)-1)
	for index, record := range records[1:] {
		if index%8 == 0 {
			labels[index] = record[0]
		} else {
			labels[index] = ""
		}
		v, _ := strconv.ParseFloat(record[1], 64)
		values[index] = v
	}

	p := plot.New()
	p.Title.Text = "Confusion & Diffusion Spread Histogram"
	p.X.Label.Text = "Changed Bits"
	p.Y.Label.Text = "Frequency"
	applyLargePlotStyle(p)

	bars, err := plotter.NewBarChart(values, vg.Points(6))
	if err != nil {
		fmt.Println("Could not create bar chart:", err)
		return
	}
	bars.Color = color.RGBA{R: 50, G: 120, B: 220, A: 255}
	p.Add(bars)
	p.NominalX(labels...)

	err = saveGraphImage("test-confusion-diffusion-spread-histogram", p, 20*vg.Inch, 7*vg.Inch)
	if err != nil {
		return
	}
}
