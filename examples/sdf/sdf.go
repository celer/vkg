package main

import (
	"fmt"
	"math"
	"os"
	"time"
	"unsafe"

	vkg "github.com/celer/vkg"
	vk "github.com/vulkan-go/vulkan"
)

func orPanic(err error) {
	if err != nil {
		panic(err)
	}

}

type Exec struct {
	mode  float32
	scale float32
	minX  float32
	minY  float32
	minZ  float32
	maxX  float32
	maxY  float32
	maxZ  float32
}

type Bounds [3]float32

const WorkgroupSize = 10

func (e *Exec) Bytes() []byte {
	s := int(unsafe.Sizeof(float32(1))) * 8
	return vkg.ToBytes(unsafe.Pointer(&e.mode), s)
}

type Float32Slice []float32

func (s Float32Slice) Bytes() []byte {
	return vkg.ToBytes(unsafe.Pointer(&s[0]), int(unsafe.Sizeof(float32(1)))*len(s))
}

type Uint32Slice []uint32

func (s Uint32Slice) Bytes() []byte {
	return vkg.ToBytes(unsafe.Pointer(&s[0]), int(unsafe.Sizeof(uint32(1)))*len(s))
}

type SDFMarcher struct {
	exec *Exec

	mesh  []float32
	count Uint32Slice

	execResource  *vkg.BufferResource
	countResource *vkg.BufferResource
	meshResource  *vkg.BufferResource

	tricount uint32

	shader         *vkg.ShaderModule
	pipelineCache  *vkg.PipelineCache
	pipelineLayout *vkg.PipelineLayout

	descriptorPool      *vkg.DescriptorPool
	descriptorSet       *vkg.DescriptorSet
	descriptorSetLayout *vkg.DescriptorSetLayout

	computePipeline *vkg.ComputePipeline
}

func NewSDFMarcher(min, max Bounds, scale float32) *SDFMarcher {
	e := &Exec{
		minX:  min[0],
		minY:  min[1],
		minZ:  min[2],
		maxX:  max[0],
		maxY:  max[1],
		maxZ:  max[2],
		scale: scale,
		mode:  0,
	}
	s := &SDFMarcher{
		exec: e,
	}

	wx, wy, wz := s.GetWorkgroupSize()

	fmt.Printf("Work group size %d %d %d\n", wx, wy, wz)

	s.count = make([]uint32, wx*wy*wz)

	return s
}

func (s *SDFMarcher) GetWorkgroupSize() (int, int, int) {
	dx, dy, dz := s.GetSize()
	wf := float64(WorkgroupSize)
	return int(math.Ceil(float64(dx) / wf)), int(math.Ceil(float64(dy) / wf)), int(math.Ceil(float64(dz) / wf))
}

func (s *SDFMarcher) GetSize() (int, int, int) {
	dx := int((s.exec.maxX - s.exec.minX) / s.exec.scale)
	dy := int((s.exec.maxY - s.exec.minY) / s.exec.scale)
	dz := int((s.exec.maxZ - s.exec.minZ) / s.exec.scale)
	return dx, dy, dz
}

func (s *SDFMarcher) Init(rpool *vkg.BufferResourcePool, device *vkg.Device) error {
	var err error
	s.descriptorPool = device.NewDescriptorPool()
	s.descriptorPool.AddPoolSize(vk.DescriptorTypeStorageBuffer, 5)

	_, err = device.CreateDescriptorPool(s.descriptorPool, 5)
	if err != nil {
		return err
	}

	s.pipelineCache, err = device.CreatePipelineCache()
	if err != nil {
		return err
	}

	err = s.createDescriptorSet(rpool, device)
	if err != nil {
		return err
	}
	err = s.createPipeline(device)
	if err != nil {
		return err
	}
	return nil
}

func (s *SDFMarcher) createPipeline(device *vkg.Device) error {
	var err error

	s.pipelineLayout, err = device.CreatePipelineLayout(s.descriptorSetLayout)
	if err != nil {
		return err
	}

	s.shader, err = device.LoadShaderModuleFromFile("shaders/sdf.comp.spv")
	if err != nil {
		return err
	}
	s.computePipeline = &vkg.ComputePipeline{}
	s.computePipeline.SetShaderStage("main", s.shader)
	s.computePipeline.SetPipelineLayout(s.pipelineLayout)
	err = device.CreateComputePipelines(s.pipelineCache, s.computePipeline)
	if err != nil {
		return err
	}
	return nil

}

