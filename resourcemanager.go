package vkg

import (
	"fmt"
	"log"

	vk "github.com/vulkan-go/vulkan"
)

const StagingPoolName = "staging"

var insufficientPoolSpaceError = fmt.Errorf("insufficient storage space in resource pool")

type ImageResourcePool struct {
	Device           *Device
	Name             string
	Usage            vk.ImageUsageFlagBits
	Sharing          vk.SharingMode
	MemoryProperties vk.MemoryPropertyFlagBits
	Size             uint64
	Allocator        IAllocator
	Memory           *DeviceMemory
	NeedsStaging     bool
	ResourceManager  *ResourceManager
}

type BufferResourcePool struct {
	Device           *Device
	Name             string
	Usage            vk.BufferUsageFlagBits
	Sharing          vk.SharingMode
	MemoryProperties vk.MemoryPropertyFlagBits
	Size             uint64
	Allocator        IAllocator
	Memory           *DeviceMemory
	NeedsStaging     bool
	ResourceManager  *ResourceManager
}

func (p *ImageResourcePool) AllocateImage(extent vk.Extent2D, format vk.Format, tiling vk.ImageTiling, usage vk.ImageUsageFlagBits) (*ImageResource, error) {
	i, err := p.Device.CreateImageWithOptions(extent, format, tiling, usage)
	if err != nil {
		return nil, err
	}

	mr := i.VKMemoryRequirements()

	mr.Deref()

	allocation := p.Allocator.Allocate(uint64(mr.Size), uint64(mr.Alignment))
	if allocation == nil {
		return nil, insufficientPoolSpaceError
	}

	err = vk.Error(vk.BindImageMemory(p.Device.VKDevice, i.VKImage, p.Memory.VKDeviceMemory, vk.DeviceSize(allocation.Offset)))
	if err != nil {
		return nil, err
	}

	img := &ImageResource{}
	img.VKImage = i.VKImage
	img.Device = i.Device
	img.VKFormat = i.VKFormat
	img.Size = uint64(mr.Size)
	img.Allocation = allocation
	img.ResourcePool = p
	img.Extent = extent

	allocation.Object = img

	return img, nil
}

func (p *ImageResourcePool) LogDetails() {
	log.Printf("Size: %d", p.Size)
	p.Allocator.LogDetails()
}

func (p *ImageResourcePool) Destroy() {
	if p.Allocator != nil {
		p.Allocator.DestroyContents()
		p.Allocator = nil
	}
	if p.Memory != nil {
		p.Memory.Destroy()
		p.Memory = nil
	}
	delete(p.ResourceManager.bufferPools, p.Name)
}

func (p *BufferResourcePool) AllocateFor(src ByteSourcer) (*BufferResource, error) {
	if vertex, ok := src.(VertexSourcer); ok {
		return p.AllocateBuffer(uint64(len(vertex.Bytes())), vk.BufferUsageVertexBufferBit)
	} else if index, ok := src.(IndexSourcer); ok {
		return p.AllocateBuffer(uint64(len(index.Bytes())), vk.BufferUsageIndexBufferBit)
	} else {
		return nil, fmt.Errorf("unknown buffer object type")
	}

}

func (p *BufferResourcePool) AllocateBuffer(size uint64, usage vk.BufferUsageFlagBits) (*BufferResource, error) {

	buffer, err := p.Device.CreateBufferWithOptions(size, usage, vk.SharingModeExclusive)
	if err != nil {
		return nil, err
	}

	mr := buffer.VKMemoryRequirements()
	mr.Deref()

	allocation := p.Allocator.Allocate(size, uint64(mr.Alignment))
	if allocation == nil {
		return nil, insufficientPoolSpaceError
	}

	buffer.Bind(p.Memory, allocation.Offset)

	ret := &BufferResource{
		Allocation:   allocation,
		ResourcePool: p,
	}

	ret.VKBuffer = buffer.VKBuffer
	ret.Device = buffer.Device
	ret.Size = buffer.Size
	ret.Usage = usage

	allocation.Object = ret

	return ret, nil
}

func (p *BufferResourcePool) LogDetails() {
	log.Printf("Size: %d, Usage: %s", p.Size, usageToString(p.Usage))
	p.Allocator.LogDetails()
}

func (p *BufferResourcePool) Destroy() {
	if p.Allocator != nil {
		p.Allocator.DestroyContents()
		p.Allocator = nil
	}
	if p.Memory != nil {
		p.Memory.Destroy()
		p.Memory = nil
	}
	delete(p.ResourceManager.bufferPools, p.Name)
}

type ResourceManager struct {
	Device      *Device
	bufferPools map[string]*BufferResourcePool
	imagePools  map[string]*ImageResourcePool
}

func (d *Device) CreateResourceManager() *ResourceManager {
	return &ResourceManager{Device: d, bufferPools: make(map[string]*BufferResourcePool), imagePools: make(map[string]*ImageResourcePool)}
}

