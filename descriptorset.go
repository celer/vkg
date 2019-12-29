package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type DescriptorSetLayout struct {
	Device                        *Device
	VKDescriptorSetLayout         vk.DescriptorSetLayout
	VKDescriptorSetLayoutBindings []vk.DescriptorSetLayoutBinding
}

func (d *DescriptorSetLayout) AddBinding(binding vk.DescriptorSetLayoutBinding) {
	if d.VKDescriptorSetLayoutBindings == nil {
		d.VKDescriptorSetLayoutBindings = make([]vk.DescriptorSetLayoutBinding, 0)
	}
	d.VKDescriptorSetLayoutBindings = append(d.VKDescriptorSetLayoutBindings, binding)
}

func (d *DescriptorSetLayout) Destroy() {
	vk.DestroyDescriptorSetLayout(d.Device.VKDevice, d.VKDescriptorSetLayout, nil)
}

func (d *Device) CreateDescriptorSetLayout(layout *DescriptorSetLayout) (*DescriptorSetLayout, error) {
	var descriptorSetLayoutCreateInfo = &vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: uint32(len(layout.VKDescriptorSetLayoutBindings)),
		PBindings:    layout.VKDescriptorSetLayoutBindings,
	}

	var descriptorSetLayout vk.DescriptorSetLayout
	err := vk.Error(vk.CreateDescriptorSetLayout(d.VKDevice, descriptorSetLayoutCreateInfo, nil, &descriptorSetLayout))
	if err != nil {
		return nil, err
	}

	layout.Device = d
	layout.VKDescriptorSetLayout = descriptorSetLayout

	return layout, nil
}

type DescriptorSet struct {
	Device               *Device
	DescriptorPool       *DescriptorPool
	VKDescriptorSet      vk.DescriptorSet
	VKWriteDiscriptorSet []vk.WriteDescriptorSet
}

func (du *DescriptorSet) AddBuffer(dstBinding int, dtype vk.DescriptorType, b *Buffer, offset int) {
	var descriptorBufferInfo = vk.DescriptorBufferInfo{}
	descriptorBufferInfo.Buffer = b.VKBuffer
	descriptorBufferInfo.Offset = vk.DeviceSize(offset)
	descriptorBufferInfo.Range = vk.DeviceSize(b.Size)

	var writeDescriptorSet = vk.WriteDescriptorSet{}
	writeDescriptorSet.SType = vk.StructureTypeWriteDescriptorSet
	writeDescriptorSet.DstBinding = uint32(dstBinding) // write to the first, and only binding.
	writeDescriptorSet.DescriptorCount = 1             // update a single descriptor.
	writeDescriptorSet.DescriptorType = dtype
	writeDescriptorSet.PBufferInfo = []vk.DescriptorBufferInfo{descriptorBufferInfo}

	if du.VKWriteDiscriptorSet == nil {
		du.VKWriteDiscriptorSet = make([]vk.WriteDescriptorSet, 0)
	}
	du.VKWriteDiscriptorSet = append(du.VKWriteDiscriptorSet, writeDescriptorSet)
}

func (du *DescriptorSet) AddCombinedImageSampler(dstBinding int, layout vk.ImageLayout, imageView vk.ImageView, sampler vk.Sampler) {

	var descriptorImageInfo = vk.DescriptorImageInfo{}
	descriptorImageInfo.ImageView = imageView
	descriptorImageInfo.ImageLayout = layout
	descriptorImageInfo.Sampler = sampler

	var writeDescriptorSet = vk.WriteDescriptorSet{}
	writeDescriptorSet.SType = vk.StructureTypeWriteDescriptorSet
	writeDescriptorSet.DstBinding = uint32(dstBinding) // write to the first, and only binding.
	writeDescriptorSet.DescriptorCount = 1             // update a single descriptor.
	writeDescriptorSet.DescriptorType = vk.DescriptorTypeCombinedImageSampler
	writeDescriptorSet.PImageInfo = []vk.DescriptorImageInfo{descriptorImageInfo}

	if du.VKWriteDiscriptorSet == nil {
		du.VKWriteDiscriptorSet = make([]vk.WriteDescriptorSet, 0)
	}
	du.VKWriteDiscriptorSet = append(du.VKWriteDiscriptorSet, writeDescriptorSet)

}

func (d *DescriptorSet) Write() {
	for i, _ := range d.VKWriteDiscriptorSet {
		d.VKWriteDiscriptorSet[i].DstSet = d.VKDescriptorSet
	}
	vk.UpdateDescriptorSets(d.Device.VKDevice, uint32(len(d.VKWriteDiscriptorSet)), d.VKWriteDiscriptorSet, 0, nil)
}
