package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	vgdraw "gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

var lineColors = []color.RGBA{
	{R: 0, G: 0, B: 0, A: 255},      // black
	{R: 220, G: 50, B: 50, A: 255},  // red
	{R: 50, G: 120, B: 220, A: 255}, // blue
	{R: 40, G: 180, B: 40, A: 255},  // green
	{R: 200, G: 130, B: 0, A: 255},  // orange
	{R: 150, G: 50, B: 200, A: 255}, // purple
}

func bitsLine(hexHash string, c color.RGBA) (*plotter.Line, error) {
	bits := hexToBits(hexHash)
	points := make(plotter.XYs, len(bits))
	for j, b := range bits {
		points[j].X = float64(j)
		points[j].Y = float64(b)
	}
	line, err := plotter.NewLine(points)
	if err != nil {
		return nil, err
	}
	line.Color = c
	line.Width = vg.Points(0.5)
	return line, nil
}

func renderPlotToImage(p *plot.Plot, w, h vg.Length) image.Image {
	canvas := vgimg.New(w, h)
	p.Draw(vgdraw.New(canvas))
	return canvas.Image()
}

func generateSensitivityGraph() {
	records, err := readCsvReportFile("test-sensitivity")
	if err != nil {
		return
	}

	initial := records[1]
	variants := records[2:] // 5 variants

	plotWidth := 20 * vg.Inch
	plotHeight := 4 * vg.Inch

	var images []image.Image

	// initial hash plot
	initialPlot := plot.New()
	initialPlot.Title.Text = fmt.Sprintf("Initial: %s", initial[0])
	initialPlot.X.Label.Text = "Bit Index"
	initialPlot.Y.Label.Text = "Bit Value"
	initialPlot.Y.Min = -0.1
	initialPlot.Y.Max = 1.1
	applyLargePlotStyle(initialPlot)

	initialLine, err := bitsLine(initial[1], lineColors[0])
	if err != nil {
		fmt.Println("Could not create line:", err)
		return
	}
	initialPlot.Add(initialLine)
	images = append(images, renderPlotToImage(initialPlot, plotWidth, plotHeight))

	for i, record := range variants {
		variantMessage := record[0]
		variantHash := record[1]
		changedBits := record[2]
		changedPct := record[3]

		p := plot.New()
		p.Title.Text = fmt.Sprintf("Variant %d: %s (changed: %s bits, %s%%)", i+1, variantMessage, changedBits, changedPct)
		p.X.Label.Text = "Bit Index"
		p.Y.Label.Text = "Bit Value"
		p.Y.Min = -0.1
		p.Y.Max = 1.1
		applyLargePlotStyle(p)

		varLine, err := bitsLine(variantHash, lineColors[i%len(lineColors)+1])
		if err != nil {
			fmt.Println("Could not create line:", err)
			return
		}

		p.Add(varLine)

		images = append(images, renderPlotToImage(p, plotWidth, plotHeight))
	}

	// stack images vertically
	totalWidth := images[0].Bounds().Dx()
	totalHeight := 0
	for _, img := range images {
		totalHeight += img.Bounds().Dy()
	}

	combined := image.NewRGBA(image.Rect(0, 0, totalWidth, totalHeight))
	draw.Draw(combined, combined.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	yOffset := 0
	for _, img := range images {
		r := image.Rect(0, yOffset, totalWidth, yOffset+img.Bounds().Dy())
		draw.Draw(combined, r, img, img.Bounds().Min, draw.Over)
		yOffset += img.Bounds().Dy()
	}

	err = os.MkdirAll("graphs", 0755)
	if err != nil {
		fmt.Println("Could not create graphs directory:", err)
		return
	}

	outFile, err := os.Create("graphs/test-sensitivity.png")
	if err != nil {
		fmt.Println("Could not create sensitivity graph file:", err)
		return
	}
	defer outFile.Close()

	err = png.Encode(outFile, combined)
	if err != nil {
		fmt.Println("Could not encode sensitivity graph:", err)
		return
	}
}
