package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

// Buffer are used to map hunks of data that are then bound to resources used by the pipeline
// and command buffers to render data.
type Buffer struct {
	Device   *Device
	VKBuffer vk.Buffer
	Size     uint64
}

func (d *Device) CreateBuffer(sizeInBytes uint64) (*Buffer, error) {
	return d.CreateBufferWithOptions(sizeInBytes, vk.BufferUsageFlags(vk.BufferUsageStorageBufferBit), vk.SharingModeExclusive)
}

func (d *Device) CreateBufferWithOptions(sizeInBytes uint64, usage vk.BufferUsageFlags, sharing vk.SharingMode) (*Buffer, error) {

	bufferCreateInfo := vk.BufferCreateInfo{
		SType:       vk.StructureTypeBufferCreateInfo,
		Size:        vk.DeviceSize(sizeInBytes),
		Usage:       usage,
		SharingMode: sharing,
	}

	var buffer vk.Buffer
	err := vk.Error(vk.CreateBuffer(d.VKDevice, &bufferCreateInfo, nil, &buffer))
	if err != nil {
		return nil, err
	}

	var ret Buffer
	ret.VKBuffer = buffer
	ret.Device = d
	ret.Size = sizeInBytes

	return &ret, nil

}

func (b *Buffer) VKMemoryRequirements() vk.MemoryRequirements {
	var memoryRequirements vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(b.Device.VKDevice, b.VKBuffer, &memoryRequirements)
	return memoryRequirements
}

func (b *Buffer) DSInfo(offset int) vk.DescriptorBufferInfo {
	var descriptorBufferInfo = vk.DescriptorBufferInfo{}
	descriptorBufferInfo.Buffer = b.VKBuffer
	descriptorBufferInfo.Offset = vk.DeviceSize(offset)
	descriptorBufferInfo.Range = vk.DeviceSize(b.Size)
	return descriptorBufferInfo
}

func (b *Buffer) AllocationRequirments() *AllocationRequirements {
	memoryRequirements := b.VKMemoryRequirements()
	mr := &memoryRequirements
	mr.Deref()

	return &AllocationRequirements{
		Size:           int(mr.Size),
		MemoryTypeBits: mr.MemoryTypeBits,
	}
}

func (b *Buffer) Bind(memory *DeviceMemory, offset uint64) error {
	return vk.Error((vk.BindBufferMemory(b.Device.VKDevice, b.VKBuffer, memory.VKDeviceMemory, vk.DeviceSize(offset))))
}

func (b *Buffer) Destroy() {
	vk.DestroyBuffer(b.Device.VKDevice, b.VKBuffer, nil)
}
