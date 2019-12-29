package main

import (
	"runtime"
	"unsafe"

	vkg "github.com/celer/vkg"
	"github.com/vulkan-go/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
	lin "github.com/xlab/linmath"
)

var Width = 800
var Height = 600

type Vertex struct {
	Pos      lin.Vec3
	Color    lin.Vec3
	TexCoord lin.Vec2
}

type VertexData []Vertex

func (v VertexData) Bytes() []byte {
	const m = 0x7fffffff
	vd := Vertex{}
	size := len(v) * int(unsafe.Sizeof(float32(1))) * int(unsafe.Sizeof(vd))
	return (*[m]byte)(unsafe.Pointer(&v[0]))[:size]
}

func (v VertexData) GetBindingDesciption() vk.VertexInputBindingDescription {
	var bindingDescription = vk.VertexInputBindingDescription{}
	bindingDescription.Binding = 0
	bindingDescription.Stride = uint32(unsafe.Sizeof(Vertex{}))
	bindingDescription.InputRate = vk.VertexInputRateVertex

	return bindingDescription
}

func (v VertexData) GetAttributeDescriptions() []vk.VertexInputAttributeDescription {
	attr := make([]vk.VertexInputAttributeDescription, 3)

	attr[0].Binding = 0
	attr[0].Location = 0
	attr[0].Format = vk.FormatR32g32b32Sfloat
	attr[0].Offset = 0

	attr[1].Binding = 0
	attr[1].Location = 1
	attr[1].Format = vk.FormatR32g32b32Sfloat
	attr[1].Offset = uint32(unsafe.Sizeof(lin.Vec3{}))

	attr[2].Binding = 0
	attr[2].Location = 2
	attr[2].Format = vk.FormatR32g32Sfloat
	attr[2].Offset = uint32(unsafe.Sizeof(lin.Vec3{})) * 2

	return attr

}

type IndexData []uint16

func (i IndexData) Bytes() []byte {
	const m = 0x7fffffff
	size := len(i) * int(unsafe.Sizeof(uint16(1)))
	return (*[m]byte)(unsafe.Pointer(&i[0]))[:size]
}

func (i IndexData) IndexType() vk.IndexType {
	return vk.IndexTypeUint16
}

type UBO struct {
	Model lin.Mat4x4
	View  lin.Mat4x4
	Proj  lin.Mat4x4
}

func (u *UBO) Bytes() []byte {
	const m = 0x7fffffff
	size := int(unsafe.Sizeof(float32(1))) * 4 * 4 * 3
	return (*[m]byte)(unsafe.Pointer(&u.Model[0]))[:size]
}

func (u *UBO) Descriptor() *vkg.Descriptor {
	return &vkg.Descriptor{
		Binding:     0,
		Set:         0,
		Type:        vk.DescriptorTypeUniformBuffer,
		ShaderStage: vk.ShaderStageFlags(vk.ShaderStageVertexBit),
	}
}

type Mesh struct {
	UBO        *UBO
	VertexData VertexData

	VertexBuffer  *vkg.HostBoundBuffer
	VertexBuffer2 *vkg.HostBoundBuffer

	UBOBuffers []*vkg.HostBoundBuffer

	textureView    vk.ImageView
	textureSampler vk.Sampler
}

func init() {
	runtime.LockOSThread()
}

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

type CubeDemo struct {
	app            *vkg.GraphicsApp
	pipelineLayout *vkg.PipelineLayout
	window         *glfw.Window

	mesh *Mesh

	vertices VertexData
	indices  []uint16

	descriptorSets []*vkg.DescriptorSet
}

