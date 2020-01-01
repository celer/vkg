package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

// Buffer in vulkan are essentially a way of identifying and describing resources to the system. So for example
// you can have a buffer which holds vertex data in a specific format, or index data or a U.B.O (Uniform Buffer Object -
// which is data which is provided to a shader), or a buffer which is a source or destination for data from other locations.
// Buffers can reside either in the host memory or in the GPU (device) memory. The buffer itself is not the allocation of
// memory but simply a description of what the memory is and where it is. Device or host memory must still be allocated using
// the physical device object to actually hold the data associated with the buffer.
type Buffer struct {
	Device   *Device
	VKBuffer vk.Buffer
	Usage    vk.BufferUsageFlagBits
	Size     uint64
}

func usageToString(usage vk.BufferUsageFlagBits) string {
	str := ""
	if usage&vk.BufferUsageTransferSrcBit == vk.BufferUsageTransferSrcBit {
		str += "TransferSrc|"
	}
	if usage&vk.BufferUsageTransferDstBit == vk.BufferUsageTransferDstBit {
		str += "TransferDst|"
	}
	if usage&vk.BufferUsageUniformBufferBit == vk.BufferUsageUniformBufferBit {
		str += "UniformBuffer|"
	}
	if usage&vk.BufferUsageStorageBufferBit == vk.BufferUsageStorageBufferBit {
		str += "StorageBuffer|"
	}
	if usage&vk.BufferUsageVertexBufferBit == vk.BufferUsageVertexBufferBit {
		str += "VertexBuffer|"
	}
	if usage&vk.BufferUsageIndexBufferBit == vk.BufferUsageIndexBufferBit {
		str += "IndexBuffer|"
	}
	if len(str) > 0 {
		str = str[:len(str)-1]
	}

	return str
}

// CreateBufferWithOptions creates a buffer object of a certain size, for a certain usage, with certain sharing options. This buffer
// can be used to describe memory which is allocated on either the host or device.
func (d *Device) CreateBufferWithOptions(sizeInBytes uint64, usage vk.BufferUsageFlagBits, sharing vk.SharingMode) (*Buffer, error) {

	bufferCreateInfo := vk.BufferCreateInfo{
		SType:       vk.StructureTypeBufferCreateInfo,
		Size:        vk.DeviceSize(sizeInBytes),
		Usage:       vk.BufferUsageFlags(usage),
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
	ret.Usage = usage

	return &ret, nil

}

func (b *Buffer) String() string {
	return fmt.Sprintf("{type:buffer size:%d , usage: %s }", b.Size, usageToString(b.Usage))
}

// VKMemoryRequirements returns the vulkan native vk.MemoryRequirements objects which can be inspected to learn more about
// the memory requirements assocaited with this buffer. (You must call .Deref() on this object to populate it).
func (b *Buffer) VKMemoryRequirements() vk.MemoryRequirements {
	var memoryRequirements vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(b.Device.VKDevice, b.VKBuffer, &memoryRequirements)
	return memoryRequirements
}

// Bind associates a specific bit of device memory with this buffer, essentially identifying the memory as being used
// by this buffer. The offset is provied so that the buffer can be bound to a certain place in memory. There is a requirement
// that the buffer be bound in accordance with the alignment requirements returned by .VKMemoryRequirements().
func (b *Buffer) Bind(memory *DeviceMemory, offset uint64) error {
	return vk.Error((vk.BindBufferMemory(b.Device.VKDevice, b.VKBuffer, memory.VKDeviceMemory, vk.DeviceSize(offset))))
}

// Destroy the buffer
func (b *Buffer) Destroy() {
	if b.VKBuffer != vk.NullBuffer {
		vk.DestroyBuffer(b.Device.VKDevice, b.VKBuffer, nil)
		b.VKBuffer = vk.NullBuffer
	}
}
