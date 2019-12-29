package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type CommandPool struct {
	Device        *Device
	QueueFamily   *QueueFamily
	VKCommandPool vk.CommandPool
}

func (c *CommandPool) Destroy() {
	vk.DestroyCommandPool(c.Device.VKDevice, c.VKCommandPool, nil)
}

func (c *CommandPool) AllocateBuffers(count int) ([]*CommandBuffer, error) {

	var commandBufferAllocateInfo = vk.CommandBufferAllocateInfo{}
	commandBufferAllocateInfo.SType = vk.StructureTypeCommandBufferAllocateInfo
	commandBufferAllocateInfo.CommandPool = c.VKCommandPool
	commandBufferAllocateInfo.Level = vk.CommandBufferLevelPrimary
	commandBufferAllocateInfo.CommandBufferCount = uint32(count)

	cmdBuffers := make([]vk.CommandBuffer, count)

	err := vk.Error(vk.AllocateCommandBuffers(c.Device.VKDevice, &commandBufferAllocateInfo, cmdBuffers))
	if err != nil {
		return nil, err
	}

	ret := make([]*CommandBuffer, count)
	for i, _ := range ret {
		ret[i] = &CommandBuffer{}
		ret[i].VKCommandBuffer = cmdBuffers[i]
	}

	return ret, nil

}

func (c *CommandPool) AllocateBuffer() (*CommandBuffer, error) {
	ret, err := c.AllocateBuffers(1)
	if err != nil {
		return nil, err
	}
	return ret[0], nil

}

func (c *CommandPool) FreeBuffers(bs []*CommandBuffer) {
	b := make([]vk.CommandBuffer, len(bs))
	for i, _ := range bs {
		b[i] = bs[i].VKCommandBuffer
	}
	vk.FreeCommandBuffers(c.Device.VKDevice, c.VKCommandPool, uint32(len(bs)), b)
}

func (c *CommandPool) FreeBuffer(b *CommandBuffer) {
	vk.FreeCommandBuffers(c.Device.VKDevice, c.VKCommandPool, 1, []vk.CommandBuffer{b.VKCommandBuffer})
}

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
