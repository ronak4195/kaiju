/******************************************************************************/
/* font_data.go                                                               */
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

package font_data

import (
	"encoding/binary"
	"os"
)

type GlyphRect struct {
	Left   float32
	Top    float32
	Right  float32
	Bottom float32
}

type GlyphData struct {
	Unicode     int32
	Advance     float32
	PlaneBounds GlyphRect // The bounding box of the glyph as it should be placed on the baseline
	AtlasBounds GlyphRect // The bounding box of the glyph in the atlas
}

type FontData struct {
	Width              int32
	Height             int32
	EmSize             float32
	LineHeight         float32
	Ascender           float32
	Descender          float32
	UnderlineY         float32
	UnderlineThickness float32
	IsMsdf             bool
	Glyphs             []GlyphData
}

func Serialize(fontData FontData, fout *os.File) {
	binary.Write(fout, binary.LittleEndian, int32(len(fontData.Glyphs)))
	binary.Write(fout, binary.LittleEndian, fontData.Width)
	binary.Write(fout, binary.LittleEndian, fontData.Height)
	binary.Write(fout, binary.LittleEndian, fontData.EmSize)
	binary.Write(fout, binary.LittleEndian, fontData.LineHeight)
	binary.Write(fout, binary.LittleEndian, fontData.Ascender)
	binary.Write(fout, binary.LittleEndian, fontData.Descender)
	binary.Write(fout, binary.LittleEndian, fontData.UnderlineY)
	binary.Write(fout, binary.LittleEndian, fontData.UnderlineThickness)
	for _, glyph := range fontData.Glyphs {
		binary.Write(fout, binary.LittleEndian, int32(glyph.Unicode))
		binary.Write(fout, binary.LittleEndian, glyph.Advance)
		binary.Write(fout, binary.LittleEndian, glyph.PlaneBounds.Left)
		binary.Write(fout, binary.LittleEndian, glyph.PlaneBounds.Top)
		binary.Write(fout, binary.LittleEndian, glyph.PlaneBounds.Right)
		binary.Write(fout, binary.LittleEndian, glyph.PlaneBounds.Bottom)
		binary.Write(fout, binary.LittleEndian, glyph.AtlasBounds.Left)
		binary.Write(fout, binary.LittleEndian, glyph.AtlasBounds.Top)
		binary.Write(fout, binary.LittleEndian, glyph.AtlasBounds.Right)
		binary.Write(fout, binary.LittleEndian, glyph.AtlasBounds.Bottom)
	}
	binary.Write(fout, binary.LittleEndian, fontData.IsMsdf)
}
