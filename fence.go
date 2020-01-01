package vkg

import (
	vk "github.com/vulkan-go/vulkan"
	"time"
)

type Fence struct {
	Device  *Device
	VKFence vk.Fence
}

func (d *Device) VKGetFenceStatus(f vk.Fence) vk.Result {
	return vk.GetFenceStatus(d.VKDevice, f)
}

func (d *Device) VKDestroyFence(f vk.Fence) {
	vk.DestroyFence(d.VKDevice, f, nil)
}

func (d *Device) VKCreateFence(signaled bool) (vk.Fence, error) {
	var fence vk.Fence
	var fenceCreateInfo = vk.FenceCreateInfo{}
	fenceCreateInfo.SType = vk.StructureTypeFenceCreateInfo
	if signaled {
		fenceCreateInfo.Flags = vk.FenceCreateFlags(vk.FenceCreateSignaledBit)
	} else {
		fenceCreateInfo.Flags = 0

	}
	err := vk.Error(vk.CreateFence(d.VKDevice, &fenceCreateInfo, nil, &fence))
	if err != nil {
		return nil, err
	}
	return fence, nil
}

func (d *Device) CreateFence() (*Fence, error) {

	fence, err := d.VKCreateFence(false)
	if err != nil {
		return nil, err
	}

	var ret Fence
	ret.VKFence = fence
	ret.Device = d
	return &ret, nil

}

func (d *Device) WaitForFences(waitForAll bool, ts time.Duration, fences ...*Fence) error {

	f := make([]vk.Fence, len(fences))
	for i, _ := range fences {
		f[i] = fences[i].VKFence
	}

	var wait vk.Bool32
	if waitForAll {
		wait = vk.True
	} else {
		wait = vk.False
	}

	err := vk.Error(vk.WaitForFences(d.VKDevice, uint32(len(fences)), f, wait, uint64(ts.Nanoseconds())))

	if err != nil {
		return err
	}

	return nil

}

func (f *Fence) Destroy() {
	vk.DestroyFence(f.Device.VKDevice, f.VKFence, nil)
}
