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

func (v VertexData) GetBindingDescription() vk.VertexInputBindingDescription {
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

	VertexResource *vkg.BufferResource
	UBOResource    *vkg.BufferResource

	descriptorSet *vkg.DescriptorSet

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

	cb, err := c.app.GraphicsCommandPool.AllocateBuffer(vk.CommandBufferLevelPrimary)
	orPanic(err)

	tpool := c.app.ResourceManager.ImagePool("textures")

	if tpool == nil {
		panic("No texture pool found")
	}

	textureResource, err := tpool.StageTextureFromDisk("image.png", cb, c.app.GraphicsQueue)
	orPanic(err)

	c.app.GraphicsCommandPool.FreeBuffer(cb)

	imageView, err := textureResource.CreateImageViewWithAspectMask(vk.ImageAspectFlags(vk.ImageAspectColorBit))
	orPanic(err)

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

	app, err := vkg.NewGraphicsApp("VulkanCube", vkg.Version{1, 0, 0})
	orPanic(err)

	c.app = app

	app.SetWindow(window)
	app.EnableDebugging()

	err = app.Init()
	orPanic(err)

	_, err = app.ResourceManager.AllocateBufferPoolWithOptions("staging", 60*1024*1024, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit, vk.BufferUsageTransferSrcBit, vk.SharingModeExclusive)
	orPanic(err)

	_, err = app.ResourceManager.AllocateImagePoolWithOptions("textures", 60*1024*1024, vk.MemoryPropertyDeviceLocalBit, vk.ImageUsageTransferDstBit|vk.ImageUsageSampledBit, vk.SharingModeExclusive)
	orPanic(err)

	c.initMesh()

	size := len(c.mesh.VertexData.Bytes()) + len(c.mesh.UBO.Bytes()) + 128

	cubePool, err := c.app.ResourceManager.AllocateBufferPoolWithOptions("cube", uint64(size), vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit, vk.BufferUsageVertexBufferBit|vk.BufferUsageUniformBufferBit, vk.SharingModeExclusive)
	orPanic(err)

	c.mesh.VertexResource, err = cubePool.AllocateBuffer(uint64(len(c.mesh.VertexData.Bytes())), vk.BufferUsageVertexBufferBit)
	orPanic(err)

	c.mesh.UBOResource, err = cubePool.AllocateBuffer(uint64(len(c.mesh.UBO.Bytes())), vk.BufferUsageUniformBufferBit)
	orPanic(err)

	// Map the data so we can simply write to it
	_, err = cubePool.Memory.Map()
	orPanic(err)

	vrb := c.mesh.VertexResource.Bytes()
	copy(vrb, c.mesh.VertexData.Bytes())

	c.mesh.UpdateUBO(c.app)

	c.loadTexture()

	c.createDescriptorSet()

	gc := app.CreateGraphicsPipelineConfig()

	gc.AddVertexDescriptor(c.mesh.VertexData)
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

func (mesh *Mesh) UpdateUBO(app *vkg.GraphicsApp) {

	var m lin.Mat4x4
	m.Dup(&mesh.UBO.Model)
	mesh.UBO.Model.Rotate(&m, 0.0, 0.0, 0.60, lin.DegreesToRadians(2))
	extent := app.GetScreenExtent()
	ratio := float32(extent.Width) / float32(extent.Height)
	mesh.UBO.Proj.Perspective(lin.DegreesToRadians(45), ratio, 0.1, 10.0)
	mesh.UBO.Proj[1][1] *= -1

	ubr := mesh.UBOResource.Bytes()

	copy(ubr, mesh.UBO.Bytes())
}

func (c *CubeDemo) createDescriptorSet() {

	dpool := c.app.Device.NewDescriptorPool()
	dpool.AddPoolSize(vk.DescriptorTypeUniformBuffer, 4)
	dpool.AddPoolSize(vk.DescriptorTypeCombinedImageSampler, 4)
	_, err := c.app.Device.CreateDescriptorPool(dpool, 4)
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

	c.mesh.descriptorSet, err = dpool.Allocate(dsl)

	c.mesh.descriptorSet.AddBuffer(0, vk.DescriptorTypeUniformBuffer, &c.mesh.UBOResource.Buffer, 0)
	c.mesh.descriptorSet.AddCombinedImageSampler(1, vk.ImageLayoutShaderReadOnlyOptimal, c.mesh.textureView, c.mesh.textureSampler)
	c.mesh.descriptorSet.Write()
	orPanic(err)

	c.pipelineLayout, err = c.app.Device.CreatePipelineLayout(dsl)
	orPanic(err)

}

func (c *CubeDemo) MakeCommandBuffer(buffer *vkg.CommandBuffer, frame int) {

	c.mesh.UpdateUBO(c.app)

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

	vk.CmdBindVertexBuffers(buffer.VK(), 0, 1, []vk.Buffer{c.mesh.VertexResource.VKBuffer}, []vk.DeviceSize{0})

	vk.CmdBindDescriptorSets(buffer.VK(), vk.PipelineBindPointGraphics,
		c.pipelineLayout.VKPipelineLayout, 0, 1,
		[]vk.DescriptorSet{c.mesh.descriptorSet.VKDescriptorSet}, 0, nil)

	vk.CmdDraw(buffer.VK(), uint32(len(c.mesh.VertexData)), 1, 0, 0)

	vk.CmdEndRenderPass(buffer.VK())

	buffer.End()
}

func (c *CubeDemo) run() {
	c.init()

	for {
		if c.window.ShouldClose() {
			break
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
