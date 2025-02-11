package rendering

import (
	"errors"
	"kaiju/assets"
	"kaiju/matrix"
	vk "kaiju/rendering/vulkan"
	"log/slog"
)

type GBufferCanvas struct {
	position       TextureId
	normal         TextureId
	albedo         TextureId
	depth          TextureId
	gbufferPass    RenderPass
	lightingPass   RenderPass
	texture        Texture
	lightingShader *Shader
}

func (g *GBufferCanvas) Create(renderer Renderer) error {
	if !g.createImages(renderer) || !g.createRenderPass(renderer) || !g.createLightingRenderPass(renderer) {
		return errors.New("failed to create the deferred rendering")
	}
	g.texture.RenderId = g.albedo
	return nil
}

func (g *GBufferCanvas) Initialize(renderer Renderer) {
	vr := renderer.(*Vulkan)
	g.lightingShader = vr.caches.ShaderCache().ShaderFromDefinition(assets.ShaderDefinitionLighting)
}

func (g *GBufferCanvas) Draw(renderer Renderer, drawings []ShaderDraw) {
	vr := renderer.(*Vulkan)
	frame := vr.currentFrame
	cmdBuffIdx := frame * MaxCommandBuffers
	for i := range drawings {
		vr.writeDrawingDescriptors(drawings[i].shader, drawings[i].instanceGroups)
	}
	cmd1 := vr.commandBuffers[cmdBuffIdx+vr.commandBuffersCount]
	vr.commandBuffersCount++
	var opaqueClear [4]vk.ClearValue
	opaqueClear[0].SetColor([]float32{0, 0, 0, 1})
	opaqueClear[1].SetColor([]float32{0, 0, 0, 1})
	opaqueClear[2].SetColor([]float32{0, 0, 0, 1})
	opaqueClear[3].SetDepthStencil(1.0, 0.0)
	beginRender(g.gbufferPass, vr.swapChainExtent, cmd1, opaqueClear[:])
	for i := range drawings {
		vr.renderEach(cmd1, drawings[i].shader, drawings[i].instanceGroups)
	}
	endRender(cmd1)

	// TODO:  We need to deal with transparent things

	mesh := NewMeshQuad(vr.caches.MeshCache())
	sd := NewShaderDataBase()
	m := matrix.Mat4Identity()
	m.Scale(matrix.Vec3{15, 15, 15})
	sd.SetModel(m)
	ig := []DrawInstanceGroup{NewDrawInstanceGroup(mesh, sd.Size())}
	ig[0].Textures = make([]*Texture, 4)
	for i := range len(ig[0].Textures) {
		ig[0].Textures[i] = &Texture{}
	}
	ig[0].Textures[0].RenderId = g.position
	ig[0].Textures[1].RenderId = g.normal
	ig[0].Textures[2].RenderId = g.albedo
	ig[0].Textures[3].RenderId = g.depth
	ig[0].AddInstance(&sd, g.lightingShader)
	vr.writeDrawingDescriptors(g.lightingShader, ig)
	cmd2 := vr.commandBuffers[cmdBuffIdx+vr.commandBuffersCount]
	vr.commandBuffersCount++
	beginRender(g.lightingPass, vr.swapChainExtent, cmd2, opaqueClear[:])
	vr.renderEach(cmd2, g.lightingShader, ig)
	endRender(cmd2)
}

func (g *GBufferCanvas) Pass(name string) *RenderPass {
	if name == "lighting" {
		return &g.lightingPass
	} else {
		return &g.gbufferPass
	}
}

func (g *GBufferCanvas) Color() *Texture {
	return &g.texture
}

func (g *GBufferCanvas) ShaderPipeline(name string) FuncPipeline {
	if name == "lighting" {
		return g.createLightingPipeline
	} else {
		return g.createGBufferPipeline
	}
}

func (g *GBufferCanvas) Destroy(renderer Renderer) {
	vr := renderer.(*Vulkan)
	vk.DeviceWaitIdle(vr.device)
	g.gbufferPass.Destroy()
	vr.textureIdFree(&g.position)
	vr.textureIdFree(&g.normal)
	vr.textureIdFree(&g.albedo)
	vr.textureIdFree(&g.depth)
	g.position = TextureId{}
	g.normal = TextureId{}
	g.albedo = TextureId{}
	g.depth = TextureId{}
}

