package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

type QueueFamilySlice []*QueueFamily

func (ql QueueFamilySlice) Filter(f func(q *QueueFamily) bool) QueueFamilySlice {
	ret := make([]*QueueFamily, 0)
	for _, q := range ql {
		if f(q) {
			ret = append(ret, q)
		}
	}
	return ret
}

func (ql QueueFamilySlice) FilterCompute() QueueFamilySlice {
	return ql.Filter(func(q *QueueFamily) bool {
		return q.IsCompute()
	})
}

func (ql QueueFamilySlice) FilterPresent(surface vk.Surface) QueueFamilySlice {
	return ql.Filter(func(q *QueueFamily) bool {
		return q.SupportsPresent(surface)
	})
}

func (ql QueueFamilySlice) FilterGraphicsAndPresent(surface vk.Surface) QueueFamilySlice {
	return ql.Filter(func(q *QueueFamily) bool {
		return q.IsGraphics() && q.SupportsPresent(surface)
	})
}

func (ql QueueFamilySlice) FilterGraphics() QueueFamilySlice {
	return ql.Filter(func(q *QueueFamily) bool {
		return q.IsGraphics()
	})
}

func (ql QueueFamilySlice) FilterTransfer() QueueFamilySlice {
	return ql.Filter(func(q *QueueFamily) bool {
		return q.IsTransfer()
	})
}

type QueueFamily struct {
	Index                   int
	PhysicalDevice          *PhysicalDevice
	VKQueueFamilyProperties vk.QueueFamilyProperties
}

func (q *QueueFamily) IsCompute() bool {
	return q.VKQueueFamilyProperties.QueueFlags&vk.QueueFlags(vk.QueueComputeBit) == vk.QueueFlags(vk.QueueComputeBit)
}

func (q *QueueFamily) IsGraphics() bool {
	return q.VKQueueFamilyProperties.QueueFlags&vk.QueueFlags(vk.QueueGraphicsBit) == vk.QueueFlags(vk.QueueGraphicsBit)

}

func (q *QueueFamily) IsTransfer() bool {
	return q.VKQueueFamilyProperties.QueueFlags&vk.QueueFlags(vk.QueueTransferBit) == vk.QueueFlags(vk.QueueTransferBit)
}

func (q *QueueFamily) SupportsPresent(surface vk.Surface) bool {
	var supportsPresent vk.Bool32
	vk.GetPhysicalDeviceSurfaceSupport(q.PhysicalDevice.VKPhysicalDevice, uint32(q.Index), surface, &supportsPresent)
	return supportsPresent == vk.True
}

func (q *QueueFamily) String() string {
	return fmt.Sprintf("{ Index: %d Compute: %v Graphics: %v Transfer: %v }", q.Index, q.IsCompute(), q.IsGraphics(), q.IsTransfer())
}
