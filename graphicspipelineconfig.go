package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

// GraphicsPipelineConfig is a utility object to ease construction of graphics pipelines
type GraphicsPipelineConfig struct {
	Device               *Device
	ShaderStages         []vk.PipelineShaderStageCreateInfo
	VertexSource         VertexSourcer
	DescriptorSetLayouts []*DescriptorSetLayout

	PipelineLayout *PipelineLayout

	// Called as the last step in config generation to allow for
	// additional configuration
	Configure func(config vk.GraphicsPipelineCreateInfo)

	// PrimativeTopology see https://www.khronos.org/registry/vulkan/specs/1.1-extensions/man/html/VkPrimitiveTopology.html
	// defaults to VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST
	PrimitiveTopology vk.PrimitiveTopology

	// PrimativeRestartEnable see https://www.khronos.org/registry/vulkan/specs/1.1-extensions/man/html/VkPipelineInputAssemblyStateCreateInfo.html
	// defaults to False
	PrimitiveRestartEnable vk.Bool32

	// PolygonMode see https://www.khronos.org/registry/vulkan/specs/1.1-extensions/man/html/VkPolygonMode.html
	// defaults to VK_POLYGON_MODE_FILL
	PolygonMode vk.PolygonMode

	// LineWidth of rasterized lines see https://www.khronos.org/registry/vulkan/specs/1.1-extensions/man/html/VkPipelineRasterizationStateCreateInfo.html
	// defaults to 1.0
	LineWidth float32

	// CullModes specifies which triangles will be culled. See https://www.khronos.org/registry/vulkan/specs/1.1/html/vkspec.html#VkCullModeFlagBits
	// Defaults to vk.CullModeBackBit
	CullMode vk.CullModeFlagBits

	// DynamicState specifies which part of the pipeline might be modified by the command buffer see
	// https://www.khronos.org/registry/vulkan/specs/1.1/html/vkspec.html#VkDynamicState
	// defaults to none
	DynamicState []vk.DynamicState

	// FrontFace specifies how the front face of a triangle is determined, see https://www.khronos.org/registry/vulkan/specs/1.1/html/vkspec.html#VkFrontFace
	// defaults to vk.FrontFaceCounterClockwise
	FrontFace vk.FrontFace

	// Add a pipeline color blend attachment state, by default it's assumed that all colors are set to no blending
	// see https://www.khronos.org/registry/vulkan/specs/1.1-extensions/man/html/VkPipelineColorBlendAttachmentState.html
	BlendAttachments []vk.PipelineColorBlendAttachmentState

	// DepthTestEnable defaults to true
	DepthTestEnable bool

	// DepthWriteEnable defaults to true
	DepthWriteEnable bool

	VertexInputBindingDescriptions   []vk.VertexInputBindingDescription
	VertexInputAttributeDescriptions []vk.VertexInputAttributeDescription

	toDestroy []IDestructable

	Viewport *vk.Viewport
}

// CreateGraphicsPipelineConfig creates a new config object
func (d *Device) CreateGraphicsPipelineConfig() *GraphicsPipelineConfig {
	return &GraphicsPipelineConfig{
		Device:                 d,
		PrimitiveTopology:      vk.PrimitiveTopologyTriangleList,
		PrimitiveRestartEnable: vk.False,
		PolygonMode:            vk.PolygonModeFill,
		LineWidth:              1.0,
		CullMode:               vk.CullModeBackBit,
		FrontFace:              vk.FrontFaceCounterClockwise,
		DepthTestEnable:        true,
		DepthWriteEnable:       true,
	}
}

func (g *GraphicsPipelineConfig) manageDestroy(d IDestructable) {
	if g.toDestroy == nil {
		g.toDestroy = make([]IDestructable, 0)
	}
	g.toDestroy = append(g.toDestroy, d)
}

func (g *GraphicsPipelineConfig) Destroy() {
	for _, d := range g.toDestroy {
		d.Destroy()
	}

}

// AddBlendAttachment adds a new blend attachment
func (g *GraphicsPipelineConfig) AddBlendAttachment(ba vk.PipelineColorBlendAttachmentState) {
	if g.BlendAttachments == nil {
		g.BlendAttachments = make([]vk.PipelineColorBlendAttachmentState, 0)
	}
	g.BlendAttachments = append(g.BlendAttachments, ba)
}

