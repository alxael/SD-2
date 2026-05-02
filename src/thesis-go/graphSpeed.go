package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

var speedLineColors = []color.RGBA{
	{R: 0, G: 0, B: 0, A: 255},       // black
	{R: 220, G: 50, B: 50, A: 255},   // red
	{R: 50, G: 120, B: 220, A: 255},  // blue
	{R: 40, G: 180, B: 40, A: 255},   // green
	{R: 200, G: 130, B: 0, A: 255},   // orange
	{R: 150, G: 50, B: 200, A: 255},  // purple
	{R: 0, G: 160, B: 160, A: 255},   // teal
	{R: 200, G: 80, B: 140, A: 255},  // pink
	{R: 110, G: 80, B: 30, A: 255},   // brown
	{R: 100, G: 100, B: 100, A: 255}, // gray
}

type fixedTicker []plot.Tick

func (t fixedTicker) Ticks(min, max float64) []plot.Tick {
	return []plot.Tick(t)
}
func generateSpeedGraph() {
	records, err := readCsvReportFile("test-speed")
	if err != nil {
		return
	}
	if len(records) < 2 {
		return
	}

	header := records[0]
	coreCount := len(header) - 1
	coreLabels := make([]string, coreCount)
	for i := 0; i < coreCount; i++ {
		// header format: "maxMBPerSecond{N}Core"
		label := header[i+1]
		label = strings.TrimPrefix(label, "maxMBPerSecond")
		label = strings.TrimSuffix(label, "Core")
		coreLabels[i] = label
	}

	rows := records[1:]
	seriesPoints := make([]plotter.XYs, coreCount)
	for i := range seriesPoints {
		seriesPoints[i] = make(plotter.XYs, len(rows))
	}

	xValues := make([]float64, len(rows))
	for r, record := range rows {
		x, _ := strconv.ParseFloat(record[0], 64)
		xValues[r] = x
		for c := 0; c < coreCount; c++ {
			y, _ := strconv.ParseFloat(record[c+1], 64)
			seriesPoints[c][r].X = x
			seriesPoints[c][r].Y = y
		}
	}

	xTicks := make([]plot.Tick, len(xValues))
	for i, x := range xValues {
		var label string
		if x < 1 {
			label = fmt.Sprintf("%g MB", x)
		} else {
			label = fmt.Sprintf("%d MB", int(x))
		}
		xTicks[i] = plot.Tick{Value: x, Label: label}
	}

	p := plot.New()
	p.Title.Text = "Hash Throughput vs Input Size"
	p.X.Label.Text = "Input Size (MB)"
	p.Y.Label.Text = "Throughput (MB/s)"
	p.X.Scale = plot.LogScale{}
	p.X.Tick.Marker = fixedTicker(xTicks)
	applyLargePlotStyle(p)

	for i, points := range seriesPoints {
		c := speedLineColors[i%len(speedLineColors)]

		line, err := plotter.NewLine(points)
		if err != nil {
			fmt.Println("Could not create line:", err)
			return
		}
		line.Color = c
		line.Width = vg.Points(2)

		scatter, err := plotter.NewScatter(points)
		if err != nil {
			fmt.Println("Could not create scatter:", err)
			return
		}
		scatter.GlyphStyle.Color = c
		scatter.GlyphStyle.Radius = vg.Points(3)

		p.Add(line, scatter)
		p.Legend.Add(fmt.Sprintf("%s core", coreLabels[i]), line, scatter)
	}

	p.Legend.Top = true
	p.Legend.Left = true

	err = saveGraphImage("test-speed", p, 16*vg.Inch, 9*vg.Inch)
	if err != nil {
		return
	}
}
