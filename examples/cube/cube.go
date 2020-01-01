package main

import (
	"runtime"
	"unsafe"

	"github.com/celer/vkg"
	"github.com/vulkan-go/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
	lin "github.com/xlab/linmath"
)

func init() {
	runtime.LockOSThread()
}

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

var Width = 800
var Height = 600

type Vertex struct {
	Pos   lin.Vec3
	Color lin.Vec3
}

type VertexData []Vertex

func (v VertexData) Bytes() []byte {
	size := len(v) * int(unsafe.Sizeof(Vertex{}))
	return vkg.ToBytes(unsafe.Pointer(&v[0]), size)
}

func (v VertexData) GetBindingDescription() vk.VertexInputBindingDescription {
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

type UBO struct {
	Model lin.Mat4x4
	View  lin.Mat4x4
	Proj  lin.Mat4x4
}

func (u *UBO) Bytes() []byte {
	size := int(unsafe.Sizeof(float32(1))) * 4 * 4 * 3
	return vkg.ToBytes(unsafe.Pointer(&u.Model[0]), size)
}

type Mesh struct {
	UBO        *UBO
	VertexData VertexData
	IndexData  vkg.IndexSliceUint16

	VertexResource *vkg.BufferResource
	IndexResource  *vkg.BufferResource
	UBOResource    *vkg.BufferResource

	descriptorSet *vkg.DescriptorSet
}

func (m *Mesh) Destroy() {
	m.VertexResource.Destroy()
	m.IndexResource.Destroy()
	m.UBOResource.Destroy()
}

func NewMesh() *Mesh {
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

	mesh.IndexData = vkg.IndexSliceUint16{
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
	return mesh
}

func (m *Mesh) setupBuffers(app *vkg.GraphicsApp, cubePool *vkg.BufferResourcePool) {
	var err error

	m.VertexResource, err = cubePool.AllocateBuffer(uint64(len(m.VertexData.Bytes())), vk.BufferUsageVertexBufferBit)
	orPanic(err)

	m.IndexResource, err = cubePool.AllocateBuffer(uint64(len(m.IndexData.Bytes())), vk.BufferUsageIndexBufferBit)
	orPanic(err)

	m.UBOResource, err = cubePool.AllocateBuffer(uint64(len(m.UBO.Bytes())), vk.BufferUsageUniformBufferBit)
	orPanic(err)

	// Map the data so we can simply write to it
	_, err = cubePool.Memory.Map()
	orPanic(err)

	copy(m.VertexResource.Bytes(), m.VertexData.Bytes())
	copy(m.IndexResource.Bytes(), m.IndexData.Bytes())

	m.updateUBO(app)

}

func (mesh *Mesh) updateUBO(app *vkg.GraphicsApp) {

	var m lin.Mat4x4
	m.Dup(&mesh.UBO.Model)
	mesh.UBO.Model.Rotate(&m, 0.0, 0.0, 0.60, lin.DegreesToRadians(2))
	extent := app.GetScreenExtent()
	ratio := float32(extent.Width) / float32(extent.Height)
	mesh.UBO.Proj.Perspective(lin.DegreesToRadians(45), ratio, 0.1, 10.0)
	mesh.UBO.Proj[1][1] *= -1

	copy(mesh.UBOResource.Bytes(), mesh.UBO.Bytes())
}

type CubeDemo struct {
	app            *vkg.GraphicsApp
	pipelineLayout *vkg.PipelineLayout
	descriptorPool *vkg.DescriptorPool
	window         *glfw.Window

	descriptorSetLayout *vkg.DescriptorSetLayout

	mesh *Mesh

	vertices VertexData
	indices  []uint16
}

func (c *CubeDemo) init() {

	// we initialize glfw, so we can create a window
	glfw.Init()

	// setup vulkan
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	vk.Init()

	// create our window
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	c.window, _ = glfw.CreateWindow(Width, Height, "VulkanCube", nil, nil)

	// create a new graphics app
	c.app, _ = vkg.NewGraphicsApp("VulkanCube", vkg.Version{1, 0, 0})

	// we need to do some configuration before we can
	// can initalize our application
	c.app.SetWindow(c.window)
	c.app.EnableDebugging()

	// initialize our graphics app
	c.app.Init()

	// create a new mesh
	c.mesh = NewMesh()

	// allocate enough data to hold all the vertices, indexes and matrix data, plus some extra to account for alignment adjustments.
	// vulkan is a stickler about memory alignments, so for example the pool we create below might need to align some times of memory
	// to meet certain requirements
	bytesNeeded := (len(c.mesh.VertexData.Bytes()) + len(c.mesh.IndexData.Bytes()) + len(c.mesh.UBO.Bytes())) + (128 * 3)

	// we allocate a new memory pool, with the size we calculated above, and we tell vulkan where we'd like to store the data
	// in this case we're gonna store all our data in the host's memory and use a memory map to sync the data to the GPU
	// so we specify HostVisible|HostCoherent to make sure we can memory map the data, and specify that we want to use this buffer
	// for vertex, index and uniform buffer storage
	cubePool, _ := c.app.ResourceManager.AllocateBufferPoolWithOptions("cube", uint64(bytesNeeded),
		vk.MemoryPropertyHostCoherentBit|vk.MemoryPropertyHostVisibleBit,
		vk.BufferUsageVertexBufferBit|vk.BufferUsageIndexBufferBit|vk.BufferUsageUniformBufferBit,
		vk.SharingModeExclusive)

	// Next we setup the buffers using the pool we just created
	c.mesh.setupBuffers(c.app, cubePool)

	// now we need to create a descriptor set which is kinda like defining a struct
	// in go - it describes the format of the data that we want to provide to our shaders
	c.descriptorSetLayout = c.createDescriptorSetLayout()
	// we will use this descriptorSetLayout as an input to our graphics pipeline we'll
	// create later, but essentially this informs the graphics pipeline about how
	// we will layout our data in our descriptor set (again layout = struct definition)
	c.pipelineLayout, _ = c.app.Device.CreatePipelineLayout(c.descriptorSetLayout)

	// now we need to create a descriptor pool, because vulkan is so focused on
	// performance pools, like the descriptor pool are frequently used to manage
	// resources we've created - so we need to create a descriptor pool to
	// manage our descriptors and avoid recreating them
	c.descriptorPool = c.createDescriptorPool()

	// next we will create our descriptor set, this is like allocating an
	// instance of the descriptor layout we defined above. Above we defined
	// how we'd layout the data in our descriptor set, now time to actually
	// allocate a descriptor and bind data into it.
	c.createDescriptorSet(c.descriptorPool, c.descriptorSetLayout)

	// the graphics pipline describes how we will display the data in our mesh and
	// how we will provide data to our shaders.
	c.createGraphicsPipeline()

	// Next we need to setup some matrixes to view our mesh
	c.mesh.UBO.Model.Identity()
	c.mesh.UBO.View.LookAt(&lin.Vec3{2, 2, 2}, &lin.Vec3{0, 0, 0}, &lin.Vec3{0, 0, 1})

	// now we need to tell the graphics app how to make command buffers
	c.app.MakeCommandBuffer = c.MakeCommandBuffer

	// now that we've done all this ground work we can go draw some stuff on the screen
	c.app.PrepareToDraw()

}

func (c *CubeDemo) createDescriptorPool() *vkg.DescriptorPool {

	dpool := c.app.Device.NewDescriptorPool()
	dpool.AddPoolSize(vk.DescriptorTypeUniformBuffer, 1)
	c.app.Device.CreateDescriptorPool(dpool, 1)

	return dpool

}

func (c *CubeDemo) createDescriptorSet(pool *vkg.DescriptorPool, descriptorSetLayout *vkg.DescriptorSetLayout) {
	// Alloate a descriptor from our pool and bind our UBO to it
	c.mesh.descriptorSet, _ = pool.Allocate(descriptorSetLayout)
	c.mesh.descriptorSet.AddBuffer(0, vk.DescriptorTypeUniformBuffer, &c.mesh.UBOResource.Buffer, 0)
	c.mesh.descriptorSet.Write()
}

func (c *CubeDemo) createGraphicsPipeline() {

	// create a graphics pipeline
	gc := c.app.CreateGraphicsPipelineConfig()

	// our mesh implements in interface which allows it
	// to describe how it's vertex data is layed out, so we provide
	// that interface to our graphics pipeline
	gc.AddVertexDescriptor(c.mesh.VertexData)

	// load some shaders
	gc.AddShaderStageFromFile("shaders/vert.spv", "main", vk.ShaderStageVertexBit)
	gc.AddShaderStageFromFile("shaders/frag.spv", "main", vk.ShaderStageFragmentBit)

	// set our pipeline layout which describes how we wish to layout our data in our descriptors
	gc.SetPipelineLayout(c.pipelineLayout)

	// lastly we tell our graphics app about this pipeline config
	// we use named pipeline configs because at it's descretion the graphics app
	// must be able to recreate the actual pipelines from it's configs
	c.app.AddGraphicsPipelineConfig("cube", gc)
}

func (c *CubeDemo) createDescriptorSetLayout() *vkg.DescriptorSetLayout {

	// define our descriptor set layout, again
	// this is much like defining a struct
	dsl := c.app.Device.NewDescriptorSetLayout()

	dsl.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vk.DescriptorTypeUniformBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageVertexBit),
	})

	descriptorSetLayout, _ := c.app.Device.CreateDescriptorSetLayout(dsl)

	return descriptorSetLayout

}