// SetCullMode sets the cull mode
func (g *GraphicsPipelineConfig) SetCullMode(mode vk.CullModeFlagBits) *GraphicsPipelineConfig {
	g.CullMode = mode
	return g
}

// SetDynamicState specifies which part of the pipeline may be changed with command buffer commands
func (g *GraphicsPipelineConfig) SetDynamicState(states ...vk.DynamicState) *GraphicsPipelineConfig {
	g.DynamicState = states
	return g
}

// AddShaderStageFromFile adds a shader from a specified file
func (g *GraphicsPipelineConfig) AddShaderStageFromFile(file, entryPoint string, stageType vk.ShaderStageFlagBits) error {
	shader, err := g.Device.LoadShaderModuleFromFile(file)
	if err != nil {
		return err
	}
	if g.ShaderStages == nil {
		g.ShaderStages = make([]vk.PipelineShaderStageCreateInfo, 0)
	}
	g.ShaderStages = append(g.ShaderStages, shader.VKPipelineShaderStageCreateInfo(stageType, entryPoint))

	g.manageDestroy(shader)

	return nil
}

// SetPipelineLayout sets the pipeline layout
func (g *GraphicsPipelineConfig) SetPipelineLayout(layout *PipelineLayout) *GraphicsPipelineConfig {
	g.PipelineLayout = layout
	return g
}

// SetShaderStages sets the shader stages directly
func (g *GraphicsPipelineConfig) SetShaderStages(shaderStages []vk.PipelineShaderStageCreateInfo) *GraphicsPipelineConfig {
	g.ShaderStages = shaderStages
	return g
}

// AddVertexDescriptor adds vertex descriptors based off the specified interface
func (g *GraphicsPipelineConfig) AddVertexDescriptor(v VertexDescriptor) *GraphicsPipelineConfig {
	if g.VertexInputBindingDescriptions == nil {
		g.VertexInputBindingDescriptions = make([]vk.VertexInputBindingDescription, 0)
	}
	if g.VertexInputAttributeDescriptions == nil {
		g.VertexInputAttributeDescriptions = make([]vk.VertexInputAttributeDescription, 0)
	}

	g.VertexInputBindingDescriptions = append(g.VertexInputBindingDescriptions, v.GetBindingDescription())
	g.VertexInputAttributeDescriptions = append(g.VertexInputAttributeDescriptions, v.GetAttributeDescriptions()...)

	return g
}

// AddDescriptorSetLayout adds a specific DescriptorSetLayout
func (g *GraphicsPipelineConfig) AddDescriptorSetLayout(d *DescriptorSetLayout) *GraphicsPipelineConfig {
	if g.DescriptorSetLayouts == nil {
		g.DescriptorSetLayouts = make([]*DescriptorSetLayout, 0)
	}
	g.DescriptorSetLayouts = append(g.DescriptorSetLayouts, d)
	return g
}

