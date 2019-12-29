package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type CommandBuffer struct {
	VKCommandBuffer vk.CommandBuffer
}

func (c *CommandBuffer) ResetAndRelease() error {
	return vk.Error(vk.ResetCommandBuffer(c.VKCommandBuffer, vk.CommandBufferResetFlags(vk.CommandBufferResetReleaseResourcesBit)))
}

func (c *CommandBuffer) Reset() error {
	return vk.Error(vk.ResetCommandBuffer(c.VKCommandBuffer, 0))
}

func (c *CommandBuffer) VK() vk.CommandBuffer {
	return c.VKCommandBuffer
}

func (c *CommandBuffer) Begin() error {
	var beginInfo = vk.CommandBufferBeginInfo{}
	beginInfo.SType = vk.StructureTypeCommandBufferBeginInfo
	beginInfo.Flags = 0
	return vk.Error(vk.BeginCommandBuffer(c.VKCommandBuffer, &beginInfo))

}

func (c *CommandBuffer) BeginOneTime() error {
	var beginInfo = vk.CommandBufferBeginInfo{}
	beginInfo.SType = vk.StructureTypeCommandBufferBeginInfo
	beginInfo.Flags = vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit)
	return vk.Error(vk.BeginCommandBuffer(c.VKCommandBuffer, &beginInfo))

}

func (c *CommandBuffer) CmdBindComputePipeline(p *ComputePipeline) {
	vk.CmdBindPipeline(c.VKCommandBuffer, vk.PipelineBindPointCompute, p.VKPipeline)
}

func (c *CommandBuffer) CmdBindDescriptorSets(bindPoint vk.PipelineBindPoint, layout *PipelineLayout, firstSet int, descriptorSets ...*DescriptorSet) {

	sets := make([]vk.DescriptorSet, len(descriptorSets))
	for i, _ := range descriptorSets {
		sets[i] = descriptorSets[i].VKDescriptorSet
	}

	vk.CmdBindDescriptorSets(c.VKCommandBuffer, bindPoint,
		layout.VKPipelineLayout, uint32(firstSet), uint32(len(descriptorSets)), sets, 0, nil)

}

func (c *CommandBuffer) CmdDispatch(x, y, z int) {
	vk.CmdDispatch(c.VKCommandBuffer, uint32(x), uint32(y), uint32(z))
}

func (c *CommandBuffer) End() error {
	return vk.Error(vk.EndCommandBuffer(c.VKCommandBuffer))
}
