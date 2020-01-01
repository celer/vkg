package imgui

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/celer/vkg"
	imgui "github.com/inkyblackness/imgui-go"
	vk "github.com/vulkan-go/vulkan"
	lin "github.com/xlab/linmath"
)

type Renderer struct {
	io  imgui.IO
	app *vkg.GraphicsApp

	ubo UBO

	uboBuffer *vkg.BufferResource

	//vertex and index buffers allocated per a frame
	transientBuffers []*vkg.BufferResource

	descriptorSet  *vkg.DescriptorSet
	pipelineLayout *vkg.PipelineLayout

	descriptorPool      *vkg.DescriptorPool
	descriptorSetLayout *vkg.DescriptorSetLayout

	fontBuffer  *vkg.ImageResource
	fontView    vk.ImageView
	fontSampler vk.Sampler

	maxVertexes int
	maxIndexes  int
}

type UBO struct {
	Proj lin.Mat4x4
}

func (u *UBO) Bytes() []byte {
	size := int(unsafe.Sizeof(float32(1))) * 4 * 4
	return vkg.ToBytes(unsafe.Pointer(&u.Proj[0]), size)

}

func NewRenderer(io imgui.IO, app *vkg.GraphicsApp, maxVertexes, maxIndexes int) (*Renderer, error) {
	return &Renderer{io: io, app: app, maxIndexes: maxIndexes, maxVertexes: maxVertexes}, nil
}

func (r *Renderer) GetBindingDescription() vk.VertexInputBindingDescription {
	vertexSize, _, _, _ := imgui.VertexBufferLayout()

	var bindingDescription = vk.VertexInputBindingDescription{}
	bindingDescription.Binding = 0
	bindingDescription.Stride = uint32(vertexSize)
	bindingDescription.InputRate = vk.VertexInputRateVertex

	return bindingDescription
}

func (r *Renderer) GetAttributeDescriptions() []vk.VertexInputAttributeDescription {
	_, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()

	attr := make([]vk.VertexInputAttributeDescription, 3)

	attr[0].Binding = 0
	attr[0].Location = 0
	attr[0].Format = vk.FormatR32g32Sfloat
	attr[0].Offset = uint32(vertexOffsetPos)

	attr[1].Binding = 0
	attr[1].Location = 1
	attr[1].Format = vk.FormatR32g32Sfloat
	attr[1].Offset = uint32(vertexOffsetUv)

	attr[2].Binding = 0
	attr[2].Location = 2
	attr[2].Format = vk.FormatR8g8b8a8Uint
	attr[2].Offset = uint32(vertexOffsetCol)

	return attr

}

func (r *Renderer) freeTransientBuffers() {
	for len(r.transientBuffers) > 2 {
		var a, b *vkg.BufferResource

		a, r.transientBuffers = r.transientBuffers[0], r.transientBuffers[1:]
		b, r.transientBuffers = r.transientBuffers[0], r.transientBuffers[1:]

		a.Free()
		b.Free()
	}

}

type Vertex struct {
	Pos [2]float32
	Uv  [2]float32
	Col [4]uint8
}

func castVertex(ptr unsafe.Pointer, size int) []Vertex {
	const m = 0x7ffffff
	return (*[m]Vertex)(ptr)[:size/int(unsafe.Sizeof(Vertex{}))]
}

func castUint16(ptr unsafe.Pointer, size int) []uint16 {
	const m = 0x7ffffff
	return (*[m]uint16)(ptr)[:size/int(unsafe.Sizeof(uint16(1)))]
}

func castBytes(ptr unsafe.Pointer, size int) []byte {
	const m = 0x7ffffff
	return (*[m]byte)(ptr)[:size]
}

