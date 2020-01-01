package vkg

import (
	vk "github.com/vulkan-go/vulkan"
	"unsafe"
)

var end = "\x00"
var endChar byte = '\x00'

//DestroyAny is a utility function which given an item will try to
// figure out how to destroy it
func (d *Device) DestroyAny(i interface{}) {

	if t, ok := i.(vk.ImageView); ok {
		vk.DestroyImageView(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.Sampler); ok {
		vk.DestroySampler(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.DescriptorPool); ok {
		vk.DestroyDescriptorPool(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.Buffer); ok {
		vk.DestroyBuffer(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.Image); ok {
		vk.DestroyImage(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.Pipeline); ok {
		vk.DestroyPipeline(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.PipelineCache); ok {
		vk.DestroyPipelineCache(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.Fence); ok {
		vk.DestroyFence(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.RenderPass); ok {
		vk.DestroyRenderPass(d.VKDevice, t, nil)
	} else if t, ok := i.(vk.Semaphore); ok {
		vk.DestroySemaphore(d.VKDevice, t, nil)
	} else if t, ok := i.(IDestructable); ok {
		t.Destroy()
	}

}

// ToBytes will take an unsafe.Pointer and length in bytes and convert it
// to a byte slice
func ToBytes(ptr unsafe.Pointer, lenInBytes int) []byte {
	const m = 0x7fffffff
	return (*[m]byte)(ptr)[:lenInBytes]
}

func safeString(s string) string {
	if len(s) == 0 {
		return end
	}
	if s[len(s)-1] != endChar {
		return s + end
	}
	return s
}

func safeStrings(list []string) []string {
	for i := range list {
		list[i] = safeString(list[i])
	}
	return list
}
