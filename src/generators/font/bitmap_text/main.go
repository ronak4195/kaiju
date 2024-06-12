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
	"encoding/json"
	"encoding/xml"
	"kaiju/generators/font/font_data"
	"kaiju/klib"
	"os"
	"path/filepath"
	"strings"
)

type bitmapTextDataChar struct {
	Id       int32 `xml:"id,attr"`
	X        int32 `xml:"x,attr"`
	Y        int32 `xml:"y,attr"`
	Width    int32 `xml:"width,attr"`
	Height   int32 `xml:"height,attr"`
	XOffset  int32 `xml:"xoffset,attr"`
	YOffset  int32 `xml:"yoffset,attr"`
	XAdvance int32 `xml:"xadvance,attr"`
	Page     int32 `xml:"page,attr"`
	Channel  int32 `xml:"chnl,attr"`
}

type bitmapTextDataChars struct {
	Chars []bitmapTextDataChar `xml:"char"`
}

type bitmapTextDataInfo struct {
	Face     string  `xml:"face,attr"`
	Size     int32   `xml:"size,attr"`
	Bold     bool    `xml:"bold,attr"`
	Italic   bool    `xml:"italic,attr"`
	Charset  string  `xml:"charset,attr"`
	Unicode  bool    `xml:"unicode,attr"`
	StretchH int32   `xml:"stretchH,attr"`
	Smooth   float32 `xml:"smooth,attr"`
	Aa       float32 `xml:"aa,attr"`
	Padding  string  `xml:"padding,attr"`
	Spacing  string  `xml:"spacing,attr"`
}

type bitmapTextDataCommon struct {
	LineHeight int32 `xml:"lineHeight,attr"`
	Base       int32 `xml:"base,attr"`
	ScaleW     int32 `xml:"scaleW,attr"`
	ScaleH     int32 `xml:"scaleH,attr"`
	Pages      int32 `xml:"pages,attr"`
	Packed     bool  `xml:"packed,attr"`
}

type BitmapTextData struct {
	CharData bitmapTextDataChars  `xml:"chars"`
	Info     bitmapTextDataInfo   `xml:"info"`
	Common   bitmapTextDataCommon `xml:"common"`
}

func processFile(fontFile string) {
	text, err := os.ReadFile(fontFile)
	if err != nil {
		panic(err)
	}
	data := BitmapTextData{}
	if err := xml.Unmarshal(text, &data); err != nil {
		panic(err)
	}
	font := strings.TrimSuffix(fontFile, ".xml")
	binFile := filepath.Join(filepath.Dir(font), "out", filepath.Base(font)+".bin")
	fontData := font_data.FontData{
		Width:              data.Common.ScaleW,
		Height:             data.Common.ScaleH,
		EmSize:             1, //float32(data.Info.Size),
		LineHeight:         float32(data.Info.Size) / float32(data.Common.LineHeight),
		Ascender:           0,
		Descender:          0,
		UnderlineY:         0,
		UnderlineThickness: 0,
		IsMsdf:             false,
		Glyphs:             make([]font_data.GlyphData, len(data.CharData.Chars)),
	}
	for i, c := range data.CharData.Chars {
		fontData.Glyphs[i].Unicode = c.Id
		fontData.Glyphs[i].PlaneBounds.Left = 0
		fontData.Glyphs[i].PlaneBounds.Top = 0
		fontData.Glyphs[i].PlaneBounds.Right = 1
		fontData.Glyphs[i].PlaneBounds.Bottom = 1
		fontData.Glyphs[i].AtlasBounds.Left = float32(c.X)
		fontData.Glyphs[i].AtlasBounds.Top = float32(c.Y)
		fontData.Glyphs[i].AtlasBounds.Right = float32(c.X) + float32(c.Width)
		fontData.Glyphs[i].AtlasBounds.Bottom = float32(c.Y) + float32(c.Height)
		fontData.Glyphs[i].Advance = float32(c.XAdvance)
	}
	jsonMap := strings.TrimSuffix(strings.TrimSuffix(binFile, ".bin"), "-bmp") + ".json"
	if _, err := os.Stat(jsonMap); err == nil {
		var msdfData font_data.MsdfData
		jsonBin := klib.MustReturn(os.ReadFile(jsonMap))
		klib.Must(json.Unmarshal(jsonBin, &msdfData))
		tmpMap := map[int32]int{}
		fontData.LineHeight = msdfData.Metrics.LineHeight
		fontData.EmSize = msdfData.Metrics.EmSize
		for i := range msdfData.Glyphs {
			tmpMap[msdfData.Glyphs[i].Unicode] = i
		}
		for i := range fontData.Glyphs {
			target := msdfData.Glyphs[tmpMap[fontData.Glyphs[i].Unicode]]
			pb := font_data.GlyphRect(target.PlaneBounds)
			// Top and bottom are inverted for some reason
			pb.Top, pb.Bottom = pb.Bottom, pb.Top
			fontData.Glyphs[i].PlaneBounds = pb
			fontData.Glyphs[i].Advance = target.Advance
		}
		os.Remove(jsonMap)
	}
	fout := klib.MustReturn(os.Create(binFile))
	defer fout.Close()
	font_data.Serialize(fontData, fout)
}

// Currently I'm using https://snowb.org/ to generate bitmap xml files
func main() {
	if len(os.Args) == 1 {
		panic("Expected the first argument to be the font description xml file")
	}
	font := os.Args[1]
	if s, err := os.Stat(font); err != nil {
		panic(err)
	} else if s.IsDir() {
		os.Mkdir(filepath.Join(font, "out"), os.ModePerm)
		klib.Must(filepath.Walk(font, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".xml" {
				processFile(path)
			}
			return nil
		}))
	} else {
		processFile(font)
	}
}