func (r *Renderer) setupUBO() {

	extent := r.app.GetScreenExtent()

	var ubo lin.Mat4x4 = [4]lin.Vec4{
		lin.Vec4{2.0, 0.0, 0.0, 0.0},
		lin.Vec4{0.0, 2.0, 0.0, 0.0},
		lin.Vec4{0.0, 0.0, 1.0, 0.0},
		lin.Vec4{-1, -1, 0.0, 1.0},
	}
	ubo[0][0] /= float32(extent.Width)
	ubo[1][1] /= float32(extent.Height)

	r.ubo.Proj = ubo

	copy(r.uboBuffer.Bytes(), r.ubo.Bytes())

}

func (r *Renderer) Render(renderpass vk.RenderPass, framebuffer vk.Framebuffer, drawData imgui.DrawData) ([]vk.CommandBuffer, error) {

	extent := r.app.GetScreenExtent()

	//FIXME investigate framebuffer vs screen size
	drawData.ScaleClipRects(imgui.Vec2{
		1.0, 1.0,
	})

	// Setup and push UBO
	r.freeTransientBuffers()

	indexSize := imgui.IndexBufferLayout()

	indexType := vk.IndexTypeUint16
	if indexSize == 4 {
		indexType = vk.IndexTypeUint32
	}

	buffers := make([]vk.CommandBuffer, 0)

	//fmt.Printf("drawData.CommandList()\n")
	for _, list := range drawData.CommandLists() {

		cmdb, err := r.app.GraphicsCommandPool.AllocateBuffer(vk.CommandBufferLevelSecondary)
		if err != nil {
			return nil, err
		}

		vpool := r.app.ResourceManager.BufferPool("imgui-vdata")

		var offset int

		vertexData, vertexDataSize := list.VertexBuffer()
		indexData, indexDataSize := list.IndexBuffer()

		vbuff, err := vpool.AllocateBuffer(uint64(vertexDataSize), vk.BufferUsageVertexBufferBit)
		if err != nil {
			return nil, fmt.Errorf("unable to allocate vertex buffer: %w", err)
		}
		r.transientBuffers = append(r.transientBuffers, vbuff)

		ibuff, err := vpool.AllocateBuffer(uint64(indexDataSize), vk.BufferUsageIndexBufferBit)
		if err != nil {
			return nil, fmt.Errorf("unable to allocate index buffer: %w", err)
		}
		r.transientBuffers = append(r.transientBuffers, ibuff)

		r.setupUBO()

		copy(vbuff.Bytes(), castBytes(vertexData, vertexDataSize))
		copy(ibuff.Bytes(), castBytes(indexData, indexDataSize))

		err = r.app.Device.FlushMappedRanges(vbuff, ibuff)
		if err != nil {
			return nil, err
		}

		cmdb.BeginContinueRenderPass(renderpass, framebuffer)

		viewport := vk.Viewport{
			Width:    float32(extent.Width),
			Height:   float32(extent.Height),
			MinDepth: 0.0,
			MaxDepth: 1.0,
		}
		vk.CmdSetViewport(cmdb.VK(), 0, 1, []vk.Viewport{viewport})

		vk.CmdBindPipeline(cmdb.VK(), vk.PipelineBindPointGraphics, r.app.GraphicsPipelines["imgui"])

		vk.CmdBindDescriptorSets(cmdb.VK(), vk.PipelineBindPointGraphics,
			r.pipelineLayout.VKPipelineLayout, 0, 1,
			[]vk.DescriptorSet{r.descriptorSet.VKDescriptorSet}, 0, nil)

		vk.CmdBindVertexBuffers(cmdb.VK(), 0, 1, []vk.Buffer{vbuff.VKBuffer}, []vk.DeviceSize{0})
		vk.CmdBindIndexBuffer(cmdb.VK(), ibuff.VKBuffer, vk.DeviceSize(0), indexType)

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {
				clipRect := cmd.ClipRect()

				scissor := vk.Rect2D{}
				scissor.Extent.Width = uint32(clipRect.Z - clipRect.X)
				scissor.Extent.Height = uint32(clipRect.W - clipRect.Y)

				scissor.Offset.X = int32(clipRect.X)
				scissor.Offset.Y = int32(clipRect.Y)

				vk.CmdSetScissor(cmdb.VK(), 0, 1, []vk.Rect2D{scissor})

				vk.CmdDrawIndexed(cmdb.VK(), uint32(cmd.ElementCount()), 1, uint32(offset), 0, 0)
			}
			offset += cmd.ElementCount()
		}
		cmdb.End()
		buffers = append(buffers, cmdb.VK())
	}

	return buffers, nil
}

