package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/celer/vkg"
	"github.com/vulkan-go/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
	lin "github.com/xlab/linmath"
)

var Width = 800
var Height = 600

type Vertex struct {
	Pos   lin.Vec3
	Color lin.Vec3
}

type VertexData []Vertex

func (v VertexData) Bytes() []byte {
	const m = 0x7fffffff
	vd := Vertex{}
	size := len(v) * int(unsafe.Sizeof(vd))
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
	attr := make([]vk.VertexInputAttributeDescription, 2)

	attr[0].Binding = 0
	attr[0].Location = 0
	attr[0].Format = vk.FormatR32g32b32Sfloat
	attr[0].Offset = 0

	attr[1].Binding = 0
	attr[1].Location = 1
	attr[1].Format = vk.FormatR32g32b32Sfloat
	attr[1].Offset = uint32(unsafe.Sizeof(lin.Vec3{}))

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
	IndexData  IndexData

	VertexResource *vkg.BufferResource
	IndexResource  *vkg.BufferResource
	UBOResource    *vkg.BufferResource

	descriptorSet *vkg.DescriptorSet
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
}

func (c *CubeDemo) initMesh() {
	mesh := &Mesh{}

	mesh.VertexData = VertexData{
		//Top
		Vertex{Pos: lin.Vec3{0.5, 0.5, -0.5}, Color: lin.Vec3{0, 0, 1}},
		Vertex{Pos: lin.Vec3{0.5, -0.5, -0.5}, Color: lin.Vec3{0, 1, 0}},
		Vertex{Pos: lin.Vec3{0.5, -0.5, 0.5}, Color: lin.Vec3{1, 0, 0}},
		Vertex{Pos: lin.Vec3{0.5, 0.5, 0.5}, Color: lin.Vec3{0, 1, 1}},

		//Bottom
		Vertex{Pos: lin.Vec3{-0.5, 0.5, 0.5}, Color: lin.Vec3{0, 1, 1}},
		Vertex{Pos: lin.Vec3{-0.5, -0.5, 0.5}, Color: lin.Vec3{1, 1, 1}},
		Vertex{Pos: lin.Vec3{-0.5, -0.5, -0.5}, Color: lin.Vec3{1, 0, 1}},
		Vertex{Pos: lin.Vec3{-0.5, 0.5, -0.5}, Color: lin.Vec3{1, 1, 1}},
	}

	mesh.IndexData = IndexData{
		//X+
		2, 1, 0,
		3, 2, 0,

		//Y+
		4, 3, 0,
		7, 4, 0,

		//Z+
		5, 2, 3,
		4, 5, 3,

		//X-
		5, 7, 6,
		7, 5, 4,

		//Y+
		1, 2, 5,
		1, 5, 6,

		//Z-
		0, 1, 6,
		0, 6, 3,
	}

	mesh.UBO = &UBO{}

	c.mesh = mesh
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

	bytesNeeded := (len(c.mesh.VertexData.Bytes()) + len(c.mesh.IndexData.Bytes()) + len(c.mesh.UBO.Bytes())) + (128 * 3)

	cubePool, err := app.ResourceManager.AllocatePoolWithOptions("cube", uint64(bytesNeeded), vk.MemoryPropertyFlags(vk.MemoryPropertyHostCoherentBit|vk.MemoryPropertyHostVisibleBit), vk.BufferUsageFlags(vk.BufferUsageStorageBufferBit), vk.SharingModeExclusive)
	orPanic(err)

	c.mesh.VertexResource, err = cubePool.AllocateBuffer(uint64(len(c.mesh.VertexData.Bytes())), vk.BufferUsageFlags(vk.BufferUsageVertexBufferBit))
	orPanic(err)

	c.mesh.IndexResource, err = cubePool.AllocateBuffer(uint64(len(c.mesh.IndexData.Bytes())), vk.BufferUsageFlags(vk.BufferUsageIndexBufferBit))
	orPanic(err)

	c.mesh.UBOResource, err = cubePool.AllocateBuffer(uint64(len(c.mesh.UBO.Bytes())), vk.BufferUsageFlags(vk.BufferUsageUniformBufferBit))
	orPanic(err)

	// Map the data so we can simply write to it
	ptr, err := cubePool.Memory.Map()
	orPanic(err)

	fmt.Printf("Pool %v", cubePool.Allocator)

	vrb, err := c.mesh.VertexResource.Bytes()
	orPanic(err)
	irb, err := c.mesh.IndexResource.Bytes()
	orPanic(err)
	copy(vrb, c.mesh.VertexData.Bytes())
	copy(irb, c.mesh.IndexData.Bytes())

	c.mesh.UpdateUBO(c.app)

	fmt.Printf("VertexData\n")

	vb := c.mesh.VertexData.Bytes()
	for i := 0; i < len(vb); i++ {
		fmt.Printf("%x ", vb[i])
	}

	fmt.Printf("\n\n")

	fmt.Printf("IndexData\n")

	ib := c.mesh.IndexData.Bytes()
	for i := 0; i < len(ib); i++ {
		fmt.Printf("%x ", ib[i])
	}

	fmt.Printf("\n\n")

	fmt.Printf("IndexData\n")

	ub := c.mesh.UBO.Bytes()
	for i := 0; i < len(ub); i++ {
		fmt.Printf("%x ", ub[i])
	}

	fmt.Printf("\n\n")

	const m = 0x7FFFFFFF
	b := (*[m]byte)(ptr)[:bytesNeeded]
	for i := 0; i < bytesNeeded; i++ {
		fmt.Printf("%x ", b[i])
	}

	fmt.Printf("\n\n")

	descriptorSetLayout := c.createDescriptorSetLayout()
	orPanic(err)

	c.pipelineLayout, err = c.app.Device.CreatePipelineLayout(descriptorSetLayout)
	orPanic(err)

	gc := app.CreateGraphicsPipelineConfig()

	gc.SetVertexDescriptor(c.mesh.VertexData)
	gc.AddShaderStageFromFile("shaders/vert.spv", "main", vk.ShaderStageVertexBit)
	gc.AddShaderStageFromFile("shaders/frag.spv", "main", vk.ShaderStageFragmentBit)
	gc.SetPipelineLayout(c.pipelineLayout)

	app.AddGraphicsPipelineConfig("cube", gc)

	dsc := &vkg.DescriptorPoolContents{}
	dsc.AddPoolSize(vk.DescriptorTypeUniformBuffer, 1)
	dsp, err := c.app.Device.CreateDescriptorPool(1, dsc)
	orPanic(err)

	c.mesh.descriptorSet, err = dsp.Allocate(descriptorSetLayout)
	orPanic(err)

	c.mesh.descriptorSet.AddBuffer(0, vk.DescriptorTypeUniformBuffer, &c.mesh.UBOResource.Buffer, 0)
	c.mesh.descriptorSet.Write()

	c.mesh.UBO.Model.Identity()

	c.mesh.UBO.View.LookAt(&lin.Vec3{2, 2, 2}, &lin.Vec3{0, 0, 0}, &lin.Vec3{0, 0, 1})

	c.app.MakeCommandBuffer = c.MakeCommandBuffer

	err = app.PrepareToDraw()
	orPanic(err)

}