func (g *GBufferCanvas) createImages(renderer Renderer) bool {
	vr := renderer.(*Vulkan)
	w := uint32(vr.swapChainExtent.Width)
	h := uint32(vr.swapChainExtent.Height)
	samples := vk.SampleCount1Bit
	//VkSampleCountFlagBits samples = vr.msaaSamples;
	imagesCreated := true
	imgs := [...]*TextureId{&g.position, &g.normal, &g.albedo}
	fmts := [...]vk.Format{vk.FormatR32g32b32a32Sfloat, vk.FormatR16g16b16a16Sfloat, vk.FormatR8g8b8a8Unorm}
	for i := range imgs {
		imagesCreated = imagesCreated && vr.CreateImage(w, h, 1, samples,
			fmts[i], vk.ImageTilingOptimal,
			vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit|vk.ImageUsageTransferSrcBit|vk.ImageUsageSampledBit),
			vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit), imgs[i], 1)
		imagesCreated = imagesCreated && vr.createImageView(imgs[i],
			vk.ImageAspectFlags(vk.ImageAspectColorBit))
		if imagesCreated {
			vr.createTextureSampler(&imgs[i].Sampler, 1, vk.FilterLinear)
		}
	}
	// Create the depth image
	depthFormat := vr.findDepthFormat()
	imagesCreated = imagesCreated && vr.CreateImage(w, h, 1,
		samples, depthFormat, vk.ImageTilingOptimal,
		vk.ImageUsageFlags(vk.ImageUsageDepthStencilAttachmentBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit), &g.depth, 1)
	imagesCreated = imagesCreated && vr.createImageView(&g.depth,
		vk.ImageAspectFlags(vk.ImageAspectDepthBit))
	vr.createTextureSampler(&g.depth.Sampler, 1, vk.FilterLinear)
	// TODO:  Is this needed?
	if imagesCreated {
		for i := range imgs {
			vr.transitionImageLayout(imgs[i],
				vk.ImageLayoutColorAttachmentOptimal, vk.ImageAspectFlags(vk.ImageAspectColorBit),
				vk.AccessFlags(vk.AccessColorAttachmentWriteBit), vk.NullCommandBuffer)
		}
		vr.transitionImageLayout(&g.depth,
			vk.ImageLayoutDepthStencilAttachmentOptimal, vk.ImageAspectFlags(vk.ImageAspectDepthBit),
			vk.AccessFlags(vk.AccessDepthStencilAttachmentWriteBit), vk.NullCommandBuffer)
	}
	return imagesCreated
}

func (g *GBufferCanvas) createRenderPass(renderer Renderer) bool {
	vr := renderer.(*Vulkan)
	var attachments [4]vk.AttachmentDescription
	imgs := [...]*TextureId{&g.position, &g.normal, &g.albedo}
	attachRefs := [3]vk.AttachmentReference{}
	for i := range imgs {
		attachments[i].Format = imgs[i].Format
		attachments[i].Samples = imgs[i].Samples
		attachments[i].LoadOp = vk.AttachmentLoadOpClear
		attachments[i].StoreOp = vk.AttachmentStoreOpStore
		attachments[i].StencilLoadOp = vk.AttachmentLoadOpDontCare
		attachments[i].StencilStoreOp = vk.AttachmentStoreOpDontCare
		attachments[i].InitialLayout = vk.ImageLayoutColorAttachmentOptimal
		attachments[i].FinalLayout = vk.ImageLayoutColorAttachmentOptimal
		attachments[i].Flags = 0
		attachRefs[i].Attachment = uint32(i)
		attachRefs[i].Layout = vk.ImageLayoutColorAttachmentOptimal
	}

	// Depth attachment
	attachments[3].Format = g.depth.Format
	attachments[3].Samples = g.depth.Samples
	attachments[3].LoadOp = vk.AttachmentLoadOpClear
	attachments[3].StoreOp = vk.AttachmentStoreOpStore
	attachments[3].StencilLoadOp = vk.AttachmentLoadOpDontCare
	attachments[3].StencilStoreOp = vk.AttachmentStoreOpDontCare
	attachments[3].InitialLayout = vk.ImageLayoutDepthStencilAttachmentOptimal
	attachments[3].FinalLayout = vk.ImageLayoutDepthStencilAttachmentOptimal
	attachments[3].Flags = 0

	// Depth attachment reference
	depthAttachmentRef := vk.AttachmentReference{}
	depthAttachmentRef.Attachment = 3
	depthAttachmentRef.Layout = vk.ImageLayoutDepthStencilAttachmentOptimal

	// 1 subpass
	subpass := vk.SubpassDescription{}
	subpass.PipelineBindPoint = vk.PipelineBindPointGraphics
	subpass.ColorAttachmentCount = uint32(len(attachRefs))
	subpass.PColorAttachments = &attachRefs[0]
	subpass.PDepthStencilAttachment = &depthAttachmentRef

	pass, err := NewRenderPass(vr.device, &vr.dbg, attachments[:],
		[]vk.SubpassDescription{subpass}, nil)
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	g.gbufferPass = pass
	err = g.gbufferPass.CreateFrameBuffer(vr,
		[]vk.ImageView{g.position.View, g.normal.View, g.albedo.View, g.depth.View},
		g.position.Width, g.position.Height)
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	return true
}

