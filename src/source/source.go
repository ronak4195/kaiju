/******************************************************************************/
/* source.go                                                                  */
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
/* IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY       */
/* CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT  */
/* OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE      */
/* OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.                              */
/******************************************************************************/

package source

import (
	"kaiju/assets"
	"kaiju/engine"
	"kaiju/rendering"
	"kaiju/rendering/loaders"
	"log/slog"
)

func Main(host *engine.Host) {
	const cubeGLB = "editor/meshes/cube.glb"
	res, err := loaders.GLTF(cubeGLB, host.AssetDatabase(), host.WorkGroup())
	if err != nil {
		slog.Error("failed to load the cube mesh", "error", err.Error())
		return
	} else if !res.IsValid() || len(res.Meshes) != 1 {
		slog.Error("cube mesh data corrupted")
		return
	}
	resMesh := res.Meshes[0]
	mesh := rendering.NewMesh(resMesh.MeshName, resMesh.Verts, resMesh.Indexes)
	host.MeshCache().AddMesh(mesh)
	e := host.NewEntity()
	sd := rendering.NewShaderDataBase()
	//tex, _ := host.TextureCache().Texture(assets.TextureSquare, rendering.TextureFilterLinear)
	drawing := rendering.Drawing{
		Renderer:   host.Window.Renderer,
		Shader:     host.ShaderCache().ShaderFromDefinition(assets.ShaderDefinitionGBuffer),
		Mesh:       mesh,
		Textures:   []*rendering.Texture{}, //[]*rendering.Texture{tex},
		ShaderData: &sd,
		Transform:  &e.Transform,
		CanvasId:   "gbuffer",
	}
	host.Drawings.AddDrawing(&drawing)
}
