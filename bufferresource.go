package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

// BufferResource is a buffer based resource, for example
// vertex buffer, index buffer, UBO,  which have been allocated
// from a larger pool of device memory. Vulkan limits the number of
// memory allocations that can be done by an application, so applications
// should manage their own pools of memory. A BufferResource is a buffer
// which has been managed by the ResourceManager.
type BufferResource struct {
	Buffer
	ResourcePool    *BufferResourcePool
	Allocation      *Allocation
	StagingResource *BufferResource
}

// VKMappedMemoryRange is provided so that the buffer implements MappedMemoryRange
// interface which can be used by device.FlushMappedRanges(...)
func (r *BufferResource) VKMappedMemoryRange() vk.MappedMemoryRange {
	return vk.MappedMemoryRange{
		SType:  vk.StructureTypeMappedMemoryRange,
		Memory: r.ResourcePool.Memory.VKDeviceMemory,
		Offset: vk.DeviceSize(r.Allocation.Offset),
		Size:   vk.DeviceSize(r.Allocation.Size),
	}
}

// RequiresStaging indicates that this particular buffer resource
// must be staged before it can be used. This is primarly
// indicative that the BufferResource is stored in device memory.
func (r *BufferResource) RequiresStaging() bool {
	return r.ResourcePool.NeedsStaging
}

func (r *BufferResource) String() string {
	return r.Buffer.String()
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
		r.StagingResource, err = stagingPool.AllocateBuffer(r.Buffer.Size, vk.BufferUsageTransferSrcBit)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("resource does not require staging")

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
func (r *BufferResource) Bytes() []byte {
	if r.RequiresStaging() {
		return nil
	}

	if r.ResourcePool.Memory.Ptr == nil {
		return nil
	}
	const m = 0x7fffffff
	s := r.Allocation.Offset
	e := r.Allocation.Offset + r.Allocation.Size

	data := (*[m]byte)(r.ResourcePool.Memory.Ptr)[s:e]

	return data
}

func (r *BufferResource) Destroy() {
	r.Free()
}

//Free this resource and it's associated resources
func (r *BufferResource) Free() {
	if r.StagingResource != nil {
		r.StagingResource.Free()
		r.StagingResource = nil
	}
	if r.Allocation != nil {
		r.ResourcePool.Allocator.Free(r.Allocation)
		r.Allocation = nil
	}
	if r.Buffer.VKBuffer != vk.NullBuffer {
		r.Buffer.Destroy()
	}
}