func (g *GBufferCanvas) createLightingRenderPass(renderer Renderer) bool {
	vr := renderer.(*Vulkan)
	var colorAttachment vk.AttachmentDescription
	colorAttachment.Format = vr.swapChainFormat
	colorAttachment.Samples = vk.SampleCount1Bit
	colorAttachment.LoadOp = vk.AttachmentLoadOpClear
	colorAttachment.StoreOp = vk.AttachmentStoreOpStore
	colorAttachment.StencilLoadOp = vk.AttachmentLoadOpDontCare
	colorAttachment.StencilStoreOp = vk.AttachmentStoreOpDontCare
	colorAttachment.InitialLayout = vk.ImageLayoutUndefined
	colorAttachment.FinalLayout = vk.ImageLayoutPresentSrc
	colorAttachment.Flags = 0

	colorAttachmentRef := vk.AttachmentReference{}
	colorAttachmentRef.Attachment = 0
	colorAttachmentRef.Layout = vk.ImageLayoutColorAttachmentOptimal

	// 1 subpass for lighting
	subpass := vk.SubpassDescription{}
	subpass.PipelineBindPoint = vk.PipelineBindPointGraphics
	subpass.ColorAttachmentCount = 1
	subpass.PColorAttachments = &colorAttachmentRef
	subpass.PDepthStencilAttachment = nil

	// Create the render pass
	attachments := []vk.AttachmentDescription{colorAttachment}
	pass, err := NewRenderPass(vr.device, &vr.dbg, attachments,
		[]vk.SubpassDescription{subpass}, nil)
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	g.lightingPass = pass
	err = g.lightingPass.CreateFrameBuffer(vr,
		[]vk.ImageView{vr.swapImages[0].View},
		int(vr.swapChainExtent.Width), int(vr.swapChainExtent.Height))
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	return true
}

func (g *GBufferCanvas) createGBufferPipelineLayout(renderer Renderer, shader *Shader) (vk.PipelineLayout, bool) {
	vr := renderer.(*Vulkan)
	pipelineLayoutInfo := vk.PipelineLayoutCreateInfo{
		SType:                  vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount:         1, // UniformBufferObject
		PSetLayouts:            &shader.RenderId.descriptorSetLayout,
		PushConstantRangeCount: 0,
		PPushConstantRanges:    nil,
	}
	var pipelineLayout vk.PipelineLayout
	if vk.CreatePipelineLayout(vr.device, &pipelineLayoutInfo, nil, &pipelineLayout) != vk.Success {
		slog.Error("Failed to create G-Buffer pipeline layout")
		return vk.PipelineLayout(vk.NullHandle), false
	} else {
		vr.dbg.add(vk.TypeToUintPtr(pipelineLayout))
	}
	return pipelineLayout, true
}

