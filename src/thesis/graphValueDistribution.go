package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	vgdraw "gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func plotFromCSV(reportName string, title string, xLabel string, yLabel string, c color.RGBA) *plot.Plot {
	records, err := readCsvReportFile(reportName)
	if err != nil {
		return nil
	}

	points := make(plotter.XYs, len(records)-1)
	for i, record := range records[1:] {
		x, _ := strconv.ParseFloat(record[0], 64)
		y, _ := strconv.ParseFloat(record[1], 64)
		points[i].X = x
		points[i].Y = y
	}

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel
	applyLargePlotStyle(p)

	scatter, err := plotter.NewScatter(points)
	if err != nil {
		fmt.Println("Could not create scatter:", err)
		return nil
	}
	scatter.GlyphStyle.Color = c
	scatter.GlyphStyle.Radius = vg.Points(2)
	scatter.GlyphStyle.Shape = vgdraw.CircleGlyph{}
	p.Add(scatter)

	return p
}

func saveSideBySide(leftPlot *plot.Plot, rightPlot *plot.Plot, filename string) {
	w := 12 * vg.Inch
	h := 7 * vg.Inch

	leftCanvas := vgimg.New(w, h)
	leftPlot.Draw(vgdraw.New(leftCanvas))
	leftImg := leftCanvas.Image()

	rightCanvas := vgimg.New(w, h)
	rightPlot.Draw(vgdraw.New(rightCanvas))
	rightImg := rightCanvas.Image()

	totalWidth := leftImg.Bounds().Dx() + rightImg.Bounds().Dx()
	totalHeight := leftImg.Bounds().Dy()

	combined := image.NewRGBA(image.Rect(0, 0, totalWidth, totalHeight))
	draw.Draw(combined, combined.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(combined, image.Rect(0, 0, leftImg.Bounds().Dx(), totalHeight), leftImg, leftImg.Bounds().Min, draw.Over)
	draw.Draw(combined, image.Rect(leftImg.Bounds().Dx(), 0, totalWidth, totalHeight), rightImg, rightImg.Bounds().Min, draw.Over)

	err := os.MkdirAll("graphs", 0755)
	if err != nil {
		fmt.Println("Could not create graphs directory:", err)
		return
	}

	outFile, err := os.Create("graphs/" + filename + ".png")
	if err != nil {
		fmt.Println("Could not create graph file:", err)
		return
	}
	defer outFile.Close()

	err = png.Encode(outFile, combined)
	if err != nil {
		fmt.Println("Could not encode graph:", err)
		return
	}
}

func generateValueDistributionGraphs() {
	blue := color.RGBA{R: 50, G: 120, B: 220, A: 255}
	red := color.RGBA{R: 220, G: 50, B: 50, A: 255}

	// null distribution
	nullInput := plotFromCSV("test-value-distribution-null-input", "Null Input (Character Values)", "Character Index", "ASCII Value", blue)
	nullOutput := plotFromCSV("test-value-distribution-null-output", "Null Output (Hex Digit Values)", "Hex Character Index", "Hex Digit Value", red)
	if nullInput != nil && nullOutput != nil {
		saveSideBySide(nullInput, nullOutput, "test-value-distribution-null")
	}

	// sample distribution
	sampleInput := plotFromCSV("test-value-distribution-sample-input", "Sample Input (Character Values)", "Character Index", "ASCII Value", blue)
	sampleOutput := plotFromCSV("test-value-distribution-sample-output", "Sample Output (Hex Digit Values)", "Hex Character Index", "Hex Digit Value", red)
	if sampleInput != nil && sampleOutput != nil {
		saveSideBySide(sampleInput, sampleOutput, "test-value-distribution-sample")
	}
}