// Need to do two of these, one if no buffer is set
// one if a buffer is set
func (s *SDFMarcher) Process(cb *vkg.CommandBuffer) {
	copy(s.execResource.Bytes(), s.exec.Bytes())

	cb.CmdBindComputePipeline(s.computePipeline)
	cb.CmdBindDescriptorSets(vk.PipelineBindPointCompute, s.pipelineLayout, 0, s.descriptorSet)
	wx, wy, wz := s.GetWorkgroupSize()
	cb.CmdDispatch(wx, wy, wz)

}

func (s *SDFMarcher) GetTriangleCount() uint32 {
	s.tricount = 0
	copy(s.count.Bytes(), s.countResource.Bytes())
	for _, c := range toUint32(s.count.Bytes()) {
		s.tricount += c

	}
	return s.tricount
}

func (s *SDFMarcher) March(device *vkg.Device, triangle *vkg.BufferResource, cb *vkg.CommandBuffer) error {
	var err error
	descriptorSetLayout := &vkg.DescriptorSetLayout{}

	descriptorSetLayout.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vk.DescriptorTypeStorageBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageComputeBit),
	})

	descriptorSetLayout.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         1,
		DescriptorType:  vk.DescriptorTypeStorageBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageComputeBit),
	})

	descriptorSetLayout.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         2,
		DescriptorType:  vk.DescriptorTypeStorageBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageComputeBit),
	})

	_, err = device.CreateDescriptorSetLayout(descriptorSetLayout)
	if err != nil {
		return err
	}

	descriptorSet, err := s.descriptorPool.Allocate(descriptorSetLayout)
	if err != nil {
		return err
	}
	descriptorSet.AddBuffer(0, vk.DescriptorTypeStorageBuffer, &s.execResource.Buffer, 0)
	descriptorSet.AddBuffer(1, vk.DescriptorTypeStorageBuffer, &s.countResource.Buffer, 0)
	descriptorSet.AddBuffer(2, vk.DescriptorTypeStorageBuffer, &triangle.Buffer, 0)
	descriptorSet.Write()

	pipelineLayout, err := device.CreatePipelineLayout(descriptorSetLayout)
	if err != nil {
		return err
	}

	s.exec.mode = 1

	copy(s.execResource.Bytes(), s.exec.Bytes())

	computePipeline := &vkg.ComputePipeline{}
	computePipeline.SetShaderStage("main", s.shader)
	computePipeline.SetPipelineLayout(pipelineLayout)
	err = device.CreateComputePipelines(s.pipelineCache, computePipeline)
	if err != nil {
		return err
	}

	cb.CmdBindComputePipeline(computePipeline)
	cb.CmdBindDescriptorSets(vk.PipelineBindPointCompute, pipelineLayout, 0, descriptorSet)
	wx, wy, wz := s.GetWorkgroupSize()
	cb.CmdDispatch(wx, wy, wz)

	return nil
}