func (g *GBufferCanvas) createGBufferPipeline(renderer Renderer, shader *Shader, shaderStages []vk.PipelineShaderStageCreateInfo) bool {
	vr := renderer.(*Vulkan)
	bDesc := vertexGetBindingDescription(shader)
	bDescCount := uint32(len(bDesc))
	for i := uint32(1); i < bDescCount; i++ {
		bDesc[i].Stride = uint32(vr.padUniformBufferSize(vk.DeviceSize(bDesc[i].Stride)))
	}
	aDesc := vertexGetAttributeDescription(shader)
	vertexInputInfo := vk.PipelineVertexInputStateCreateInfo{
		SType:                           vk.StructureTypePipelineVertexInputStateCreateInfo,
		VertexBindingDescriptionCount:   bDescCount,
		VertexAttributeDescriptionCount: uint32(len(aDesc)),
		PVertexBindingDescriptions:      &bDesc[0],
		PVertexAttributeDescriptions:    &aDesc[0],
	}
	topology := vk.PrimitiveTopologyTriangleList
	switch shader.DriverData.DrawMode {
	case MeshDrawModePoints:
		topology = vk.PrimitiveTopologyPointList
	case MeshDrawModeLines:
		topology = vk.PrimitiveTopologyLineList
	case MeshDrawModeTriangles:
		topology = vk.PrimitiveTopologyTriangleList
	case MeshDrawModePatches:
		topology = vk.PrimitiveTopologyPatchList
	}
	inputAssembly := vk.PipelineInputAssemblyStateCreateInfo{
		SType:                  vk.StructureTypePipelineInputAssemblyStateCreateInfo,
		PrimitiveRestartEnable: vk.False,
		Topology:               topology,
	}
	viewport := vk.Viewport{
		X:        0.0,
		Y:        0.0,
		Width:    float32(vr.swapChainExtent.Width),
		Height:   float32(vr.swapChainExtent.Height),
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}
	scissor := vk.Rect2D{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: vr.swapChainExtent,
	}
	dynamicStates := []vk.DynamicState{
		vk.DynamicStateViewport,
		vk.DynamicStateScissor,
	}
	dynamicState := vk.PipelineDynamicStateCreateInfo{
		SType:             vk.StructureTypePipelineDynamicStateCreateInfo,
		DynamicStateCount: uint32(len(dynamicStates)),
		PDynamicStates:    &dynamicStates[0],
	}
	viewportState := vk.PipelineViewportStateCreateInfo{
		SType:         vk.StructureTypePipelineViewportStateCreateInfo,
		ViewportCount: 1,
		PViewports:    &viewport,
		ScissorCount:  1,
		PScissors:     &scissor,
	}
	rasterizer := vk.PipelineRasterizationStateCreateInfo{
		SType:                   vk.StructureTypePipelineRasterizationStateCreateInfo,
		DepthClampEnable:        vk.False,
		RasterizerDiscardEnable: vk.False,
		PolygonMode:             vk.PolygonModeFill,
		LineWidth:               1,
		CullMode:                vk.CullModeFlags(vk.CullModeBackBit),
		FrontFace:               vk.FrontFaceClockwise,
		DepthBiasEnable:         vk.False,
		DepthBiasConstantFactor: 0,
		DepthBiasClamp:          0,
		DepthBiasSlopeFactor:    0,
	}
	multisampling := vk.PipelineMultisampleStateCreateInfo{
		SType:                 vk.StructureTypePipelineMultisampleStateCreateInfo,
		SampleShadingEnable:   vk.False,
		RasterizationSamples:  vk.SampleCount1Bit,
		MinSampleShading:      1,
		PSampleMask:           nil,
		AlphaToCoverageEnable: vk.False,
		AlphaToOneEnable:      vk.False,
	}
	colorBlendAttachments := [3]vk.PipelineColorBlendAttachmentState{}
	allChannels := vk.ColorComponentFlags(vk.ColorComponentRBit | vk.ColorComponentGBit | vk.ColorComponentBBit | vk.ColorComponentABit)
	for i := 0; i < len(colorBlendAttachments); i++ {
		colorBlendAttachments[i] = vk.PipelineColorBlendAttachmentState{
			ColorWriteMask:      allChannels,
			BlendEnable:         vk.False, // No blending for G-Buffer pass
			SrcColorBlendFactor: vk.BlendFactorOne,
			DstColorBlendFactor: vk.BlendFactorZero,
			ColorBlendOp:        vk.BlendOpAdd,
			SrcAlphaBlendFactor: vk.BlendFactorOne,
			DstAlphaBlendFactor: vk.BlendFactorZero,
			AlphaBlendOp:        vk.BlendOpAdd,
		}
	}
	colorBlending := vk.PipelineColorBlendStateCreateInfo{
		SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
		LogicOpEnable:   vk.False,
		LogicOp:         vk.LogicOpCopy,
		AttachmentCount: uint32(len(colorBlendAttachments)),
		PAttachments:    &colorBlendAttachments[0],
		BlendConstants:  [4]float32{0.0, 0.0, 0.0, 0.0},
	}
	pipelineLayout, ok := g.createGBufferPipelineLayout(renderer, shader)
	if !ok {
		return false
	}
	shader.RenderId.pipelineLayout = pipelineLayout
	depthStencil := vk.PipelineDepthStencilStateCreateInfo{
		SType:                 vk.StructureTypePipelineDepthStencilStateCreateInfo,
		DepthTestEnable:       vk.True,
		DepthWriteEnable:      vk.True,
		DepthCompareOp:        vk.CompareOpLess,
		DepthBoundsTestEnable: vk.False,
		StencilTestEnable:     vk.False,
	}
	pipelineInfo := vk.GraphicsPipelineCreateInfo{
		SType:               vk.StructureTypeGraphicsPipelineCreateInfo,
		StageCount:          uint32(len(shaderStages)),
		PStages:             &shaderStages[0],
		PVertexInputState:   &vertexInputInfo,
		PInputAssemblyState: &inputAssembly,
		PViewportState:      &viewportState,
		PRasterizationState: &rasterizer,
		PMultisampleState:   &multisampling,
		PColorBlendState:    &colorBlending,
		PDynamicState:       &dynamicState,
		Layout:              shader.RenderId.pipelineLayout,
		RenderPass:          g.gbufferPass.Handle,
		Subpass:             0,
		BasePipelineHandle:  vk.Pipeline(vk.NullHandle),
		PDepthStencilState:  &depthStencil,
	}
	tess := vk.PipelineTessellationStateCreateInfo{}
	if len(shader.CtrlPath) > 0 || len(shader.EvalPath) > 0 {
		tess.SType = vk.StructureTypePipelineTessellationStateCreateInfo
		tess.PatchControlPoints = 3
		pipelineInfo.PTessellationState = &tess
	}
	success := true
	pipelines := [1]vk.Pipeline{}
	if vk.CreateGraphicsPipelines(vr.device, vk.PipelineCache(vk.NullHandle), 1, &pipelineInfo, nil, &pipelines[0]) != vk.Success {
		success = false
		slog.Error("Failed to create G-Buffer graphics pipeline")
	} else {
		vr.dbg.add(vk.TypeToUintPtr(pipelines[0]))
	}
	shader.RenderId.graphicsPipeline = pipelines[0]
	return success
}

