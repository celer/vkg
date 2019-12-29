package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type DescriptorPool struct {
	Device           *Device
	VKDescriptorPool vk.DescriptorPool
}

type DescriptorPoolContents struct {
	VKDescriptorPoolSize []vk.DescriptorPoolSize
}

func (d *DescriptorPoolContents) AddPoolSize(dtype vk.DescriptorType, count int) {
	if d.VKDescriptorPoolSize == nil {
		d.VKDescriptorPoolSize = make([]vk.DescriptorPoolSize, 0)
	}
	d.VKDescriptorPoolSize = append(d.VKDescriptorPoolSize, vk.DescriptorPoolSize{
		Type:            dtype,
		DescriptorCount: uint32(count),
	})
}

func (d *Device) CreateDescriptorPool(maxSets int, contents *DescriptorPoolContents) (*DescriptorPool, error) {

	var descriptorPoolCreateInfo = vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       uint32(maxSets),
		PoolSizeCount: uint32(len(contents.VKDescriptorPoolSize)),
		PPoolSizes:    contents.VKDescriptorPoolSize,
	}

	var descriptorPool vk.DescriptorPool
	err := vk.Error(vk.CreateDescriptorPool(d.VKDevice, &descriptorPoolCreateInfo, nil, &descriptorPool))

	if err != nil {
		return nil, err
	}

	var ret DescriptorPool

	ret.Device = d
	ret.VKDescriptorPool = descriptorPool

	return &ret, nil

}

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