func (r *Renderer) Init() error {
	var err error

	if !r.app.ResourceManager.HasStagingPool() {
		_, err = r.app.ResourceManager.AllocateStagingPool(32 * 1024 * 1024)
		if err != nil {
			return err
		}
	}

	err = r.createVertexAndIndexBuffers()
	if err != nil {
		return err
	}
	err = r.createFontTexture()
	if err != nil {
		return err
	}
	err = r.createDescriptorSet()
	if err != nil {
		return err
	}
	err = r.createGraphicsPipeline()
	if err != nil {
		return err
	}
	r.transientBuffers = make([]*vkg.BufferResource, 0)
	return nil
}

func (r *Renderer) Destroy() {
	r.uboBuffer.Destroy()
	r.fontBuffer.Destroy()
	r.freeTransientBuffers()

	r.app.Device.DestroyAny(r.fontView)
	r.app.Device.DestroyAny(r.fontSampler)

	r.descriptorPool.Destroy()
	//r.pipelineLayout.Destroy()

	r.descriptorSetLayout.Destroy()

	r.app.ResourceManager.BufferPool("imgui-vdata").Memory.Unmap()
	r.app.ResourceManager.BufferPool("imgui-vdata").Destroy()

	r.app.ResourceManager.ImagePool("imgui-fonts").Destroy()
}

func (r *Renderer) createDescriptorSet() error {

	dpool := r.app.Device.NewDescriptorPool()
	dpool.AddPoolSize(vk.DescriptorTypeUniformBuffer, 1)
	dpool.AddPoolSize(vk.DescriptorTypeCombinedImageSampler, 1)
	_, err := r.app.Device.CreateDescriptorPool(dpool, 1)
	if err != nil {
		return err
	}

	r.descriptorPool = dpool

	dsl := r.app.Device.NewDescriptorSetLayout()
	dsl.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vk.DescriptorTypeUniformBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageVertexBit),
	})
	dsl.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:            1,
		DescriptorCount:    1,
		PImmutableSamplers: nil,
		StageFlags:         vk.ShaderStageFlags(vk.ShaderStageFragmentBit),
		DescriptorType:     vk.DescriptorTypeCombinedImageSampler,
	})

	_, err = r.app.Device.CreateDescriptorSetLayout(dsl)
	if err != nil {
		return err
	}

	r.descriptorSet, err = dpool.Allocate(dsl)
	if err != nil {
		return err
	}
	r.descriptorSet.AddBuffer(0, vk.DescriptorTypeUniformBuffer, &r.uboBuffer.Buffer, 0)
	r.descriptorSet.AddCombinedImageSampler(1, vk.ImageLayoutShaderReadOnlyOptimal, r.fontView, r.fontSampler)
	r.descriptorSet.Write()

	r.descriptorSetLayout = dsl

	r.pipelineLayout, err = r.app.Device.CreatePipelineLayout(dsl)
	if err != nil {
		return err
	}
	return nil
}

