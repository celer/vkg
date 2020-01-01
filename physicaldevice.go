package vkg

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

type VKPresentModes []vk.PresentMode

func (v VKPresentModes) Filter(f vk.PresentMode) VKPresentModes {
	ret := make(VKPresentModes, 0)
	for _, s := range v {
		if f == s {
			ret = append(ret, s)
		}
	}
	return ret
}

type VKSurfaceFormats []vk.SurfaceFormat

func (v VKSurfaceFormats) Filter(f func(f vk.SurfaceFormat) bool) VKSurfaceFormats {
	ret := make(VKSurfaceFormats, 0)
	for _, s := range v {
		s.Deref()
		if f(s) {
			ret = append(ret, s)
		}
	}
	return ret
}

type PhysicalDevice struct {
	DeviceName                 string
	VKPhysicalDevice           vk.PhysicalDevice
	VKPhysicalDeviceProperties vk.PhysicalDeviceProperties
}

func (p *PhysicalDevice) GetSurfacePresentModes(surface vk.Surface) (VKPresentModes, error) {
	var count uint32
	err := vk.Error(vk.GetPhysicalDeviceSurfacePresentModes(p.VKPhysicalDevice, surface, &count, nil))
	if err != nil {
		return nil, err
	}

	f := make([]vk.PresentMode, count)
	err = vk.Error(vk.GetPhysicalDeviceSurfacePresentModes(p.VKPhysicalDevice, surface, &count, f))
	if err != nil {
		return nil, err
	}

	return f, nil

}

func (p *PhysicalDevice) GetSurfaceFormats(surface vk.Surface) (VKSurfaceFormats, error) {
	var count uint32
	err := vk.Error(vk.GetPhysicalDeviceSurfaceFormats(p.VKPhysicalDevice, surface, &count, nil))
	if err != nil {
		return nil, err
	}

	f := make([]vk.SurfaceFormat, count)
	err = vk.Error(vk.GetPhysicalDeviceSurfaceFormats(p.VKPhysicalDevice, surface, &count, f))
	if err != nil {
		return nil, err
	}

	return f, nil

}

func (p *PhysicalDevice) GetSurfaceCapabilities(surface vk.Surface) (*vk.SurfaceCapabilities, error) {
	var caps vk.SurfaceCapabilities
	err := vk.Error(vk.GetPhysicalDeviceSurfaceCapabilities(p.VKPhysicalDevice, surface, &caps))
	if err != nil {
		return nil, err
	}

	return &caps, err
}

func (p *PhysicalDevice) String() string {
	return p.DeviceName
}

func (p *PhysicalDevice) QueueFamilies() (QueueFamilySlice, error) {
	var queueFamilyCount uint32

	vk.GetPhysicalDeviceQueueFamilyProperties(p.VKPhysicalDevice, &queueFamilyCount, nil)

	if queueFamilyCount == 0 {
		return nil, nil
	}

	queues := make([]vk.QueueFamilyProperties, queueFamilyCount)

	vk.GetPhysicalDeviceQueueFamilyProperties(p.VKPhysicalDevice, &queueFamilyCount, queues)

	ret := make([]*QueueFamily, queueFamilyCount)
	for i, queue := range queues {

		ret[i] = &QueueFamily{Index: i, PhysicalDevice: p, VKQueueFamilyProperties: queue}

		ret[i].VKQueueFamilyProperties.Deref()

	}

	return ret, nil

}

type CreateDeviceOptions struct {
	EnabledExtensions []string
	EnabledLayers     []string
}

func (p *PhysicalDevice) CreateLogicalDeviceWithOptions(qfs QueueFamilySlice, options *CreateDeviceOptions) (*Device, error) {

	queueCreateInfos := make([]vk.DeviceQueueCreateInfo, len(qfs))
	for j, q := range qfs {

		queueCreateInfo := vk.DeviceQueueCreateInfo{
			SType:            vk.StructureTypeDeviceQueueCreateInfo,
			QueueFamilyIndex: uint32(q.Index),
			QueueCount:       1,
			PQueuePriorities: []float32{1.0},
		}

		queueCreateInfos[j] = queueCreateInfo

	}

	deviceFeatures := p.VKPhysicalDeviceFeatures()

	deviceCreateInfo := vk.DeviceCreateInfo{
		SType:                vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount: uint32(len(qfs)),
		PQueueCreateInfos:    queueCreateInfos,
		PEnabledFeatures:     []vk.PhysicalDeviceFeatures{deviceFeatures},
	}

	if options != nil {
		if options.EnabledExtensions != nil {
			deviceCreateInfo.EnabledExtensionCount = uint32(len(options.EnabledExtensions))
			deviceCreateInfo.PpEnabledExtensionNames = safeStrings(options.EnabledExtensions)
		}
		if options.EnabledLayers != nil {
			deviceCreateInfo.EnabledLayerCount = uint32(len(options.EnabledLayers))
			deviceCreateInfo.PpEnabledLayerNames = safeStrings(options.EnabledLayers)
		}
	}

	var ldevice vk.Device

	err := vk.Error(vk.CreateDevice(p.VKPhysicalDevice, &deviceCreateInfo, nil, &ldevice))
	if err != nil {
		return nil, err
	}

	var device Device
	device.PhysicalDevice = p
	device.VKDevice = ldevice

	return &device, nil
}