func (s *SDFMarcher) createDescriptorSet(rpool *vkg.BufferResourcePool, device *vkg.Device) error {
	execSize := len(s.exec.Bytes()) + 2*1024
	countSize := len(s.count.Bytes()) + 2*1024

	var err error

	s.execResource, err = rpool.AllocateBuffer(uint64(execSize), vk.BufferUsageStorageBufferBit)
	if err != nil {
		return err
	}

	s.countResource, err = rpool.AllocateBuffer(uint64(countSize), vk.BufferUsageStorageBufferBit)
	if err != nil {
		return err
	}

	s.descriptorSetLayout = &vkg.DescriptorSetLayout{}

	s.descriptorSetLayout.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vk.DescriptorTypeStorageBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageComputeBit),
	})

	s.descriptorSetLayout.AddBinding(vk.DescriptorSetLayoutBinding{
		Binding:         1,
		DescriptorType:  vk.DescriptorTypeStorageBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageComputeBit),
	})

	_, err = device.CreateDescriptorSetLayout(s.descriptorSetLayout)
	if err != nil {
		return err
	}

	s.descriptorSet, err = s.descriptorPool.Allocate(s.descriptorSetLayout)
	if err != nil {
		return err
	}
	s.descriptorSet.AddBuffer(0, vk.DescriptorTypeStorageBuffer, &s.execResource.Buffer, 0)
	s.descriptorSet.AddBuffer(1, vk.DescriptorTypeStorageBuffer, &s.countResource.Buffer, 0)
	s.descriptorSet.Write()

	return nil
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

	totalTime := time.Now()

	rm := ldevice.CreateResourceManager()

	bytesNeeded := 512 * 1024 * 1024

	rpool, err := rm.AllocateBufferPoolWithOptions("compute", uint64(bytesNeeded), vk.MemoryPropertyHostCoherentBit|vk.MemoryPropertyHostVisibleBit, vk.BufferUsageStorageBufferBit, vk.SharingModeExclusive)

	rpool.Memory.Map()

	sdf := NewSDFMarcher(Bounds{-5, -5, -5}, Bounds{5, 5, 5}, 0.05)

	err = sdf.Init(rpool, ldevice)
	orPanic(err)

	cpool, err := ldevice.CreateCommandPool(queues.FilterCompute()[0])
	orPanic(err)

	cb, err := cpool.AllocateBuffer(vk.CommandBufferLevelPrimary)
	orPanic(err)

	err = cb.BeginOneTime()
	orPanic(err)

	sdf.Process(cb)

	cb.End()

	fence, err := ldevice.CreateFence()
	orPanic(err)

	now := time.Now()
	computeQueue.SubmitWithFence(fence, cb)
	fmt.Printf("Compute time %v\n", time.Since(now))

	ldevice.WaitForFences(true, time.Duration(vk.MaxUint64), fence)

	cpool.FreeBuffer(cb)

	tc := sdf.GetTriangleCount()
	fmt.Printf("%d triangles\n", tc)
	//sdf.printCounts()

	tbs := (tc * 3 * 3 * 4) + (6 * 1024)
	//(3 verts * 3 points * sizeof(4))

	fmt.Printf("%d bytes for mesh %d (MB) \n", tbs, tbs/1024/1024)

	triangleResource, err := rpool.AllocateBuffer(uint64(tbs), vk.BufferUsageStorageBufferBit)
	orPanic(err)

	cb, err = cpool.AllocateBuffer(vk.CommandBufferLevelPrimary)
	orPanic(err)

	err = cb.BeginOneTime()
	orPanic(err)
	sdf.March(ldevice, triangleResource, cb)
	cb.End()

	fence, err = ldevice.CreateFence()
	orPanic(err)

	now = time.Now()
	computeQueue.SubmitWithFence(fence, cb)

	ldevice.WaitForFences(true, time.Duration(vk.MaxUint64), fence)
	fmt.Printf("Compute time %v\n", time.Since(now))
	//sdf.printCounts()

	/*
		data := toFloat32(triangleResource.Bytes())
		for i := 0; i <= int(tc); i++ {
			if i%9 == 0 {
				fmt.Printf("\n")
			}
			fmt.Printf("%f ", data[i])
		}*/
	fmt.Printf("Writing mesh %v\n", time.Since(totalTime))

	out, err := os.Create("out.stl")
	orPanic(err)
	data := make([]float32, tc*9)
	copy(data, toFloat32(triangleResource.Bytes()))
	fmt.Fprintf(out, "solid foo\n")
	for i := 0; i < int(tc); i += 1 {
		fmt.Fprintf(out, "facet normal 0.0 0.0 0.0\n")
		fmt.Fprintf(out, "\touter loop\n")

		r := 9

		fmt.Fprintf(out, "\t\tvertex %f %f %f\n", data[i*r+0], data[i*r+1], data[i*r+2])
		fmt.Fprintf(out, "\t\tvertex %f %f %f\n", data[i*r+3], data[i*r+4], data[i*r+5])
		fmt.Fprintf(out, "\t\tvertex %f %f %f\n", data[i*r+6], data[i*r+7], data[i*r+8])

		fmt.Fprintf(out, "\tendloop\n")
		fmt.Fprintf(out, "endfacet\n")
	}
	fmt.Fprintf(out, "endsolid foo\n")
	out.Close()

	cpool.FreeBuffer(cb)

}

func (s *SDFMarcher) printCounts() {
	copy(s.count.Bytes(), s.countResource.Bytes())
	for _, c := range toUint32(s.count.Bytes()) {
		fmt.Printf("%d ", c)
	}
	fmt.Printf("\n")
}

func toFloat32(d []byte) []float32 {
	const m = 0x7fffffff
	s := len(d) / int(unsafe.Sizeof(float32(1)))
	return (*[m]float32)(unsafe.Pointer(&d[0]))[:s]
}

func toUint32(d []byte) []uint32 {
	const m = 0x7fffffff
	s := len(d) / int(unsafe.Sizeof(uint32(1)))
	return (*[m]uint32)(unsafe.Pointer(&d[0]))[:s]
}
