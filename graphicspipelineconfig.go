package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type IGraphicsPipelineConfig interface {
	VKGraphicsPipelineCreateInfo(screenExtents vk.Extent2D) (vk.GraphicsPipelineCreateInfo, error)
}

type GraphicsPipelineConfig struct {
	Device               *Device
	ShaderStages         []vk.PipelineShaderStageCreateInfo
	VertexSource         VertexSource
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
	BlendAttachments []vk.PipelineColorBlendAttachmentState

	// DepthTestEnable defaults to true
	DepthTestEnable bool

	// DepthWriteEnable defaults to true
	DepthWriteEnable bool
}

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

func (g *GraphicsPipelineConfig) AddBlendAttachment(ba vk.PipelineColorBlendAttachmentState) {
	if g.BlendAttachments == nil {
		g.BlendAttachments = make([]vk.PipelineColorBlendAttachmentState, 0)
	}
	g.BlendAttachments = append(g.BlendAttachments, ba)
}

func (g *GraphicsPipelineConfig) SetCullMode(mode vk.CullModeFlagBits) *GraphicsPipelineConfig {
	g.CullMode = mode
	return g
}

func (g *GraphicsPipelineConfig) SetDynamicState(states ...vk.DynamicState) *GraphicsPipelineConfig {
	g.DynamicState = states
	return g
}

func (g *GraphicsPipelineConfig) AddShaderStageFromFile(file, entryPoint string, stageType vk.ShaderStageFlagBits) error {
	shader, err := g.Device.LoadShaderModuleFromFile(file)
	if err != nil {
		return nil
	}
	if g.ShaderStages == nil {
		g.ShaderStages = make([]vk.PipelineShaderStageCreateInfo, 0)
	}
	g.ShaderStages = append(g.ShaderStages, shader.VKPipelineShaderStageCreateInfo(stageType, entryPoint))
	return nil
}

func (g *GraphicsPipelineConfig) SetPipelineLayout(layout *PipelineLayout) *GraphicsPipelineConfig {
	g.PipelineLayout = layout
	return g
}

func (g *GraphicsPipelineConfig) SetShaderStages(shaderStages []vk.PipelineShaderStageCreateInfo) *GraphicsPipelineConfig {
	g.ShaderStages = shaderStages
	return g
}

//FIXME should accomodate a list
func (g *GraphicsPipelineConfig) SetVertexDescriptor(v VertexSource) *GraphicsPipelineConfig {
	g.VertexSource = v
	return g
}

func (g *GraphicsPipelineConfig) AddDescriptorSetLayout(d *DescriptorSetLayout) *GraphicsPipelineConfig {
	if g.DescriptorSetLayouts == nil {
		g.DescriptorSetLayouts = make([]*DescriptorSetLayout, 0)
	}
	g.DescriptorSetLayouts = append(g.DescriptorSetLayouts, d)
	return g
}

func (g *GraphicsPipelineConfig) VKGraphicsPipelineCreateInfo(extent vk.Extent2D) (vk.GraphicsPipelineCreateInfo, error) {

	var vertexInputState = vk.PipelineVertexInputStateCreateInfo{}
	vertexInputState.SType = vk.StructureTypePipelineVertexInputStateCreateInfo

	if g.VertexSource == nil {
		vertexInputState.VertexBindingDescriptionCount = 0
		vertexInputState.PVertexBindingDescriptions = nil // Optional
		vertexInputState.VertexAttributeDescriptionCount = 0
		vertexInputState.PVertexAttributeDescriptions = nil // Optional

	} else {
		vertexInputState.VertexBindingDescriptionCount = 1
		vertexInputState.PVertexBindingDescriptions = []vk.VertexInputBindingDescription{g.VertexSource.GetBindingDesciption()}
		attrs := g.VertexSource.GetAttributeDescriptions()
		vertexInputState.VertexAttributeDescriptionCount = uint32(len(attrs))
		vertexInputState.PVertexAttributeDescriptions = attrs
	}

	var inputAssemblyState = vk.PipelineInputAssemblyStateCreateInfo{}
	inputAssemblyState.SType = vk.StructureTypePipelineInputAssemblyStateCreateInfo
	inputAssemblyState.Topology = g.PrimitiveTopology
	inputAssemblyState.PrimitiveRestartEnable = g.PrimitiveRestartEnable

	var viewport = vk.Viewport{}
	viewport.X = 0.0
	viewport.Y = 0.0
	viewport.Width = float32(extent.Width)
	viewport.Height = float32(extent.Height)
	viewport.MinDepth = 0.0
	viewport.MaxDepth = 1.0

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
	rasterState.PolygonMode = vk.PolygonModeFill
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