func (c *CubeDemo) initMesh() {
	mesh := &Mesh{}

	var gVertexBufferData = []float32{
		-1.0, -1.0, -1.0, // -X side
		-1.0, -1.0, 1.0,
		-1.0, 1.0, 1.0,
		-1.0, 1.0, 1.0,
		-1.0, 1.0, -1.0,
		-1.0, -1.0, -1.0,

		-1.0, -1.0, -1.0, // -Z side
		1.0, 1.0, -1.0,
		1.0, -1.0, -1.0,
		-1.0, -1.0, -1.0,
		-1.0, 1.0, -1.0,
		1.0, 1.0, -1.0,

		-1.0, -1.0, -1.0, // -Y side
		1.0, -1.0, -1.0,
		1.0, -1.0, 1.0,
		-1.0, -1.0, -1.0,
		1.0, -1.0, 1.0,
		-1.0, -1.0, 1.0,

		-1.0, 1.0, -1.0, // +Y side
		-1.0, 1.0, 1.0,
		1.0, 1.0, 1.0,
		-1.0, 1.0, -1.0,
		1.0, 1.0, 1.0,
		1.0, 1.0, -1.0,

		1.0, 1.0, -1.0, // +X side
		1.0, 1.0, 1.0,
		1.0, -1.0, 1.0,
		1.0, -1.0, 1.0,
		1.0, -1.0, -1.0,
		1.0, 1.0, -1.0,

		-1.0, 1.0, 1.0, // +Z side
		-1.0, -1.0, 1.0,
		1.0, 1.0, 1.0,
		-1.0, -1.0, 1.0,
		1.0, -1.0, 1.0,
		1.0, 1.0, 1.0,
	}

	var gUVBufferData = []float32{
		0.0, 1.0, // -X side
		1.0, 1.0,
		1.0, 0.0,
		1.0, 0.0,
		0.0, 0.0,
		0.0, 1.0,

		1.0, 1.0, // -Z side
		0.0, 0.0,
		0.0, 1.0,
		1.0, 1.0,
		1.0, 0.0,
		0.0, 0.0,

		1.0, 0.0, // -Y side
		1.0, 1.0,
		0.0, 1.0,
		1.0, 0.0,
		0.0, 1.0,
		0.0, 0.0,

		1.0, 0.0, // +Y side
		0.0, 0.0,
		0.0, 1.0,
		1.0, 0.0,
		0.0, 1.0,
		1.0, 1.0,

		1.0, 0.0, // +X side
		0.0, 0.0,
		0.0, 1.0,
		0.0, 1.0,
		1.0, 1.0,
		1.0, 0.0,

		0.0, 0.0, // +Z side
		0.0, 1.0,
		1.0, 0.0,
		0.0, 1.0,
		1.0, 1.0,
		1.0, 0.0,
	}

	mesh.VertexData = make([]Vertex, len(gVertexBufferData)/3)
	for i := 0; i < len(gVertexBufferData); i = i + 3 {
		mesh.VertexData[i/3].Pos = lin.Vec3{gVertexBufferData[i] / 2, gVertexBufferData[i+1] / 2, gVertexBufferData[i+2] / 2}

	}
	for i := 0; i < len(gUVBufferData); i = i + 2 {
		mesh.VertexData[i/2].TexCoord = lin.Vec2{gUVBufferData[i], gUVBufferData[i+1]}
	}

	mesh.UBO = &UBO{}

	c.mesh = mesh
}

func (c *CubeDemo) loadTexture() {
	image, err := c.app.Device.StageImageFromDisk("image.png")
	orPanic(err)

	cb, err := c.app.GraphicsCommandPool.AllocateBuffer()
	orPanic(err)

	cb.BeginOneTime()
	cb.TransitionImageLayout(image, vk.FormatR8g8b8a8Unorm, vk.ImageLayoutUndefined, vk.ImageLayoutTransferDstOptimal)
	cb.CopyImage(image)
	cb.TransitionImageLayout(image, vk.FormatR8g8b8a8Unorm, vk.ImageLayoutTransferDstOptimal, vk.ImageLayoutShaderReadOnlyOptimal)
	cb.End()

	imageView, err := image.CreateImageViewWithAspectMask(vk.ImageAspectFlags(vk.ImageAspectColorBit))
	orPanic(err)

	c.app.GraphicsQueue.SubmitWaitIdle(cb)

	c.app.GraphicsCommandPool.FreeBuffer(cb)

	var sampler vk.Sampler
	vk.CreateSampler(c.app.Device.VKDevice, &vk.SamplerCreateInfo{
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

	c.mesh.textureSampler = sampler
	c.mesh.textureView = imageView.VKImageView

}

func (c *CubeDemo) init() {

	orPanic(glfw.Init())

	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	orPanic(vk.Init())

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(Width, Height, "VulkanCube", nil, nil)

	orPanic(err)

	c.window = window

	app, err := vkg.NewApp("VulkanCube", vkg.Version{1, 0, 0})
	orPanic(err)

	c.app = app

	app.SetWindow(window)
	app.EnableDebugging()

	err = app.Init()
	orPanic(err)

	c.initMesh()

	c.mesh.VertexBuffer, err = c.app.Device.CreateHostBoundBuffer(c.mesh.VertexData)
	orPanic(err)
	c.mesh.VertexBuffer.Map()

	c.mesh.UBOBuffers = make([]*vkg.HostBoundBuffer, 4)
	for i, _ := range c.mesh.UBOBuffers {
		c.mesh.UBOBuffers[i], err = c.app.Device.CreateHostBoundBuffer(c.mesh.UBO)
		orPanic(err)
		c.mesh.UBOBuffers[i].Map()
	}

	c.loadTexture()

	c.createDescriptorSet()

	gc := app.CreateGraphicsPipelineConfig()

	gc.SetVertexDescriptor(c.mesh.VertexData)
	gc.AddShaderStageFromFile("shaders/vert.spv", "main", vk.ShaderStageVertexBit)
	gc.AddShaderStageFromFile("shaders/frag.spv", "main", vk.ShaderStageFragmentBit)
	gc.SetPipelineLayout(c.pipelineLayout)

	app.AddGraphicsPipelineConfig("cube", gc)

	c.mesh.UBO.Model.Identity()

	c.mesh.UBO.View.LookAt(&lin.Vec3{2, 2, 2}, &lin.Vec3{0, 0, 0}, &lin.Vec3{0, 0, 1})

	c.app.MakeCommandBuffer = c.MakeCommandBuffer

	err = app.PrepareToDraw()
	orPanic(err)

}

func (c *CubeDemo) createDescriptorSet() {

	dsc := &vkg.DescriptorPoolContents{}
	dsc.AddPoolSize(vk.DescriptorTypeUniformBuffer, 4)
	dsc.AddPoolSize(vk.DescriptorTypeCombinedImageSampler, 4)
	dsp, err := c.app.Device.CreateDescriptorPool(4, dsc)
	orPanic(err)

	d := c.mesh.UBO.Descriptor()

	dsl := &vkg.DescriptorSetLayout{}
	dsl.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         uint32(d.Binding),
		DescriptorType:  d.Type,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(d.ShaderStage),
	})

	dsl.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:            1,
		DescriptorCount:    1,
		PImmutableSamplers: nil,
		StageFlags:         vk.ShaderStageFlags(vk.ShaderStageFragmentBit),
		DescriptorType:     vk.DescriptorTypeCombinedImageSampler,
	})

	c.app.Device.CreateDescriptorSetLayout(dsl)

	c.descriptorSets = make([]*vkg.DescriptorSet, 0)

	for i := 0; i < c.app.NumFramebuffers(); i++ {

		descriptorSet, err := dsp.Allocate(dsl)
		orPanic(err)

		descriptorSet.AddBuffer(0, vk.DescriptorTypeUniformBuffer, c.mesh.UBOBuffers[i].HostBuffer, 0)
		descriptorSet.AddCombinedImageSampler(1, vk.ImageLayoutShaderReadOnlyOptimal, c.mesh.textureView, c.mesh.textureSampler)
		descriptorSet.Write()
		orPanic(err)

		c.descriptorSets = append(c.descriptorSets, descriptorSet)
	}

	c.pipelineLayout, err = c.app.Device.CreatePipelineLayout(dsl)
	orPanic(err)

}

