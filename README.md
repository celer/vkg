# Intro

This repo is very much a work in progress of utilizing Vulkan with go. I've wrapped the vulkan-go/vulkan APIs to make
them a little more idiomatic, and easier to use and also provided a bunch of utility classes. 

Here is where I'm at:

  * Provides an easier to use API than the native Vulkan APIs
  * Provides for access to all underlying vulkan data structures, so it makes things easy without hiding the necessary bits to utilize the full vulkan API
  * Works with ImGUI
  * Custom memory allocator see allocator.go
  * Utility class called GraphicsApp which does most of the bootstrapping required to get a vulkan app up and going
  * Can display meshes and textures
  
If you want to get a good idea of where I'm going checkout examples/imgui

Here is where I expect to go;

  * More documentation
  * More examples
  * Unit tests

I'm hoping to continue pushing on this repo more in the next few weeks. 

# Screenshots

Here is a picture of the examples/imgui program:

![Example program](/assets/imgui.png)

Here is a picture of the examples/texture program:

![Example program](/assets/texture.png)


# Quick example code

[Here is the example this code comes from](/examples/cube/cube.go)

How to initialize a new graphics app:
```go
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
```

How to create memory resources from allocated pools

```go
// we allocate a new memory pool, with the size we calculated above, and we tell vulkan where we'd like to store the data
// in this case we're gonna store all our data in the host's memory and use a memory map to sync the data to the GPU
// so we specify HostVisible|HostCoherent to make sure we can memory map the data, and specify that we want to use this buffer
// for vertex, index and uniform buffer storage
cubePool, _ := c.app.ResourceManager.AllocateBufferPoolWithOptions("cube", uint64(bytesNeeded),
	vk.MemoryPropertyHostCoherentBit|vk.MemoryPropertyHostVisibleBit,
	vk.BufferUsageVertexBufferBit|vk.BufferUsageIndexBufferBit|vk.BufferUsageUniformBufferBit,
	vk.SharingModeExclusive)

m.VertexResource, _ = cubePool.AllocateBuffer(uint64(len(m.VertexData.Bytes())), vk.BufferUsageVertexBufferBit)

m.IndexResource, _ = cubePool.AllocateBuffer(uint64(len(m.IndexData.Bytes())), vk.BufferUsageIndexBufferBit)

m.UBOResource, _ = cubePool.AllocateBuffer(uint64(len(m.UBO.Bytes())), vk.BufferUsageUniformBufferBit)
```

How to map memory:

```go
// Map the data so we can simply write to it
cubePool.Memory.Map()

copy(m.VertexResource.Bytes(), m.VertexData.Bytes())
copy(m.IndexResource.Bytes(), m.IndexData.Bytes())

```

How to configure a custom graphics pipeline
```go
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
```

How to send commands to the graphics queue

```go
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
```