func (r *ResourceManager) GetStagingPool() *BufferResourcePool {
	return r.bufferPools[StagingPoolName]
}

func (r *ResourceManager) AllocateDeviceTexturePool(name string, size uint64) (*ImageResourcePool, error) {
	return r.AllocateImagePoolWithOptions(name, size, vk.MemoryPropertyDeviceLocalBit, vk.ImageUsageTransferDstBit|vk.ImageUsageSampledBit, vk.SharingModeExclusive)
}

func (r *ResourceManager) AllocateImagePoolWithOptions(name string, size uint64, mprops vk.MemoryPropertyFlagBits, usage vk.ImageUsageFlagBits, sharing vk.SharingMode) (*ImageResourcePool, error) {
	needsStaging := false

	//FIXME this could be smarter about detecting integrated devies to really see if staging is needed
	if vk.MemoryPropertyFlagBits(mprops)&vk.MemoryPropertyDeviceLocalBit == vk.MemoryPropertyDeviceLocalBit {
		needsStaging = true
	}

	a := &LinearAllocator{Size: size}

	p := &ImageResourcePool{
		Device:           r.Device,
		Name:             name,
		Usage:            usage,
		Sharing:          sharing,
		MemoryProperties: mprops,
		Size:             size,
		Allocator:        a,
		NeedsStaging:     needsStaging,
		ResourceManager:  r,
	}

	if needsStaging {
		usage |= vk.ImageUsageTransferDstBit
	}

	buffer, err := r.Device.CreateImageWithOptions(vk.Extent2D{Width: 800, Height: 600}, vk.FormatR8g8b8a8Uint, vk.ImageTilingOptimal, usage)
	if err != nil {
		return nil, err
	}
	defer buffer.Destroy()

	mr := buffer.VKMemoryRequirements()
	mr.Deref()

	memory, err := r.Device.Allocate(int(size), mr.MemoryTypeBits, mprops)
	if err != nil {
		return nil, err
	}
	p.Memory = memory

	r.imagePools[name] = p

	return p, nil

}

func (r *ResourceManager) Destroy() {
	for _, p := range r.bufferPools {
		p.Destroy()
	}
	for _, p := range r.imagePools {
		p.Destroy()
	}
}

func (r *ResourceManager) HasStagingPool() bool {
	return r.bufferPools[StagingPoolName] != nil
}

func (r *ResourceManager) AllocateStagingPool(size uint64) (*BufferResourcePool, error) {
	return r.AllocateBufferPoolWithOptions(StagingPoolName, size, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit, vk.BufferUsageTransferSrcBit, vk.SharingModeExclusive)
}

func (r *ResourceManager) AllocateHostVertexAndIndexBufferPool(name string, size uint64) (*BufferResourcePool, error) {
	return r.AllocateBufferPoolWithOptions(name, size, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit, vk.BufferUsageVertexBufferBit|vk.BufferUsageIndexBufferBit, vk.SharingModeExclusive)
}

func (r *ResourceManager) AllocateBufferPoolWithOptions(name string, size uint64, mprops vk.MemoryPropertyFlagBits, usage vk.BufferUsageFlagBits, sharing vk.SharingMode) (*BufferResourcePool, error) {
	needsStaging := false

	//FIXME this could be smarter about detecting integrated devies to really see if staging is needed
	if vk.MemoryPropertyFlagBits(mprops)&vk.MemoryPropertyDeviceLocalBit == vk.MemoryPropertyDeviceLocalBit {
		needsStaging = true
	}

	a := &LinearAllocator{Size: size}

	p := &BufferResourcePool{
		Device:           r.Device,
		Name:             name,
		Usage:            usage,
		Sharing:          sharing,
		MemoryProperties: mprops,
		Size:             size,
		Allocator:        a,
		NeedsStaging:     needsStaging,
		ResourceManager:  r,
	}

	if needsStaging {
		usage |= vk.BufferUsageTransferDstBit
	}

	buffer, err := r.Device.CreateBufferWithOptions(size, usage, sharing)
	if err != nil {
		return nil, err
	}
	defer buffer.Destroy()

	mr := buffer.VKMemoryRequirements()
	mr.Deref()

	memory, err := r.Device.Allocate(int(size), mr.MemoryTypeBits, mprops)
	if err != nil {
		return nil, err
	}
	p.Memory = memory

	r.bufferPools[name] = p

	return p, nil
}

func (r *ResourceManager) LogDetails() {
	for name, pool := range r.bufferPools {
		log.Printf("Buffer Pool: %s", name)
		pool.LogDetails()
	}
	for name, pool := range r.imagePools {
		log.Printf("Image Pool: %s", name)
		pool.LogDetails()
	}
}

func (r *ResourceManager) ImagePool(name string) *ImageResourcePool {
	return r.imagePools[name]
}

func (r *ResourceManager) BufferPool(name string) *BufferResourcePool {
	return r.bufferPools[name]
}