func (c *CubeDemo) createDescriptorSetLayout() *vkg.DescriptorSetLayout {
	dsl := &vkg.DescriptorSetLayout{}

	d := c.mesh.UBO.Descriptor()

	dsl.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         uint32(d.Binding),
		DescriptorType:  d.Type,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(d.ShaderStage),
	})

	descriptorSetLayout, err := c.app.Device.CreateDescriptorSetLayout(dsl)
	orPanic(err)

	return descriptorSetLayout

}

func (mesh *Mesh) UpdateUBO(app *vkg.GraphicsApp) {

	var m lin.Mat4x4
	m.Dup(&mesh.UBO.Model)
	mesh.UBO.Model.Rotate(&m, 0.0, 0.0, 0.60, lin.DegreesToRadians(2))
	extent := app.GetScreenExtent()
	ratio := float32(extent.Width) / float32(extent.Height)
	mesh.UBO.Proj.Perspective(lin.DegreesToRadians(45), ratio, 0.1, 10.0)
	mesh.UBO.Proj[1][1] *= -1

	ubr, err := mesh.UBOResource.Bytes()
	orPanic(err)

	copy(ubr, mesh.UBO.Bytes())
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

	vk.CmdBindIndexBuffer(buffer.VK(), c.mesh.IndexResource.VKBuffer, vk.DeviceSize(0), vk.IndexTypeUint16)

	vk.CmdBindDescriptorSets(buffer.VK(), vk.PipelineBindPointGraphics,
		c.pipelineLayout.VKPipelineLayout, 0, 1,
		[]vk.DescriptorSet{c.mesh.descriptorSet.VKDescriptorSet}, 0, nil)

	vk.CmdDrawIndexed(buffer.VK(), uint32(len(c.mesh.IndexData)), 1, 0, 0, 0)

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