// VKGraphicsPipelineCreateInfo uses the provided config information to create a vulkank vk.GraphicsPipelineCreateInfo structure
func (g *GraphicsPipelineConfig) VKGraphicsPipelineCreateInfo(extent vk.Extent2D) (vk.GraphicsPipelineCreateInfo, error) {

	var vertexInputState = vk.PipelineVertexInputStateCreateInfo{}
	vertexInputState.SType = vk.StructureTypePipelineVertexInputStateCreateInfo

	vertexInputState.VertexBindingDescriptionCount = uint32(len(g.VertexInputBindingDescriptions))
	vertexInputState.PVertexBindingDescriptions = g.VertexInputBindingDescriptions
	vertexInputState.VertexAttributeDescriptionCount = uint32(len(g.VertexInputAttributeDescriptions))
	vertexInputState.PVertexAttributeDescriptions = g.VertexInputAttributeDescriptions

	var inputAssemblyState = vk.PipelineInputAssemblyStateCreateInfo{}
	inputAssemblyState.SType = vk.StructureTypePipelineInputAssemblyStateCreateInfo
	inputAssemblyState.Topology = g.PrimitiveTopology
	inputAssemblyState.PrimitiveRestartEnable = g.PrimitiveRestartEnable

	var viewport = vk.Viewport{}
	if g.Viewport == nil {
		viewport.X = 0.0
		viewport.Y = 0.0
		viewport.Width = float32(extent.Width)
		viewport.Height = float32(extent.Height)
		viewport.MinDepth = 0.0
		viewport.MaxDepth = 1.0
	} else {
		viewport = *g.Viewport
	}

	var scissor = vk.Rect2D{}
	scissor.Offset = vk.Offset2D{X: 0, Y: 0}
	scissor.Extent = extent

	var viewportState = vk.PipelineViewportStateCreateInfo{}
	viewportState.SType = vk.StructureTypePipelineViewportStateCreateInfo
	viewportState.ViewportCount = 1
	viewportState.PViewports = []vk.Viewport{viewport}
	viewportState.ScissorCount = 1
	viewportState.PScissors = []vk.Rect2D{scissor}

	var rasterState = vk.PipelineRasterizationStateCreateInfo{}
	rasterState.SType = vk.StructureTypePipelineRasterizationStateCreateInfo
	rasterState.DepthClampEnable = vk.False
	rasterState.RasterizerDiscardEnable = vk.False
	rasterState.PolygonMode = g.PolygonMode
	rasterState.LineWidth = g.LineWidth
	rasterState.CullMode = vk.CullModeFlags(g.CullMode)

	rasterState.FrontFace = g.FrontFace
	rasterState.DepthBiasEnable = vk.False

	var multisampleState = vk.PipelineMultisampleStateCreateInfo{}
	multisampleState.SType = vk.StructureTypePipelineMultisampleStateCreateInfo
	multisampleState.SampleShadingEnable = vk.False
	multisampleState.RasterizationSamples = vk.SampleCount1Bit

	blendAttachments := []vk.PipelineColorBlendAttachmentState{}
	if g.BlendAttachments == nil {
		blendAttachments = []vk.PipelineColorBlendAttachmentState{{
			ColorWriteMask: vk.ColorComponentFlags(vk.ColorComponentRBit | vk.ColorComponentGBit | vk.ColorComponentBBit | vk.ColorComponentABit),
			BlendEnable:    vk.False,
		}}
	} else {
		blendAttachments = g.BlendAttachments
	}

	var colorBlendState = vk.PipelineColorBlendStateCreateInfo{
		SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
		AttachmentCount: uint32(len(blendAttachments)),
		PAttachments:    blendAttachments,
	}

	dynamicState := vk.PipelineDynamicStateCreateInfo{
		SType:             vk.StructureTypePipelineDynamicStateCreateInfo,
		PDynamicStates:    g.DynamicState,
		DynamicStateCount: uint32(len(g.DynamicState)),
	}

	dte := vk.True
	if !g.DepthTestEnable {
		dte = vk.False
	}

	dwe := vk.True
	if !g.DepthWriteEnable {
		dwe = vk.False
	}

	var depthStencil = vk.PipelineDepthStencilStateCreateInfo{
		SType:                 vk.StructureTypePipelineDepthStencilStateCreateInfo,
		DepthTestEnable:       vk.Bool32(dte),
		DepthWriteEnable:      vk.Bool32(dwe),
		DepthCompareOp:        vk.CompareOpLess,
		DepthBoundsTestEnable: vk.False,
		MinDepthBounds:        0.0,
		MaxDepthBounds:        1.0,
		StencilTestEnable:     vk.False,
	}

	var pipelineLayout vk.PipelineLayout
	if g.PipelineLayout != nil {
		pipelineLayout = g.PipelineLayout.VKPipelineLayout
	}

	pipelineCreateInfos := vk.GraphicsPipelineCreateInfo{
		SType:               vk.StructureTypeGraphicsPipelineCreateInfo,
		StageCount:          uint32(len(g.ShaderStages)),
		PStages:             g.ShaderStages,
		PVertexInputState:   &vertexInputState,
		PInputAssemblyState: &inputAssemblyState,
		PDepthStencilState:  &depthStencil,
		PViewportState:      &viewportState,
		PRasterizationState: &rasterState,
		PMultisampleState:   &multisampleState,
		PColorBlendState:    &colorBlendState,
		PDynamicState:       &dynamicState,
		Layout:              pipelineLayout,
		Subpass:             0,
	}

	if g.Configure != nil {
		g.Configure(pipelineCreateInfos)
	}

	return pipelineCreateInfos, nil

}
