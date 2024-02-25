/******************************************************************************/
/* editor.go                                                                   */
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

package editor

import (
	"errors"
	"kaiju/assets"
	"kaiju/assets/asset_importer"
	"kaiju/assets/asset_info"
	"kaiju/cameras"
	"kaiju/editor/cache/editor_cache"
	"kaiju/editor/content/content_opener"
	"kaiju/editor/memento"
	"kaiju/editor/project"
	"kaiju/editor/selection"
	"kaiju/editor/stages"
	"kaiju/editor/ui/content_window"
	"kaiju/editor/ui/editor_window"
	"kaiju/editor/ui/hierarchy"
	"kaiju/editor/ui/log_window"
	"kaiju/editor/ui/menu"
	"kaiju/editor/ui/project_window"
	"kaiju/editor/viewport/controls"
	"kaiju/editor/viewport/tools/deleter"
	"kaiju/editor/viewport/tools/transform_tools"
	"kaiju/engine"
	"kaiju/hid"
	"kaiju/host_container"
	"kaiju/klib"
	"kaiju/matrix"
	"kaiju/profiler"
	"kaiju/rendering"
	"kaiju/systems/logging"
	tests "kaiju/tests/rendering_tests"
	"kaiju/tools/html_preview"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	projectTemplate = "project_template.zip"
)

type Editor struct {
	container      *host_container.Container
	menu           *menu.Menu
	editorDir      string
	project        string
	history        memento.History
	cam            controls.EditorCamera
	assetImporters asset_importer.ImportRegistry
	stageManager   stages.Manager
	contentOpener  content_opener.Opener
	logWindow      *log_window.LogWindow
	selection      selection.Selection
	transformTool  transform_tools.TransformTool
	windowListing  editor_window.Listing
	// TODO:  Testing tools
	overlayCanvas rendering.Canvas
}

func (e *Editor) Closed()                               {}
func (e *Editor) Init()                                 {}
func (e *Editor) Tag() string                           { return editor_cache.MainWindow }
func (e *Editor) Container() *host_container.Container  { return e.container }
func (e *Editor) Host() *engine.Host                    { return e.container.Host }
func (e *Editor) StageManager() *stages.Manager         { return &e.stageManager }
func (e *Editor) ContentOpener() *content_opener.Opener { return &e.contentOpener }
func (e *Editor) Selection() *selection.Selection       { return &e.selection }
func (e *Editor) History() *memento.History             { return &e.history }
func (e *Editor) WindowListing() *editor_window.Listing { return &e.windowListing }

func addConsole(host *engine.Host) {
	html_preview.SetupConsole(host)
	profiler.SetupConsole(host)
	tests.SetupConsole(host)
}

func New() *Editor {
	logStream := logging.Initialize(nil)
	ed := &Editor{
		assetImporters: asset_importer.NewImportRegistry(),
		editorDir:      filepath.Clean(filepath.Dir(klib.MustReturn(os.Executable())) + "/.."),
		history:        memento.NewHistory(100),
	}
	ed.container = host_container.New("Kaiju Editor", logStream)
	host := ed.container.Host
	editor_window.OpenWindow(ed,
		engine.DefaultWindowWidth, engine.DefaultWindowHeight, -1, -1)
	ed.container.RunFunction(func() {
		addConsole(ed.container.Host)
	})
	host.SetFrameRateLimit(60)
	tc := cameras.ToTurntable(host.Camera.(*cameras.StandardCamera))
	host.Camera = tc
	tc.SetYawPitchZoom(0, -25, 16)
	ed.stageManager = stages.NewManager(host, &ed.assetImporters, &ed.history)
	ed.selection = selection.New(host, &ed.history)
	ed.assetImporters.Register(asset_importer.OBJImporter{})
	ed.assetImporters.Register(asset_importer.PNGImporter{})
	ed.assetImporters.Register(asset_importer.StageImporter{})
	ed.contentOpener = content_opener.New(
		&ed.assetImporters, ed.container, &ed.history)
	ed.contentOpener.Register(content_opener.ObjOpener{})
	ed.contentOpener.Register(content_opener.StageOpener{})
	return ed
}

func (e *Editor) setProject(project string) error {
	project = strings.TrimSpace(project)
	if project == "" {
		return errors.New("target project is not possible")
	}
	if _, err := os.Stat(project); os.IsNotExist(err) {
		return err
	}
	e.project = project
	if err := os.Chdir(project); err != nil {
		return err
	}
	return asset_info.InitForCurrentProject()
}

