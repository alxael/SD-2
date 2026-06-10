package main

import (
	"fmt"
	"image/color"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func generateChangedBitsDistributionGraph() {
	records, err := readCsvReportFile("test-changed-bits-distribution")
	if err != nil {
		return
	}

	values := make(plotter.Values, len(records)-1)
	labels := make([]string, len(records)-1)
	total := 0.0
	for index, record := range records[1:] {
		if index%16 == 0 {
			labels[index] = record[0]
		} else {
			labels[index] = ""
		}
		v, _ := strconv.ParseFloat(record[1], 64)
		values[index] = v
		total += v
	}
	mean := total / float64(len(values))

	p := plot.New()
	p.Title.Text = "Distribuția numărului de biți modificați"
	p.X.Label.Text = "Indice bit"
	p.Y.Label.Text = "Număr de modificări"
	p.Y.Min = 0
	p.Y.Max = 6000
	applyLargePlotStyle(p)

	bars, err := plotter.NewBarChart(values, vg.Points(6))
	if err != nil {
		fmt.Println("Could not create bar chart:", err)
		return
	}
	bars.Color = color.RGBA{R: 50, G: 120, B: 220, A: 255}
	bars.LineStyle.Width = 0
	p.Add(bars)
	p.NominalX(labels...)

	meanLine, err := plotter.NewLine(plotter.XYs{
		{X: -0.5, Y: mean},
		{X: float64(len(values)) - 0.5, Y: mean},
	})
	if err != nil {
		fmt.Println("Could not create mean line:", err)
		return
	}
	meanLine.Color = color.RGBA{R: 220, G: 50, B: 50, A: 255}
	meanLine.Width = vg.Points(2)
	p.Add(meanLine)

	p.Legend.Add("Observat", bars)
	p.Legend.Add("Medie", meanLine)
	p.Legend.Top = true

	err = saveGraphImage("test-changed-bits-distribution", p, 20*vg.Inch, 7*vg.Inch)
	if err != nil {
		return
	}
}
