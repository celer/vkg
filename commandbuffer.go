package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

// CommandBuffers describe a sequence of commands that will be executed
// upon being sent to a device queue. Not all available vulkan commands
// are wrapped by this package. It is expected that the calling application
// must call the native vulkan command APIs.
type CommandBuffer struct {
	VKCommandBuffer vk.CommandBuffer
}

// ResetAndRelease will reset this commandbuffer and release the associated resources
func (c *CommandBuffer) ResetAndRelease() error {
	return vk.Error(vk.ResetCommandBuffer(c.VKCommandBuffer, vk.CommandBufferResetFlags(vk.CommandBufferResetReleaseResourcesBit)))
}

// Reset this command buffer
func (c *CommandBuffer) Reset() error {
	return vk.Error(vk.ResetCommandBuffer(c.VKCommandBuffer, 0))
}

// VK is a utility function for accessing the native vulkan command buffer
func (c *CommandBuffer) VK() vk.CommandBuffer {
	return c.VKCommandBuffer
}

// Begin capturing work for this command buffer
func (c *CommandBuffer) BeginContinueRenderPass(renderpass vk.RenderPass, framebuffer vk.Framebuffer) error {
	var beginInfo = vk.CommandBufferBeginInfo{}
	beginInfo.SType = vk.StructureTypeCommandBufferBeginInfo
	beginInfo.Flags = vk.CommandBufferUsageFlags(vk.CommandBufferUsageRenderPassContinueBit)

	inheritInfo := vk.CommandBufferInheritanceInfo{}
	inheritInfo.SType = vk.StructureTypeCommandBufferInheritanceInfo
	inheritInfo.Framebuffer = framebuffer
	inheritInfo.RenderPass = renderpass

	beginInfo.PInheritanceInfo = []vk.CommandBufferInheritanceInfo{inheritInfo}

	return vk.Error(vk.BeginCommandBuffer(c.VKCommandBuffer, &beginInfo))

}

// Begin capturing work for this command buffer
func (c *CommandBuffer) Begin() error {
	var beginInfo = vk.CommandBufferBeginInfo{}
	beginInfo.SType = vk.StructureTypeCommandBufferBeginInfo
	beginInfo.Flags = 0
	return vk.Error(vk.BeginCommandBuffer(c.VKCommandBuffer, &beginInfo))

}

// BeginOneTime begins capturing work for this command buffer, with the stipulation that it will only be used once (instead of put back in the pool of command buffers)
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

// End describing work for this command buffer
func (c *CommandBuffer) End() error {
	return vk.Error(vk.EndCommandBuffer(c.VKCommandBuffer))
}
