package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

type Device struct {
	PhysicalDevice *PhysicalDevice
	VKDevice       vk.Device
}

func (d *Device) Destroy() {
	vk.DestroyDevice(d.VKDevice, nil)
}

func (d *Device) String() string {
	return fmt.Sprintf("{ PhysicalDevice: %s }", d.PhysicalDevice)
}

func (d *Device) WaitIdle() {
	vk.DeviceWaitIdle(d.VKDevice)
}

func (d *Device) GetQueue(qf *QueueFamily) *Queue {

	var vkq vk.Queue

	vk.GetDeviceQueue(d.VKDevice, uint32(qf.Index), 0, &vkq)

	var queue Queue
	queue.QueueFamily = qf
	queue.Device = d
	queue.VKQueue = vkq

	return &queue
}

type AllocationRequirements struct {
	Size           int
	MemoryTypeBits uint32
}

func (d *Device) AllocateForBuffer(b *Buffer, memoryProperties vk.MemoryPropertyFlags) (*DeviceMemory, error) {
	ar := b.AllocationRequirments()
	mem, err := d.Allocate(ar.Size, ar.MemoryTypeBits, memoryProperties)
	if err != nil {
		return nil, err
	}
	return mem, err
}

func (d *Device) Allocate(sizeInBytes int, memoryTypeBits uint32, memoryProperties vk.MemoryPropertyFlags) (*DeviceMemory, error) {

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
