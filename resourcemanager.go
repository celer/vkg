package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

const StagingPoolName = "staging"

var insufficientPoolSpaceError = fmt.Errorf("insufficient storage space in resource pool")

//BufferResource is a buffer based resource, for example
// vertex buffer, index buffer, ubo.
type BufferResource struct {
	Buffer
	ResourcePool    *ResourcePool
	Allocation      *Allocation
	StagingResource *BufferResource
}

// RequiresStaging indicates that this particular buffer resource
// must be staged before it can be used
func (r *BufferResource) RequiresStaging() bool {
	return r.ResourcePool.NeedsStaging
}

// AllocateStagingResource will allocate an apporpriate resource
// which can be used for staging this resource. Once allocated
// it must be explicitly free'd. The staging resource is allocated
// from a resource pool called 'staging', which the program must create
func (r *BufferResource) AllocateStagingResource() error {
	if r.ResourcePool.NeedsStaging {
		stagingPool := r.ResourcePool.ResourceManager.GetStagingPool()
		if stagingPool == nil {
			return fmt.Errorf("failed to acquire pool with name 'staging' for staging resources, please insure it has been created")
		}
		var err error
		r.StagingResource, err = stagingPool.AllocateBuffer(r.Buffer.Size, vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit))
		if err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("resource does not require staging")
	}

}

// FreeStagingResource will free the staged resource associated with this resource
func (r *BufferResource) FreeStagingResource() {
	if r.StagingResource != nil {
		r.StagingResource.Free()
	}
}

// CmdCopyBufferFromStagedResource will populate this buffer from the previously
// allocated staged resource
func (c *CommandBuffer) CmdCopyBufferFromStagedResource(resource *BufferResource) {
	vk.CmdCopyBuffer(c.VK(), resource.StagingResource.Buffer.VKBuffer, resource.Buffer.VKBuffer, 1, []vk.BufferCopy{
		vk.BufferCopy{
			SrcOffset: 0,
			DstOffset: vk.DeviceSize(resource.Allocation.Offset),
			Size:      vk.DeviceSize(resource.Allocation.Size),
		},
	})
}

// Bytes returns a byte slice representing the mapped memory, which can be
// read from or copied to
func (r *BufferResource) Bytes() ([]byte, error) {
	if r.RequiresStaging() {
		return nil, fmt.Errorf("resource requires staging")
	}

	if r.ResourcePool.Memory.Ptr == nil {
		return nil, fmt.Errorf("memory in resource pool must be mapped first")
	}
	const m = 0x7fffffff
	s := r.Allocation.Offset
	e := r.Allocation.Offset + r.Allocation.Size

	data := (*[m]byte)(r.ResourcePool.Memory.Ptr)[s:e]

	return data, nil
}

//Free this resource and it's associated resources
func (r *BufferResource) Free() {
	if r.StagingResource != nil {
		r.StagingResource.Free()
	}
	r.ResourcePool.Allocator.Free(r.Allocation)
	r.Buffer.Destroy()
}

type ResourcePool struct {
	Device           *Device
	Name             string
	Usage            vk.BufferUsageFlags
	Sharing          vk.SharingMode
	MemoryProperties vk.MemoryPropertyFlags
	Size             uint64
	Allocator        IAllocator
	Memory           *DeviceMemory
	NeedsStaging     bool
	ResourceManager  *ResourceManager
}

func (p *ResourcePool) AllocateBuffer(size uint64, usage vk.BufferUsageFlags) (*BufferResource, error) {

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

	return ret, nil
}

func (p *ResourcePool) Destroy() {
	p.Memory.Destroy()
}

type ResourceManager struct {
	Device *Device
	Pools  map[string]*ResourcePool
}

func (d *Device) CreateResourceManager() *ResourceManager {
	return &ResourceManager{Device: d, Pools: make(map[string]*ResourcePool)}
}

func (r *ResourceManager) GetStagingPool() *ResourcePool {
	return r.Pools[StagingPoolName]
}

func (r *ResourceManager) AllocatePoolWithOptions(name string, size uint64, mprops vk.MemoryPropertyFlags, usage vk.BufferUsageFlags, sharing vk.SharingMode) (*ResourcePool, error) {
	needsStaging := false

	//FIXME this could be smarter about detecting integrated devies to really see if staging is needed
	if vk.MemoryPropertyFlagBits(mprops)&vk.MemoryPropertyDeviceLocalBit == vk.MemoryPropertyDeviceLocalBit {
		needsStaging = true
	}

	a := &LinearAllocator{Size: size}

	p := &ResourcePool{
		Device:           r.Device,
		Name:             name,
		Usage:            usage,
		Sharing:          sharing,
		MemoryProperties: mprops,
		Size:             size,
		Allocator:        a,
		NeedsStaging:     needsStaging,
	}

	if needsStaging {
		usage |= vk.BufferUsageFlags(vk.BufferUsageTransferDstBit)
	}

	buffer, err := r.Device.CreateBufferWithOptions(size, usage, sharing)
	if err != nil {
		return nil, err
	}
	defer buffer.Destroy()
	memory, err := r.Device.AllocateForBuffer(buffer, mprops)
	if err != nil {
		return nil, err
	}
	p.Memory = memory

	r.Pools[name] = p

	return p, nil

}
