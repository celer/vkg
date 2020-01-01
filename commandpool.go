package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

// CommandPool for managing a pool of command buffers
type CommandPool struct {
	Device        *Device
	QueueFamily   *QueueFamily
	VKCommandPool vk.CommandPool
}

// CreateCommandPool given a specific queue family
func (d *Device) CreateCommandPool(q *QueueFamily) (*CommandPool, error) {
	var commandPoolCreateInfo = vk.CommandPoolCreateInfo{}
	commandPoolCreateInfo.SType = vk.StructureTypeCommandPoolCreateInfo
	commandPoolCreateInfo.Flags = vk.CommandPoolCreateFlags(vk.CommandPoolCreateResetCommandBufferBit | vk.CommandPoolCreateTransientBit)
	commandPoolCreateInfo.QueueFamilyIndex = uint32(q.Index)

	var commandPool vk.CommandPool

	err := vk.Error(vk.CreateCommandPool(d.VKDevice, &commandPoolCreateInfo, nil, &commandPool))

	if err != nil {
		return nil, err
	}

	var ret CommandPool
	ret.Device = d
	ret.QueueFamily = q
	ret.VKCommandPool = commandPool

	return &ret, nil

}

// AllocateBuffers allocates some number of command buffers
func (c *CommandPool) AllocateBuffers(count int, level vk.CommandBufferLevel) ([]*CommandBuffer, error) {

	var commandBufferAllocateInfo = vk.CommandBufferAllocateInfo{}
	commandBufferAllocateInfo.SType = vk.StructureTypeCommandBufferAllocateInfo
	commandBufferAllocateInfo.CommandPool = c.VKCommandPool
	commandBufferAllocateInfo.Level = level
	commandBufferAllocateInfo.CommandBufferCount = uint32(count)

	cmdBuffers := make([]vk.CommandBuffer, count)

	err := vk.Error(vk.AllocateCommandBuffers(c.Device.VKDevice, &commandBufferAllocateInfo, cmdBuffers))
	if err != nil {
		return nil, err
	}

	ret := make([]*CommandBuffer, count)
	for i := range ret {
		ret[i] = &CommandBuffer{}
		ret[i].VKCommandBuffer = cmdBuffers[i]
	}

	return ret, nil

}

// AllocateBuffer allocates a single command buffer
func (c *CommandPool) AllocateBuffer(level vk.CommandBufferLevel) (*CommandBuffer, error) {
	ret, err := c.AllocateBuffers(1, level)
	if err != nil {
		return nil, err
	}
	return ret[0], nil

}

// FreeBuffers frees a set of command buffers
func (c *CommandPool) FreeBuffers(bs []*CommandBuffer) {
	b := make([]vk.CommandBuffer, len(bs))
	for i := range bs {
		b[i] = bs[i].VKCommandBuffer
	}
	vk.FreeCommandBuffers(c.Device.VKDevice, c.VKCommandPool, uint32(len(bs)), b)
}

// FreeBuffer frees a single buffer
func (c *CommandPool) FreeBuffer(b *CommandBuffer) {
	vk.FreeCommandBuffers(c.Device.VKDevice, c.VKCommandPool, 1, []vk.CommandBuffer{b.VKCommandBuffer})
}

// Destroy this command pool
func (c *CommandPool) Destroy() {
	vk.DestroyCommandPool(c.Device.VKDevice, c.VKCommandPool, nil)
}
