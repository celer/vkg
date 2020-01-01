package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

type ImageResource struct {
	Image
	ResourcePool    *ImageResourcePool
	Allocation      *Allocation
	StagingResource *BufferResource
	// Does this resource have it's own pool it is responsible for?
	IndividualPool bool
}

// NewImageResourceWithOptions will create a image resource which has it's own exclusive pool
func (r *ResourceManager) NewImageResourceWithOptions(extent vk.Extent2D, format vk.Format, tiling vk.ImageTiling, usage vk.ImageUsageFlagBits, sharing vk.SharingMode, mprops vk.MemoryPropertyFlagBits) (*ImageResource, error) {

	ir := &ImageResource{}

	img, err := r.Device.CreateImageWithOptions(extent, format, tiling, usage)
	if err != nil {
		return nil, err
	}

	mr := img.VKMemoryRequirements()
	mr.Deref()

	memory, err := r.Device.Allocate(int(mr.Size), mr.MemoryTypeBits, mprops)
	if err != nil {
		return nil, err
	}

	err = vk.Error(vk.BindImageMemory(r.Device.VKDevice, img.VKImage, memory.VKDeviceMemory, vk.DeviceSize(0)))
	if err != nil {
		return nil, err
	}

	pool := &ImageResourcePool{}
	pool.ResourceManager = r
	pool.Usage = usage
	pool.MemoryProperties = mprops
	pool.Sharing = sharing
	pool.Memory = memory

	ir.VKImage = img.VKImage
	ir.Device = img.Device
	ir.VKFormat = format
	ir.Extent = extent
	ir.ResourcePool = pool
	ir.IndividualPool = true

	return ir, nil

}

// RequiresStaging indicates that this particular buffer resource
// must be staged before it can be used
func (r *ImageResource) RequiresStaging() bool {
	return r.ResourcePool.NeedsStaging
}

// AllocateStagingResource will allocate an apporpriate resource
// which can be used for staging this resource. Once allocated
// it must be explicitly free'd. The staging resource is allocated
// from a resource pool called 'staging', which the program must create
func (r *ImageResource) AllocateStagingResource() error {
	if r.ResourcePool.NeedsStaging {
		stagingPool := r.ResourcePool.ResourceManager.GetStagingPool()
		if stagingPool == nil {
			return fmt.Errorf("failed to acquire pool with name 'staging' for staging resources, please insure it has been created")
		}
		var err error
		r.StagingResource, err = stagingPool.AllocateBuffer(r.Image.Size, vk.BufferUsageTransferSrcBit)
		if err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("resource does not require staging")
	}

}

// FreeStagingResource will free the staged resource associated with this resource
func (r *ImageResource) FreeStagingResource() {
	if r.StagingResource != nil {
		r.StagingResource.Free()
		r.StagingResource = nil
	}
}

// Bytes returns a byte slice representing the mapped memory, which can be
// read from or copied to
func (r *ImageResource) Bytes() ([]byte, error) {
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

func (r *ImageResource) String() string {
	return "image"
}

func (r *ImageResource) Destroy() {
	r.Free()
}

//Free this resource and it's associated resources
func (r *ImageResource) Free() {
	if r.StagingResource != nil {
		r.StagingResource.Free()
		r.StagingResource = nil
	}
	if r.IndividualPool && r.ResourcePool != nil {
		r.ResourcePool.Destroy()
		r.ResourcePool = nil
	} else if r.Allocation != nil {
		r.ResourcePool.Allocator.Free(r.Allocation)
		r.Allocation = nil
	}
	r.Image.Destroy()
}

func (cb *CommandBuffer) StageImageResource(img *ImageResource) error {
	if img.StagingResource == nil {
		return fmt.Errorf("no staging resource has been allocated")
	}
	vk.CmdCopyBufferToImage(cb.VK(), img.StagingResource.VKBuffer, img.VKImage, vk.ImageLayoutTransferDstOptimal, 1, []vk.BufferImageCopy{
		vk.BufferImageCopy{
			BufferOffset:      0,
			BufferRowLength:   0,
			BufferImageHeight: 0,
			ImageSubresource: vk.ImageSubresourceLayers{
				AspectMask:     vk.ImageAspectFlags(vk.ImageAspectColorBit),
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			ImageOffset: vk.Offset3D{},
			ImageExtent: vk.Extent3D{
				Width: uint32(img.Extent.Width), Height: uint32(img.Extent.Height), Depth: 1,
			},
		},
	})
	return nil
}

func (cb *CommandBuffer) TransitionImageLayout(img *ImageResource, format vk.Format, oldLayout, newLayout vk.ImageLayout) {
	var barrier = vk.ImageMemoryBarrier{}
	barrier.SType = vk.StructureTypeImageMemoryBarrier
	barrier.OldLayout = oldLayout
	barrier.NewLayout = newLayout
	barrier.SrcQueueFamilyIndex = vk.QueueFamilyIgnored
	barrier.DstQueueFamilyIndex = vk.QueueFamilyIgnored
	barrier.Image = img.VKImage
	barrier.SubresourceRange.AspectMask = vk.ImageAspectFlags(vk.ImageAspectColorBit)
	barrier.SubresourceRange.BaseMipLevel = 0
	barrier.SubresourceRange.LevelCount = 1
	barrier.SubresourceRange.BaseArrayLayer = 0
	barrier.SubresourceRange.LayerCount = 1
	barrier.SrcAccessMask = 0
	barrier.DstAccessMask = 0

	var sourceStage, destStage vk.PipelineStageFlags

	if oldLayout == vk.ImageLayoutUndefined && newLayout == vk.ImageLayoutTransferDstOptimal {
		barrier.SrcAccessMask = 0
		barrier.DstAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)

		sourceStage = vk.PipelineStageFlags(vk.PipelineStageTopOfPipeBit)
		destStage = vk.PipelineStageFlags(vk.PipelineStageTransferBit)

	} else if oldLayout == vk.ImageLayoutTransferDstOptimal && newLayout == vk.ImageLayoutShaderReadOnlyOptimal {
		barrier.SrcAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)
		barrier.DstAccessMask = vk.AccessFlags(vk.AccessShaderReadBit)

		sourceStage = vk.PipelineStageFlags(vk.PipelineStageTransferBit)
		destStage = vk.PipelineStageFlags(vk.PipelineStageFragmentShaderBit)
	}

	vk.CmdPipelineBarrier(cb.VK(), sourceStage, destStage, 0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{barrier})

}
