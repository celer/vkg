package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"

	vkg "github.com/celer/vkg"
)

const WIDTH = 3200
const HEIGHT = 2400
const WORKGROUP_SIZE = 32

func orPanic(err error) {
	if err != nil {
		panic(err)
	}

}

type Pixel struct {
	r float32
	g float32
	b float32
	a float32
}

func main() {

	err := vkg.InitializeForComputeOnly()
	orPanic(err)

	app := vkg.App{
		Name: "TestApp",
	}

	app.EnableDebugging()

	instance, err := app.CreateInstance()
	orPanic(err)

	pdevices, err := instance.PhysicalDevices()
	orPanic(err)

	if len(pdevices) == 0 {
		panic("no physical devices found")
	}

	pdevice := pdevices[0]

	queues, err := pdevice.QueueFamilies()
	orPanic(err)

	ldevice, err := pdevice.CreateLogicalDevice(queues.FilterCompute())
	orPanic(err)

	computeQueue := ldevice.GetQueue(queues.FilterCompute()[0])

	rm := ldevice.CreateResourceManager()
	p := Pixel{}

	bytesNeeded := uint64(WIDTH * HEIGHT * int(unsafe.Sizeof(p)))

	rpool, err := rm.AllocateBufferPoolWithOptions("compute", bytesNeeded, vk.MemoryPropertyHostCoherentBit|vk.MemoryPropertyHostVisibleBit, vk.BufferUsageStorageBufferBit, vk.SharingModeExclusive)
	orPanic(err)

	bres, err := rpool.AllocateBuffer(bytesNeeded, vk.BufferUsageStorageBufferBit)
	orPanic(err)

	dsl := &vkg.DescriptorSetLayout{}

	dsl.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vk.DescriptorTypeStorageBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageComputeBit),
	})

	dsl, err = ldevice.CreateDescriptorSetLayout(dsl)

	dpool := ldevice.NewDescriptorPool()
	dpool.AddPoolSize(vk.DescriptorTypeStorageBuffer, 1)

	_, err = ldevice.CreateDescriptorPool(dpool, 1)
	orPanic(err)

	dset, err := dpool.Allocate(dsl)
	orPanic(err)

	dset.AddBuffer(0, vk.DescriptorTypeStorageBuffer, &bres.Buffer, 0)

	dset.Write()

	shader, err := ldevice.LoadShaderModuleFromFile("shaders/comp.spv")
	orPanic(err)

	pipelineLayout, err := ldevice.CreatePipelineLayout(dsl)
	orPanic(err)

	computePipeline := &vkg.ComputePipeline{}
	computePipeline.SetShaderStage("main", shader)
	computePipeline.SetPipelineLayout(pipelineLayout)

	cache, err := ldevice.CreatePipelineCache()
	orPanic(err)

	err = ldevice.CreateComputePipelines(cache, computePipeline)
	orPanic(err)

	cpool, err := ldevice.CreateCommandPool(queues.FilterCompute()[0])
	orPanic(err)

	cb, err := cpool.AllocateBuffer(vk.CommandBufferLevelPrimary)
	orPanic(err)

	err = cb.BeginOneTime()
	orPanic(err)

	cb.CmdBindComputePipeline(computePipeline)
	cb.CmdBindDescriptorSets(vk.PipelineBindPointCompute, pipelineLayout, 0, dset)
	cb.CmdDispatch(int(math.Ceil(float64(WIDTH/float32(WORKGROUP_SIZE)))),
		int(math.Ceil(float64(HEIGHT/float32(WORKGROUP_SIZE)))), 1)

	cb.End()

	fence, err := ldevice.CreateFence()
	orPanic(err)

	computeQueue.SubmitWithFence(fence, cb)

	ldevice.WaitForFences(true, 10*time.Second, fence)

	cpool.FreeBuffer(cb)

	rpool.Memory.Map()

	data := bres.Bytes()

	saveImage(data)

	rpool.Memory.Unmap()
	bres.Free()
	fence.Destroy()
	rpool.Destroy()
	dpool.Free(dset)
	dpool.Destroy()
	pipelineLayout.Destroy()
	computePipeline.Destroy()
	cache.Destroy()
	dsl.Destroy()
	cpool.Destroy()
	shader.Destroy()
	ldevice.Destroy()
	instance.Destroy()

}

func saveImage(data []byte) {

	out := image.NewRGBA(image.Rectangle{
		Max: image.Point{
			X: WIDTH, Y: HEIGHT,
		},
	})
	const s = WIDTH * HEIGHT

	pixels := (*[s]Pixel)(unsafe.Pointer(&data[0]))[:s]

	out.Pix = make([]uint8, WIDTH*HEIGHT*4)

	for y := 0; y < HEIGHT; y++ {
		for x := 0; x < WIDTH; x++ {
			out.Set(x, y, color.RGBA{
				uint8(pixels[y*WIDTH+x].r * 255),
				uint8(pixels[y*WIDTH+x].g * 255),
				uint8(pixels[y*WIDTH+x].b * 255),
				uint8(pixels[y*WIDTH+x].a * 255),
			})
		}
	}

	outFile, err := os.OpenFile("out.png", os.O_CREATE|os.O_RDWR, 0644)
	orPanic(err)

	png.Encode(outFile, out)

}
