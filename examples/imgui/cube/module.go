package cube

import (
	"unsafe"

	"github.com/celer/vkg/examples/imgui/app"
	gui "github.com/celer/vkg/examples/imgui/imgui"
	"github.com/inkyblackness/imgui-go"

	"github.com/celer/vkg"
	vk "github.com/vulkan-go/vulkan"
	lin "github.com/xlab/linmath"
)

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

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

type Mesh struct {
	UBO        *UBO
	VertexData VertexData
	IndexData  IndexData

	VertexResource *vkg.BufferResource
	IndexResource  *vkg.BufferResource
	UBOResource    *vkg.BufferResource

	descriptorSetLayout *vkg.DescriptorSetLayout
	descriptorSet       *vkg.DescriptorSet
}

func (m *Mesh) Destroy() {
	m.VertexResource.Destroy()
	m.IndexResource.Destroy()
	m.UBOResource.Destroy()
	m.descriptorSetLayout.Destroy()
}

type CubeModule struct {
	pipelineLayout *vkg.PipelineLayout
	descriptorPool *vkg.DescriptorPool

	mesh *Mesh

	vertices VertexData
	indices  []uint16

	ui *gui.ImGUIModule

	spin bool
}

func NewCubeModule(app *app.AppBase, ui *gui.ImGUIModule) (*CubeModule, error) {

	c := &CubeModule{ui: ui, spin: true}

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

	bytesNeeded := (len(c.mesh.VertexData.Bytes()) + len(c.mesh.IndexData.Bytes()) + len(c.mesh.UBO.Bytes())) + (128 * 3)

	cubePool, err := app.ResourceManager.AllocateBufferPoolWithOptions("cube", uint64(bytesNeeded), vk.MemoryPropertyHostCoherentBit|vk.MemoryPropertyHostVisibleBit, vk.BufferUsageStorageBufferBit, vk.SharingModeExclusive)
	orPanic(err)

	c.mesh.VertexResource, err = cubePool.AllocateBuffer(uint64(len(c.mesh.VertexData.Bytes())), vk.BufferUsageVertexBufferBit)
	orPanic(err)

	c.mesh.IndexResource, err = cubePool.AllocateBuffer(uint64(len(c.mesh.IndexData.Bytes())), vk.BufferUsageIndexBufferBit)
	orPanic(err)

	c.mesh.UBOResource, err = cubePool.AllocateBuffer(uint64(len(c.mesh.UBO.Bytes())), vk.BufferUsageUniformBufferBit)
	orPanic(err)

	// Map the data so we can simply write to it
	_, err = cubePool.Memory.Map()
	orPanic(err)

	vrb := c.mesh.VertexResource.Bytes()
	irb := c.mesh.IndexResource.Bytes()

	copy(vrb, c.mesh.VertexData.Bytes())
	copy(irb, c.mesh.IndexData.Bytes())

	c.mesh.UpdateUBO(app)

	descriptorSetLayout := c.createDescriptorSetLayout(app)
	orPanic(err)
	c.mesh.descriptorSetLayout = descriptorSetLayout

	c.pipelineLayout, err = app.Device.CreatePipelineLayout(descriptorSetLayout)
	orPanic(err)

	gc := app.CreateGraphicsPipelineConfig()

	gc.AddVertexDescriptor(c.mesh.VertexData)
	gc.AddShaderStageFromFile("shaders/cube/vert.spv", "main", vk.ShaderStageVertexBit)
	gc.AddShaderStageFromFile("shaders/cube/frag.spv", "main", vk.ShaderStageFragmentBit)
	gc.SetPipelineLayout(c.pipelineLayout)

	app.AddGraphicsPipelineConfig("cube", gc)

	dpool := app.Device.NewDescriptorPool()
	dpool.AddPoolSize(vk.DescriptorTypeUniformBuffer, 1)
	_, err = app.Device.CreateDescriptorPool(dpool, 1)
	orPanic(err)

	c.descriptorPool = dpool

	c.mesh.descriptorSet, err = dpool.Allocate(descriptorSetLayout)
	orPanic(err)

	c.mesh.descriptorSet.AddBuffer(0, vk.DescriptorTypeUniformBuffer, &c.mesh.UBOResource.Buffer, 0)
	c.mesh.descriptorSet.Write()

	c.mesh.UBO.Model.Identity()

	c.mesh.UBO.View.LookAt(&lin.Vec3{2, 2, 2}, &lin.Vec3{0, 0, 0}, &lin.Vec3{0, 0, 1})

	c.ui.AddUI(c)

	return c, nil
}

func (c *CubeModule) DrawUI() {
	imgui.Begin("Cube")
	imgui.Checkbox("Spin", &c.spin)
	imgui.End()
}

func (c *CubeModule) createDescriptorSetLayout(app *app.AppBase) *vkg.DescriptorSetLayout {
	dsl := app.Device.NewDescriptorSetLayout()

	dsl.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vk.DescriptorTypeUniformBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageVertexBit),
	})

	descriptorSetLayout, err := app.Device.CreateDescriptorSetLayout(dsl)
	orPanic(err)

	return descriptorSetLayout

}

func (mesh *Mesh) UpdateUBO(app *app.AppBase) {
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

func (c *CubeModule) NewFrame(base *app.AppBase) {}
func (c *CubeModule) PostFrame()                 {}

func (c *CubeModule) Destroy() {
	c.mesh.Destroy()
	c.pipelineLayout.Destroy()
	c.descriptorPool.Destroy()
}
func (c *CubeModule) CreateCommandBuffers(renderPass vk.RenderPass, framebuffer vk.Framebuffer, app *app.AppBase) ([]vk.CommandBuffer, error) {
	if c.spin {

		c.mesh.UpdateUBO(app)

	}

	buffer, err := app.GraphicsCommandPool.AllocateBuffer(vk.CommandBufferLevelSecondary)
	if err != nil {
		return nil, err
	}
	buffer.BeginContinueRenderPass(renderPass, framebuffer)
	vk.CmdBindPipeline(buffer.VK(), vk.PipelineBindPointGraphics, app.GraphicsPipelines["cube"])

	vk.CmdBindVertexBuffers(buffer.VK(), 0, 1, []vk.Buffer{c.mesh.VertexResource.VKBuffer}, []vk.DeviceSize{0})

	vk.CmdBindIndexBuffer(buffer.VK(), c.mesh.IndexResource.VKBuffer, vk.DeviceSize(0), vk.IndexTypeUint16)

	vk.CmdBindDescriptorSets(buffer.VK(), vk.PipelineBindPointGraphics,
		c.pipelineLayout.VKPipelineLayout, 0, 1,
		[]vk.DescriptorSet{c.mesh.descriptorSet.VKDescriptorSet}, 0, nil)

	vk.CmdDrawIndexed(buffer.VK(), uint32(len(c.mesh.IndexData)), 1, 0, 0, 0)

	buffer.End()
	return []vk.CommandBuffer{buffer.VK()}, nil

}