func (e *Editor) setupViewportGrid() {
	const gridCount = 20
	const halfGridCount = gridCount / 2
	points := make([]matrix.Vec3, 0, gridCount*4)
	for i := -halfGridCount; i <= halfGridCount; i++ {
		fi := float32(i)
		points = append(points, matrix.Vec3{fi, 0, -halfGridCount})
		points = append(points, matrix.Vec3{fi, 0, halfGridCount})
		points = append(points, matrix.Vec3{-halfGridCount, 0, fi})
		points = append(points, matrix.Vec3{halfGridCount, 0, fi})
	}
	host := e.Host()
	grid := rendering.NewMeshGrid(host.MeshCache(), "viewport_grid",
		points, matrix.Color{0.5, 0.5, 0.5, 1})
	shader := host.ShaderCache().ShaderFromDefinition(assets.ShaderDefinitionGrid)
	host.Drawings.AddDrawing(&rendering.Drawing{
		Renderer: host.Window.Renderer,
		Shader:   shader,
		Mesh:     grid,
		CanvasId: "default",
		ShaderData: &rendering.ShaderDataBasic{
			ShaderDataBase: rendering.NewShaderDataBase(),
			Color:          matrix.Color{0.5, 0.5, 0.5, 1},
		},
	})
}

func (e *Editor) OpenProject() {
	cx, cy := e.Host().Window.Center()
	projectWindow, _ := project_window.New(
		filepath.Join(e.editorDir, projectTemplate), cx, cy)
	projectPath := <-projectWindow.Selected
	if projectPath == "" {
		return
	}
	e.pickProject(projectPath)
}

func (e *Editor) pickProject(projectPath string) {
	if err := e.setProject(projectPath); err != nil {
		return
	}
	project.ScanContent(&e.assetImporters)
}

func (e *Editor) SetupUI() {
	cx, cy := e.Host().Window.Center()
	projectWindow, _ := project_window.New(projectTemplate, cx, cy)
	projectPath := <-projectWindow.Selected
	if projectPath == "" {
		e.Host().Close()
		return
	}
	e.Host().CreatingEditorEntities()
	e.logWindow = log_window.New(e.Host().LogStream)
	e.menu = menu.New(e.container, e.logWindow, &e.contentOpener, e)
	e.setupViewportGrid()
	{
		// TODO:  Testing tools
		win := e.Host().Window
		ot := &rendering.OITCanvas{}
		ot.Initialize(win.Renderer, float32(win.Width()), float32(win.Height()))
		ot.Create(win.Renderer)
		win.Renderer.RegisterCanvas("editor_overlay", ot)
		dc := e.Host().Window.Renderer.DefaultCanvas()
		dc.(*rendering.OITCanvas).ClearColor = matrix.ColorTransparent()
		ot.ClearColor = matrix.ColorTransparent()
		e.overlayCanvas = ot
		e.transformTool = transform_tools.New(e.Host(),
			&e.selection, "editor_overlay", &e.history)
		e.selection.Changed.Add(func() {
			e.transformTool.Disable()
		})
	}
	e.Host().DoneCreatingEditorEntities()
	e.Host().Updater.AddUpdate(e.update)
	e.windowListing.Add(e)
	e.pickProject(projectPath)
}

func (ed *Editor) update(delta float64) {
	if ed.cam.Update(ed.Host(), delta) {
		return
	}
	if ed.transformTool.Update(ed.Host()) {
		return
	}
	ed.selection.Update(ed.Host())
	kb := &ed.Host().Window.Keyboard
	if kb.KeyDown(hid.KeyboardKeyF) && ed.selection.HasSelection() {
		b := ed.selection.Bounds()
		c := ed.Host().Camera.(*cameras.TurntableCamera)
		c.SetLookAt(b.Center.Negative())
		z := b.Extent.Length()
		if z <= 0.01 {
			z = 5
		} else {
			z *= 2
		}
		c.SetZoom(z)
	} else if kb.KeyDown(hid.KeyboardKeyG) {
		ed.transformTool.Enable(transform_tools.ToolStateMove)
	} else if kb.KeyDown(hid.KeyboardKeyR) {
		ed.transformTool.Enable(transform_tools.ToolStateRotate)
	} else if kb.KeyDown(hid.KeyboardKeyS) {
		ed.transformTool.Enable(transform_tools.ToolStateScale)
	} else if kb.HasCtrl() {
		if kb.KeyDown(hid.KeyboardKeyZ) {
			ed.history.Undo()
		} else if kb.KeyDown(hid.KeyboardKeyY) {
			ed.history.Redo()
		} else if kb.KeyUp(hid.KeyboardKeySpace) {
			content_window.New(&ed.contentOpener, ed)
		} else if kb.KeyUp(hid.KeyboardKeyH) {
			hierarchy.New(ed)
		} else if kb.KeyUp(hid.KeyboardKeyS) {
			ed.stageManager.Save()
		} else if kb.KeyUp(hid.KeyboardKeyP) {
			ed.selection.Parent(&ed.history)
		}
	} else if kb.KeyDown(hid.KeyboardKeyDelete) {
		deleter.DeleteSelected(&ed.history, &ed.selection,
			slices.Clone(ed.selection.Entities()))
	}
}

func (e *Editor) SaveLayout() {
	e.windowListing.CloseAll()
	if err := editor_cache.SaveWindowCache(); err != nil {
		slog.Error("Failed to save the window cache", slog.String("error", err.Error()))
	}
}