func (c *CubeDemo) MakeCommandBuffer(buffer *vkg.CommandBuffer, frame int) {

	var m lin.Mat4x4

	m.Dup(&c.mesh.UBO.Model)

	c.mesh.UBO.Model.Rotate(&m, 0.0, 0.0, 0.10, lin.DegreesToRadians(2))
	extent := c.app.GetScreenExtent()
	ratio := float32(extent.Width) / float32(extent.Height)
	c.mesh.UBO.Proj.Perspective(lin.DegreesToRadians(45), ratio, 0.1, 10.0)
	c.mesh.UBO.Proj[1][1] *= -1

	c.mesh.UBOBuffers[frame].Map()

	buffer.Reset()

	clearValues := make([]vk.ClearValue, 2)

	clearValues[0].SetColor([]float32{0.2, 0.2, 0.2, 1})
	clearValues[1].SetDepthStencil(1, 0)

	buffer.Begin()

	renderPassBeginInfo := vk.RenderPassBeginInfo{
		SType:       vk.StructureTypeRenderPassBeginInfo,
		RenderPass:  c.app.VKRenderPass,
		Framebuffer: c.app.Framebuffers[frame],
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{
				X: 0, Y: 0,
			},
			Extent: c.app.GetScreenExtent(),
		},
		ClearValueCount: 2,
		PClearValues:    clearValues,
	}

	vk.CmdBeginRenderPass(buffer.VK(), &renderPassBeginInfo, vk.SubpassContentsInline)

	vk.CmdBindPipeline(buffer.VK(), vk.PipelineBindPointGraphics, c.app.GraphicsPipelines["cube"])

	vk.CmdBindVertexBuffers(buffer.VK(), 0, 1, []vk.Buffer{c.mesh.VertexBuffer.HostBuffer.VKBuffer}, []vk.DeviceSize{0})

	vk.CmdBindDescriptorSets(buffer.VK(), vk.PipelineBindPointGraphics,
		c.pipelineLayout.VKPipelineLayout, 0, 1,
		[]vk.DescriptorSet{c.descriptorSets[frame].VKDescriptorSet}, 0, nil)

	vk.CmdDraw(buffer.VK(), uint32(len(c.mesh.VertexData)), 1, 0, 0)

	vk.CmdEndRenderPass(buffer.VK())

	buffer.End()
}

func (c *CubeDemo) run() {
	c.init()

	for {
		if c.window.ShouldClose() {
			return
		}
		glfw.PollEvents()
		err := c.app.DrawFrameSync()
		orPanic(err)

	}

	c.app.Destroy()

}

func main() {
	c := &CubeDemo{}
	c.run()
}