func (p *PhysicalDevice) CreateLogicalDevice(qfs QueueFamilySlice) (*Device, error) {
	return p.CreateLogicalDeviceWithOptions(qfs, nil)
}

func (p *PhysicalDevice) VKPhysicalDeviceFeatures() vk.PhysicalDeviceFeatures {
	var deviceFeatures vk.PhysicalDeviceFeatures
	vk.GetPhysicalDeviceFeatures(p.VKPhysicalDevice, &deviceFeatures)
	return deviceFeatures
}

type MemoryTypeSlice []vk.MemoryType

func (m MemoryTypeSlice) Filter(f func(properties vk.MemoryPropertyFlagBits) bool) MemoryTypeSlice {
	res := make(MemoryTypeSlice, 0)
	for i := 0; i < len(m); i++ {
		if f(vk.MemoryPropertyFlagBits(m[i].PropertyFlags)) {
			res = append(res, m[i])
		}
	}
	return res
}

func (m MemoryTypeSlice) NumHostCoherent() int {
	return len(m.Filter(func(properties vk.MemoryPropertyFlagBits) bool {
		return properties&vk.MemoryPropertyHostCoherentBit != 0
	}))
}

/*
func (m MemoryTypeSlice) NumDeviceVisible() int {
	return len(m.Filter(func(properties vk.MemoryPropertyFlagBits) bool {
		return properties&vk.MemoryPropertyDeviceVisibleBit != 0
	}))
}*/

func (m MemoryTypeSlice) NumHostVisibleAndCoherent() int {
	return len(m.Filter(func(properties vk.MemoryPropertyFlagBits) bool {
		return properties&vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit != vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit
	}))
}

func (m MemoryTypeSlice) NumHostVisible() int {
	return len(m.Filter(func(properties vk.MemoryPropertyFlagBits) bool {
		return properties&vk.MemoryPropertyHostVisibleBit != 0
	}))
}

func (p *PhysicalDevice) MemoryTypes() []vk.MemoryType {
	mp := p.VKPhysicalDeviceMemoryProperties()
	mp.Deref()

	ret := make([]vk.MemoryType, 0)

	var i uint32
	for i = 0; i < mp.MemoryTypeCount; i++ {
		mt := mp.MemoryTypes[i]
		mt.Deref()
		ret = append(ret, mt)
	}
	return ret

}

func (p *PhysicalDevice) VKPhysicalDeviceMemoryProperties() vk.PhysicalDeviceMemoryProperties {
	var memoryProperties vk.PhysicalDeviceMemoryProperties

	vk.GetPhysicalDeviceMemoryProperties(p.VKPhysicalDevice, &memoryProperties)
	return memoryProperties
}

func (p *PhysicalDevice) FindMemoryType(memoryTypeBits uint32, properties vk.MemoryPropertyFlagBits) (uint32, error) {
	memoryProperties := p.VKPhysicalDeviceMemoryProperties()
	mp := &memoryProperties
	mp.Deref()

	/*
	   How does this search work?
	   See the documentation of VkPhysicalDeviceMemoryProperties for a detailed description.
	*/
	var i uint32
	for i = 0; i < mp.MemoryTypeCount; i++ {
		mt := mp.MemoryTypes[i]

		mt.Deref()
		if memoryTypeBits&(1<<i) != 0 &&
			vk.MemoryPropertyFlagBits(mt.PropertyFlags)&properties == properties {
			return i, nil
		}
	}
	return 0, fmt.Errorf("No matching memory type found")
}

func (p *PhysicalDevice) SupportedExtensions() ([]vk.ExtensionProperties, error) {
	var count uint32
	err := vk.Error(vk.EnumerateDeviceExtensionProperties(p.VKPhysicalDevice, "", &count, nil))
	if err != nil {
		return nil, err
	}

	ext := make([]vk.ExtensionProperties, count)

	err = vk.Error(vk.EnumerateDeviceExtensionProperties(p.VKPhysicalDevice, "", &count, ext))
	if err != nil {
		return nil, err
	}
	return ext, nil
}
