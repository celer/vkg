package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

type Queue struct {
	Device      *Device
	QueueFamily *QueueFamily
	VKQueue     vk.Queue
}

func (q *Queue) WaitIdle() error {
	return vk.Error(vk.QueueWaitIdle(q.VKQueue))
}

func (q *Queue) SubmitWaitIdle(buffers ...*CommandBuffer) error {
	var submitInfo = vk.SubmitInfo{}
	submitInfo.SType = vk.StructureTypeSubmitInfo
	submitInfo.CommandBufferCount = uint32(len(buffers)) // submit a single command buffer

	b := make([]vk.CommandBuffer, len(buffers))
	for i, _ := range buffers {
		b[i] = buffers[i].VKCommandBuffer
	}

	submitInfo.PCommandBuffers = b // the command buffer to submit.

	err := vk.Error(vk.QueueSubmit(q.VKQueue, 1, []vk.SubmitInfo{submitInfo}, nil))
	if err != nil {
		return err
	}

	vk.QueueWaitIdle(q.VKQueue)

	return nil

}

func (q *Queue) SubmitWithFence(fence *Fence, buffers ...*CommandBuffer) error {
	var submitInfo = vk.SubmitInfo{}
	submitInfo.SType = vk.StructureTypeSubmitInfo
	submitInfo.CommandBufferCount = uint32(len(buffers)) // submit a single command buffer

	b := make([]vk.CommandBuffer, len(buffers))
	for i, _ := range buffers {
		b[i] = buffers[i].VKCommandBuffer
	}

	submitInfo.PCommandBuffers = b // the command buffer to submit.

	err := vk.Error(vk.QueueSubmit(q.VKQueue, 1, []vk.SubmitInfo{submitInfo}, fence.VKFence))
	if err != nil {
		return err
	}

	return nil

}

func (q *Queue) String() string {
	return fmt.Sprintf("{Device: %s QueueFamily: %s}", q.Device.String(), q.QueueFamily.String())
}
