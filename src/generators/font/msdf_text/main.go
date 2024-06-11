/******************************************************************************/
/* main.go                                                                    */
/******************************************************************************/
/*                           This file is part of:                            */
/*                                KAIJU ENGINE                                */
/*                          https://kaijuengine.org                           */
/******************************************************************************/
/* MIT License                                                                */
/*                                                                            */
/* Copyright (c) 2023-present Kaiju Engine authors (AUTHORS.md).              */
/* Copyright (c) 2015-present Brent Farris.                                   */
/*                                                                            */
/* May all those that this source may reach be blessed by the LORD and find   */
/* peace and joy in life.                                                     */
/* Everyone who drinks of this water will be thirsty again; but whoever       */
/* drinks of the water that I will give him shall never thirst; John 4:13-14  */
/*                                                                            */
/* Permission is hereby granted, free of charge, to any person obtaining a    */
/* copy of this software and associated documentation files (the "Software"), */
/* to deal in the Software without restriction, including without limitation  */
/* the rights to use, copy, modify, merge, publish, distribute, sublicense,   */
/* and/or sell copies of the Software, and to permit persons to whom the      */
/* Software is furnished to do so, subject to the following conditions:       */
/*                                                                            */
/* The above copyright, blessing, biblical verse, notice and                  */
/* this permission notice shall be included in all copies or                  */
/* substantial portions of the Software.                                      */
/*                                                                            */
/* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS    */
/* OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF                 */
/* MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.     */
/* IN NO EVENT SHALL THE /* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY    */
/* CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT  */
/* OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE      */
/* OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.                              */
/******************************************************************************/

package main

import (
	"bufio"
	"encoding/json"
	"kaiju/generators/font/font_data"
	"kaiju/klib"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const binDir = "../tools/content_tools/"

type rect struct {
	Left   float32 `json:"left"`
	Top    float32 `json:"top"`
	Right  float32 `json:"right"`
	Bottom float32 `json:"bottom"`
}

type glyph struct {
	Unicode     int32   `json:"unicode"`
	Advance     float32 `json:"advance"`
	PlaneBounds rect    `json:"planeBounds"` // The bounding box of the glyph as it should be placed on the baseline
	AtlasBounds rect    `json:"atlasBounds"` // The bounding box of the glyph in the atlas
}

type atlas struct {
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

type metrics struct {
	EmSize             float32 `json:"emSize"`
	LineHeight         float32 `json:"lineHeight"`
	Ascender           float32 `json:"ascender"`
	Descender          float32 `json:"descender"`
	UnderlineY         float32 `json:"underlineY"`
	UnderlineThickness float32 `json:"underlineThickness"`
}

type kerning struct {
	Unicode1 int32   `json:"unicode1"`
	Unicode2 int32   `json:"unicode2"`
	Advance  float32 `json:"advance"`
}

type fontData struct {
	Glyphs  []glyph   `json:"glyphs"`
	Atlas   atlas     `json:"atlas"`
	Metrics metrics   `json:"metrics"`
	Kerning []kerning `json:"kerning"`
}

func processFile(ttfName string) {
	println("Processing", ttfName)
	name := filepath.Base(ttfName)
	ttfDir := filepath.Dir(ttfName)
	ttfFile := filepath.Join(ttfName + ".ttf")
	jsonFile := filepath.Join(ttfDir, "out", name+".json")
	binFile := filepath.Join(ttfDir, "out", name+".bin")
	pngFile := filepath.Join(ttfDir, "out", name+".png")

	cmd := exec.Command(filepath.Join(binDir, "msdf-atlas-gen.exe"),
		"-font", ttfFile,
		"-pxrange", "4",
		"-size", "64",
		"-charset", filepath.Join(ttfDir, "charset.txt"),
		"-fontname", ttfName,
		"-type", "msdf",
		"-format", "png",
		"-pots",
		"-json", jsonFile,
		"-imageout", pngFile)
	out := klib.MustReturn(cmd.StdoutPipe())
	scanner := bufio.NewScanner(out)
	klib.Must(cmd.Start())
	for scanner.Scan() {
		println(scanner.Text())
	}
	klib.Must(cmd.Wait())
	jsonBin := klib.MustReturn(os.ReadFile(jsonFile))
	fout := klib.MustReturn(os.Create(binFile))
	defer fout.Close()
	var msdfData fontData
	klib.Must(json.NewDecoder(strings.NewReader(string(jsonBin))).Decode(&msdfData))
	fontData := font_data.FontData{
		Width:              msdfData.Atlas.Width,
		Height:             msdfData.Atlas.Height,
		EmSize:             msdfData.Metrics.EmSize,
		LineHeight:         msdfData.Metrics.LineHeight,
		Ascender:           msdfData.Metrics.Ascender,
		Descender:          msdfData.Metrics.Descender,
		UnderlineY:         msdfData.Metrics.UnderlineY,
		UnderlineThickness: msdfData.Metrics.UnderlineThickness,
		IsMsdf:             true,
		Glyphs:             make([]font_data.GlyphData, len(msdfData.Glyphs)),
	}
	for i, glyph := range msdfData.Glyphs {
		fontData.Glyphs[i].Unicode = glyph.Unicode
		fontData.Glyphs[i].Advance = glyph.Advance
		fontData.Glyphs[i].PlaneBounds.Left = glyph.PlaneBounds.Left
		fontData.Glyphs[i].PlaneBounds.Top = glyph.PlaneBounds.Top
		fontData.Glyphs[i].PlaneBounds.Right = glyph.PlaneBounds.Right
		fontData.Glyphs[i].PlaneBounds.Bottom = glyph.PlaneBounds.Bottom
		fontData.Glyphs[i].AtlasBounds.Left = glyph.AtlasBounds.Left
		fontData.Glyphs[i].AtlasBounds.Top = glyph.AtlasBounds.Top
		fontData.Glyphs[i].AtlasBounds.Right = glyph.AtlasBounds.Right
		fontData.Glyphs[i].AtlasBounds.Bottom = glyph.AtlasBounds.Bottom
	}
	font_data.Serialize(fontData, fout)
	os.Remove(jsonFile)
}

func main() {
	if len(os.Args) == 1 {
		panic("Expected the first argument to be the TTF file to convert")
	}
	ttfName := os.Args[1]
	dirName := filepath.Dir(ttfName)
	if s, err := os.Stat(dirName); err != nil {
		panic(err)
	} else if s.IsDir() {
		os.Mkdir(filepath.Join(dirName, "out"), os.ModePerm)
		klib.Must(filepath.Walk(dirName, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".ttf" {
				processFile(strings.TrimSuffix(path, ".ttf"))
			}
			return nil
		}))
	} else {
		processFile(strings.TrimSuffix(ttfName, ".ttf"))
	}
}