func (c *CubeDemo) MakeCommandBuffer(buffer *vkg.CommandBuffer, frame int) {
	// notice the 'frame' parameter in our function signature?
	// it's required because we are rotating through a small number of
	// frame buffers. So if we build a command buffer with the same
	// vertex buffer for each frame buffer we may have issues with
	// our result. So we need to either make sure that our command
	// buffers are not using the same resources, because we may have
	// the GPU working on multiple command buffers at once. In this
	// case were not concerned because we've purposefully told
	// the graphics app to draw one frame at a time, without allowing
	// resources to overlap.

	// update our matricies so that our cube rotates
	c.mesh.updateUBO(c.app)

	// command buffers are a sequence of instructions that the GPU
	// will execute. In this case our graphics app that we are using
	// will do all sorts of management around utilizing command buffers.
	//
	// but the gist of it is that the GPU has some number of work queues
	// for different purposes that queues can be submitted to.

	// because command buffers are allocated from a pool
	// and reused we must reset it
	buffer.Reset()

	// clear values are used to clear the screen and depth buffer
	clearValues := make([]vk.ClearValue, 2)
	clearValues[0].SetColor([]float32{0.2, 0.2, 0.2, 1})
	clearValues[1].SetDepthStencil(1, 0)

	// begin recording commands
	buffer.Begin()

	// create a render pass struct
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

	// we tell vulkan which graphics pipeline we want to use - the one we defined above
	vk.CmdBindPipeline(buffer.VK(), vk.PipelineBindPointGraphics, c.app.GraphicsPipelines["cube"])

	// tell it which buffer our vertex data comes from
	vk.CmdBindVertexBuffers(buffer.VK(), 0, 1, []vk.Buffer{c.mesh.VertexResource.VKBuffer}, []vk.DeviceSize{0})

	// tell it which buffer our index data comes from
	vk.CmdBindIndexBuffer(buffer.VK(), c.mesh.IndexResource.VKBuffer, vk.DeviceSize(0), vk.IndexTypeUint16)

	// tell vulkan about our descriptor sets which feed data to our shaders
	vk.CmdBindDescriptorSets(buffer.VK(), vk.PipelineBindPointGraphics,
		c.pipelineLayout.VKPipelineLayout, 0, 1,
		[]vk.DescriptorSet{c.mesh.descriptorSet.VKDescriptorSet}, 0, nil)

	// lastly tell vulkan which indexes to draw
	vk.CmdDrawIndexed(buffer.VK(), uint32(len(c.mesh.IndexData)), 1, 0, 0, 0)

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
	c.Destroy()
	c.app.Destroy()
}

func (c *CubeDemo) Destroy() {
	c.mesh.Destroy()
	c.pipelineLayout.Destroy()
	c.descriptorPool.Destroy()
	c.descriptorSetLayout.Destroy()
}

func main() {
	c := &CubeDemo{}
	c.run()
}