func (r *Renderer) createGraphicsPipeline() error {

	gc := r.app.CreateGraphicsPipelineConfig()

	gc.AddVertexDescriptor(r)
	gc.AddBlendAttachment(vk.PipelineColorBlendAttachmentState{
		ColorWriteMask:      vk.ColorComponentFlags(vk.ColorComponentRBit | vk.ColorComponentGBit | vk.ColorComponentBBit | vk.ColorComponentABit),
		SrcColorBlendFactor: vk.BlendFactorSrcAlpha,
		DstColorBlendFactor: vk.BlendFactorOneMinusSrcAlpha,
		ColorBlendOp:        vk.BlendOpAdd,
		SrcAlphaBlendFactor: vk.BlendFactorOne,
		DstAlphaBlendFactor: vk.BlendFactorZero,
		AlphaBlendOp:        vk.BlendOpAdd,
		BlendEnable:         vk.True,
	})
	err := gc.AddShaderStageFromFile("shaders/vert.spv", "main", vk.ShaderStageVertexBit)
	if err != nil {
		return err
	}
	err = gc.AddShaderStageFromFile("shaders/frag.spv", "main", vk.ShaderStageFragmentBit)
	if err != nil {
		return err
	}
	gc.SetDynamicState(vk.DynamicStateViewport, vk.DynamicStateScissor)
	gc.SetCullMode(vk.CullModeNone)
	gc.DepthWriteEnable = false
	gc.DepthTestEnable = false
	gc.SetPipelineLayout(r.pipelineLayout)

	r.app.AddGraphicsPipelineConfig("imgui", gc)

	return nil

}

func (r *Renderer) createVertexAndIndexBuffers() error {

	vertexSize, _, _, _ := imgui.VertexBufferLayout()
	indexSize := imgui.IndexBufferLayout()
	uboSize := int(unsafe.Sizeof(UBO{}))

	poolSize := vertexSize*r.maxVertexes + indexSize*r.maxIndexes + (1024 * 1024 * 10) + uboSize

	vpool, err := r.app.ResourceManager.AllocateHostVertexAndIndexBufferPool("imgui-vdata", uint64(poolSize))
	if err != nil {
		return fmt.Errorf("unable to allocate vertex pool: %w", err)
	}

	r.uboBuffer, err = vpool.AllocateBuffer(uint64(uboSize), vk.BufferUsageUniformBufferBit)
	if err != nil {
		return fmt.Errorf("unable to allocate buffer for ubo: %w", err)
	}

	vpool.Memory.Map()

	return nil

}

func (r *Renderer) createFontTexture() error {

	fontTexture := r.io.Fonts().TextureDataRGBA32()

	tpool, err := r.app.ResourceManager.AllocateDeviceTexturePool("imgui-fonts", 12*1024*1024)
	if err != nil {
		return err
	}

	cb, err := r.app.GraphicsCommandPool.AllocateBuffer(vk.CommandBufferLevelPrimary)
	if err != nil {
		return err
	}

	fontImg := image.NewRGBA(image.Rectangle{Max: image.Point{X: fontTexture.Width, Y: fontTexture.Height}})

	const m = 0x7FFFFFFF

	fontImg.Pix = (*[m]byte)(fontTexture.Pixels)[:fontTexture.Width*fontTexture.Height*4]

	fontBuffer, err := tpool.StageTextureFromImage(fontImg, cb, r.app.GraphicsQueue)
	if err != nil {
		return err
	}

	r.app.GraphicsCommandPool.FreeBuffer(cb)

	imageView, err := fontBuffer.CreateImageViewWithAspectMask(vk.ImageAspectFlags(vk.ImageAspectColorBit))
	if err != nil {
		return err
	}

	var sampler vk.Sampler
	vk.CreateSampler(r.app.Device.VKDevice, &vk.SamplerCreateInfo{
		SType:                   vk.StructureTypeSamplerCreateInfo,
		MagFilter:               vk.FilterLinear,
		MinFilter:               vk.FilterLinear,
		MipmapMode:              vk.SamplerMipmapModeLinear,
		AddressModeU:            vk.SamplerAddressModeRepeat,
		AddressModeV:            vk.SamplerAddressModeRepeat,
		AddressModeW:            vk.SamplerAddressModeRepeat,
		AnisotropyEnable:        vk.False,
		MaxAnisotropy:           1,
		CompareOp:               vk.CompareOpAlways,
		BorderColor:             vk.BorderColorIntOpaqueBlack,
		UnnormalizedCoordinates: vk.False,
	}, nil, &sampler)

	r.fontSampler = sampler
	r.fontView = imageView.VKImageView
	r.fontBuffer = fontBuffer

	return nil
}
