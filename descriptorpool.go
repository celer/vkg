package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

// DescriptorPool is essentially a resource manager for descriptor pools provided by Vulkan.
type DescriptorPool struct {
	Device               *Device
	VKDescriptorPool     vk.DescriptorPool
	VKDescriptorPoolSize []vk.DescriptorPoolSize
}

func (d *Device) NewDescriptorPool() *DescriptorPool {
	return &DescriptorPool{Device: d}
}

// AddPoolSize informs the descriptor pool how many of a certain descriptortype it will contain
func (d *DescriptorPool) AddPoolSize(dtype vk.DescriptorType, count int) {
	if d.VKDescriptorPoolSize == nil {
		d.VKDescriptorPoolSize = make([]vk.DescriptorPoolSize, 0)
	}
	d.VKDescriptorPoolSize = append(d.VKDescriptorPoolSize, vk.DescriptorPoolSize{
		Type:            dtype,
		DescriptorCount: uint32(count),
	})
}

// CreateDescriptorPool creates the descriptor pool
func (d *Device) CreateDescriptorPool(pool *DescriptorPool, maxSets int) (*DescriptorPool, error) {

	var descriptorPoolCreateInfo = vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       uint32(maxSets),
		Flags:         vk.DescriptorPoolCreateFlags(vk.DescriptorPoolCreateFreeDescriptorSetBit),
		PoolSizeCount: uint32(len(pool.VKDescriptorPoolSize)),
		PPoolSizes:    pool.VKDescriptorPoolSize,
	}

	var descriptorPool vk.DescriptorPool
	err := vk.Error(vk.CreateDescriptorPool(d.VKDevice, &descriptorPoolCreateInfo, nil, &descriptorPool))

	if err != nil {
		return nil, err
	}

	pool.Device = d
	pool.VKDescriptorPool = descriptorPool

	return pool, nil

}

// Allocate allocates a descriptor set from the pool given the descriptor set layout
func (d *DescriptorPool) Allocate(layouts ...*DescriptorSetLayout) (*DescriptorSet, error) {

	descriptorSetAllocateInfo := vk.DescriptorSetAllocateInfo{}
	descriptorSetAllocateInfo.SType = vk.StructureTypeDescriptorSetAllocateInfo
	descriptorSetAllocateInfo.DescriptorPool = d.VKDescriptorPool
	descriptorSetAllocateInfo.DescriptorSetCount = uint32(len(layouts))

	dsl := make([]vk.DescriptorSetLayout, len(layouts))

	for i, ds := range layouts {
		dsl[i] = ds.VKDescriptorSetLayout
	}

	descriptorSetAllocateInfo.PSetLayouts = dsl

	var descriptorSet vk.DescriptorSet
	err := vk.Error(vk.AllocateDescriptorSets(d.Device.VKDevice, &descriptorSetAllocateInfo, &descriptorSet))

	if err != nil {
		return nil, err
	}

	var ret DescriptorSet

	ret.Device = d.Device
	ret.VKDescriptorSet = descriptorSet
	ret.DescriptorPool = d

	return &ret, nil

}

func (d *DescriptorPool) Reset() error {
	return vk.Error(vk.ResetDescriptorPool(d.Device.VKDevice, d.VKDescriptorPool, 0))
}

func (d *DescriptorPool) Free(ds *DescriptorSet) error {
	var descriptorSet vk.DescriptorSet

	descriptorSet = ds.VKDescriptorSet

	return vk.Error(vk.FreeDescriptorSets(d.Device.VKDevice, d.VKDescriptorPool, 1, &descriptorSet))
}

func (d *DescriptorPool) Destroy() {
	vk.DestroyDescriptorPool(d.Device.VKDevice, d.VKDescriptorPool, nil)
}