func (g *GBufferCanvas) createLightingPipelineLayout(renderer Renderer, shader *Shader) (vk.PipelineLayout, bool) {
	vr := renderer.(*Vulkan)
	pipelineLayoutInfo := vk.PipelineLayoutCreateInfo{
		SType:                  vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount:         1,
		PSetLayouts:            &shader.RenderId.descriptorSetLayout,
		PushConstantRangeCount: 0,
		PPushConstantRanges:    nil,
	}
	var pipelineLayout vk.PipelineLayout
	if vk.CreatePipelineLayout(vr.device, &pipelineLayoutInfo, nil, &pipelineLayout) != vk.Success {
		slog.Error("Failed to create lighting pipeline layout")
		return vk.PipelineLayout(vk.NullHandle), false
	} else {
		vr.dbg.add(vk.TypeToUintPtr(pipelineLayout))
	}
	return pipelineLayout, true
}

func (g *GBufferCanvas) createLightingPipeline(renderer Renderer, shader *Shader, shaderStages []vk.PipelineShaderStageCreateInfo) bool {
	vr := renderer.(*Vulkan)
	vertexInputInfo := vk.PipelineVertexInputStateCreateInfo{
		SType: vk.StructureTypePipelineVertexInputStateCreateInfo,
	}
	inputAssembly := vk.PipelineInputAssemblyStateCreateInfo{
		SType:                  vk.StructureTypePipelineInputAssemblyStateCreateInfo,
		PrimitiveRestartEnable: vk.False,
		Topology:               vk.PrimitiveTopologyTriangleList,
	}
	viewport := vk.Viewport{
		X:        0,
		Y:        0,
		Width:    float32(vr.swapChainExtent.Width),
		Height:   float32(vr.swapChainExtent.Height),
		MinDepth: 0,
		MaxDepth: 1,
	}
	scissor := vk.Rect2D{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: vr.swapChainExtent, // Or use swapchain/render target extent
	}
	dynamicStates := []vk.DynamicState{
		vk.DynamicStateViewport,
		vk.DynamicStateScissor,
	}
	dynamicState := vk.PipelineDynamicStateCreateInfo{
		SType:             vk.StructureTypePipelineDynamicStateCreateInfo,
		DynamicStateCount: uint32(len(dynamicStates)),
		PDynamicStates:    &dynamicStates[0],
	}
	viewportState := vk.PipelineViewportStateCreateInfo{
		SType:         vk.StructureTypePipelineViewportStateCreateInfo,
		ViewportCount: 1,
		PViewports:    &viewport,
		ScissorCount:  1,
		PScissors:     &scissor,
	}
	rasterizer := vk.PipelineRasterizationStateCreateInfo{
		SType:                   vk.StructureTypePipelineRasterizationStateCreateInfo,
		DepthClampEnable:        vk.False,
		RasterizerDiscardEnable: vk.False,
		PolygonMode:             vk.PolygonModeFill,
		LineWidth:               1,
		CullMode:                vk.CullModeFlags(vk.CullModeNone),
		FrontFace:               vk.FrontFaceClockwise,
		DepthBiasEnable:         vk.False,
		DepthBiasConstantFactor: 0,
		DepthBiasClamp:          0,
		DepthBiasSlopeFactor:    0,
	}
	multisampling := vk.PipelineMultisampleStateCreateInfo{
		SType:                 vk.StructureTypePipelineMultisampleStateCreateInfo,
		SampleShadingEnable:   vk.False,
		RasterizationSamples:  vk.SampleCount1Bit,
		MinSampleShading:      1,
		PSampleMask:           nil,
		AlphaToCoverageEnable: vk.False,
		AlphaToOneEnable:      vk.False,
	}
	colorBlendAttachment := [1]vk.PipelineColorBlendAttachmentState{}
	allChannels := vk.ColorComponentFlags(vk.ColorComponentRBit | vk.ColorComponentGBit | vk.ColorComponentBBit | vk.ColorComponentABit)
	colorBlendAttachment[0] = vk.PipelineColorBlendAttachmentState{
		ColorWriteMask:      allChannels,
		BlendEnable:         vk.False,
		SrcColorBlendFactor: vk.BlendFactorOne,
		DstColorBlendFactor: vk.BlendFactorZero,
		ColorBlendOp:        vk.BlendOpAdd,
		SrcAlphaBlendFactor: vk.BlendFactorOne,
		DstAlphaBlendFactor: vk.BlendFactorZero,
		AlphaBlendOp:        vk.BlendOpAdd,
	}
	colorBlending := vk.PipelineColorBlendStateCreateInfo{
		SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
		LogicOpEnable:   vk.False,
		LogicOp:         vk.LogicOpCopy,
		AttachmentCount: 1,
		PAttachments:    &colorBlendAttachment[0],
		BlendConstants:  [4]float32{0.0, 0.0, 0.0, 0.0},
	}
	pipelineLayout, ok := g.createLightingPipelineLayout(renderer, shader)
	if !ok {
		return false
	}
	shader.RenderId.pipelineLayout = pipelineLayout
	depthStencil := vk.PipelineDepthStencilStateCreateInfo{
		SType:                 vk.StructureTypePipelineDepthStencilStateCreateInfo,
		DepthTestEnable:       vk.False,
		DepthWriteEnable:      vk.False,
		DepthCompareOp:        vk.CompareOpLessOrEqual,
		DepthBoundsTestEnable: vk.False,
		StencilTestEnable:     vk.False,
	}
	pipelineInfo := vk.GraphicsPipelineCreateInfo{
		SType:               vk.StructureTypeGraphicsPipelineCreateInfo,
		StageCount:          uint32(len(shaderStages)),
		PStages:             &shaderStages[0],
		PVertexInputState:   &vertexInputInfo,
		PInputAssemblyState: &inputAssembly,
		PViewportState:      &viewportState,
		PRasterizationState: &rasterizer,
		PMultisampleState:   &multisampling,
		PColorBlendState:    &colorBlending,
		PDynamicState:       &dynamicState,
		Layout:              shader.RenderId.pipelineLayout,
		RenderPass:          g.lightingPass.Handle,
		Subpass:             0,
		BasePipelineHandle:  vk.Pipeline(vk.NullHandle),
		PDepthStencilState:  &depthStencil,
	}
	tess := vk.PipelineTessellationStateCreateInfo{}
	if len(shader.CtrlPath) > 0 || len(shader.EvalPath) > 0 {
		tess.SType = vk.StructureTypePipelineTessellationStateCreateInfo
		tess.PatchControlPoints = 3
		pipelineInfo.PTessellationState = &tess
	}
	success := true
	pipelines := [1]vk.Pipeline{}
	if vk.CreateGraphicsPipelines(vr.device, vk.PipelineCache(vk.NullHandle), 1, &pipelineInfo, nil, &pipelines[0]) != vk.Success {
		success = false
		slog.Error("Failed to create lighting graphics pipeline")
	} else {
		vr.dbg.add(vk.TypeToUintPtr(pipelines[0]))
	}
	shader.RenderId.graphicsPipeline = pipelines[0]
	return success
}
