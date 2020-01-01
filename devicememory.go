package vkg

import (
	"sync/atomic"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

// DeviceMemory maps to Vulkan DeviceMemory and can either be memory on the host or on the device
type DeviceMemory struct {
	Device         *Device
	VKDeviceMemory vk.DeviceMemory
	Size           uint64
	MapCount       int32
	Ptr            unsafe.Pointer
}

// IsMapped returns true if the device memory is currently mapped
func (d *DeviceMemory) IsMapped() bool {
	return atomic.LoadInt32(&d.MapCount) > 0
}

// Destroy destorys this memory
func (d *DeviceMemory) Destroy() {
	vk.FreeMemory(d.Device.VKDevice, d.VKDeviceMemory, nil)
}

// MapCopyUnmap will map this memory, copy the specified data to it and unmap
func (d *DeviceMemory) MapCopyUnmap(data []byte) error {
	pm, err := d.MapWithSize(len(data))
	if err != nil {
		return err
	}

	const m = 0x7fffffff
	outData := (*[m]byte)(pm)[:len(data)]

	copy(outData, data)

	d.Unmap()
	return nil
}

// MapWithOffset will map the memory with a certain size and offset
func (d *DeviceMemory) MapWithOffset(size uint64, offset uint64) (unsafe.Pointer, error) {
	var res unsafe.Pointer
	err := vk.Error(vk.MapMemory(d.Device.VKDevice, d.VKDeviceMemory, vk.DeviceSize(offset), vk.DeviceSize(size), 0, &res))
	if err != nil {
		return nil, err
	}
	atomic.AddInt32(&d.MapCount, 1)
	return res, nil
}

// Map will map the entirety of this memory
func (d *DeviceMemory) Map() (unsafe.Pointer, error) {
	var res unsafe.Pointer
	err := vk.Error(vk.MapMemory(d.Device.VKDevice, d.VKDeviceMemory, 0, vk.DeviceSize(d.Size), 0, &res))
	if err != nil {
		return nil, err
	}
	atomic.AddInt32(&d.MapCount, 1)
	d.Ptr = res
	return res, nil
}

// MapWithSize will map this memory starting at offset 0 with a particular size
func (d *DeviceMemory) MapWithSize(size int) (unsafe.Pointer, error) {
	var res unsafe.Pointer
	err := vk.Error(vk.MapMemory(d.Device.VKDevice, d.VKDeviceMemory, 0, vk.DeviceSize(size), 0, &res))
	if err != nil {
		return nil, err
	}
	atomic.AddInt32(&d.MapCount, 1)
	return res, nil
}

// Unmap this memory
func (d *DeviceMemory) Unmap() {
	d.Ptr = nil
	vk.UnmapMemory(d.Device.VKDevice, d.VKDeviceMemory)
	atomic.AddInt32(&d.MapCount, -1)
}
