package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

// Device is a logical device per Vulkan terminology
type Device struct {
	PhysicalDevice *PhysicalDevice
	VKDevice       vk.Device
}

// Destroy destroys the device
func (d *Device) Destroy() {
	vk.DestroyDevice(d.VKDevice, nil)
}

// String is a stringer interface for the device
func (d *Device) String() string {
	return fmt.Sprintf("{ PhysicalDevice: %s }", d.PhysicalDevice)
}

// WaitIdle waits until the device is idle
func (d *Device) WaitIdle() error {
	return vk.Error(vk.DeviceWaitIdle(d.VKDevice))
}

// FlushMappedRanges will flush mapped memory ranges, it can take a BufferResource directly, as it implements the required interface
func (d *Device) FlushMappedRanges(r ...MappedMemoryRange) error {

	d.PhysicalDevice.VKPhysicalDeviceProperties.Limits.Deref()

	atomSize := d.PhysicalDevice.VKPhysicalDeviceProperties.Limits.NonCoherentAtomSize

	ranges := make([]vk.MappedMemoryRange, len(r))
	for i := range r {
		ranges[i] = r[i].VKMappedMemoryRange()

		// we need to make sure the range is a mltiple of atomSize
		m := (ranges[i].Size % atomSize)
		ranges[i].Size = ranges[i].Size - m + atomSize

	}

	return vk.Error(vk.FlushMappedMemoryRanges(d.VKDevice, uint32(len(ranges)), ranges))
}

// GetQueue gets a queue matching a specific queue family
func (d *Device) GetQueue(qf *QueueFamily) *Queue {

	var vkq vk.Queue

	vk.GetDeviceQueue(d.VKDevice, uint32(qf.Index), 0, &vkq)

	var queue Queue
	queue.QueueFamily = qf
	queue.Device = d
	queue.VKQueue = vkq

	return &queue
}

// Allocate allocates a certain amount and type of memory
func (d *Device) Allocate(sizeInBytes int, memoryTypeBits uint32, memoryProperties vk.MemoryPropertyFlagBits) (*DeviceMemory, error) {

	var allocateInfo = vk.MemoryAllocateInfo{}
	allocateInfo.SType = vk.StructureTypeMemoryAllocateInfo
	allocateInfo.AllocationSize = vk.DeviceSize(sizeInBytes)

	var err error

	allocateInfo.MemoryTypeIndex, err = d.PhysicalDevice.FindMemoryType(
		memoryTypeBits,
		memoryProperties)

	if err != nil {
		return nil, err
	}

	var deviceMemory vk.DeviceMemory

	err = vk.Error(vk.AllocateMemory(d.VKDevice, &allocateInfo, nil, &deviceMemory))
	if err != nil {
		return nil, err
	}

	var ret DeviceMemory

	ret.Size = uint64(sizeInBytes)
	ret.Device = d
	ret.VKDeviceMemory = deviceMemory

	return &ret, nil
}
