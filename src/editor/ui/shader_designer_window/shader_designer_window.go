package shader_designer_window

import (
	"kaiju/host_container"
	"kaiju/markup"
	"kaiju/markup/document"
	"kaiju/ui"
)

type ShaderDesignerData struct {
	Vert                    string
	Frag                    string
	Geom                    string
	Tese                    string
	Tesc                    string
	Topology                string
	PrimitiveRestart        bool
	DepthClampEnable        bool
	RasterizerDiscardEnable bool
	PolygonMode             string
	CullMode                string
	FrontFace               string
	DepthBiasEnable         bool
	DepthBiasConstantFactor string
	DepthBiasClamp          string
	DepthBiasSlopeFactor    string
	LineWidth               string
	RasterizationSamples    string
	SampleShadingEnable     bool
	MinSampleShading        string
	AlphaToCoverageEnable   bool
	AlphaToOneEnable        bool
	LogicOpEnable           bool
	LogicOp                 string
	BlendConstants0         string
	BlendConstants1         string
	BlendConstants2         string
	BlendConstants3         string
	DepthTestEnable         bool
	DepthWriteEnable        bool
	DepthCompareOp          string
	DepthBoundsTestEnable   bool
	StencilTestEnable       bool
	FrontFailOp             string
	FrontPassOp             string
	FrontDepthFailOp        string
	FrontCompareOp          string
	FrontCompareMask        string
	FrontWriteMask          string
	FrontReference          string
	BackFailOp              string
	BackPassOp              string
	BackDepthFailOp         string
	BackCompareOp           string
	BackCompareMask         string
	BackWriteMask           string
	BackReference           string
	MinDepthBounds          string
	MaxDepthBounds          string
	PatchControlPoints      string
}

func New() {
	const html = "editor/ui/shader_designer/shader_designer_window.html"
	container := host_container.New("Shader Designer", nil)
	uiMan := ui.Manager{}
	uiMan.Init(container.Host)
	go container.Run(640, 480, -1, -1)
	<-container.PrepLock
	shaderData := ShaderDesignerData{}
	container.RunFunction(func() {
		markup.DocumentFromHTMLAsset(&uiMan, html, shaderData, map[string]func(*document.Element){
			//"openContributions": openContributions,
		})
	})
}
