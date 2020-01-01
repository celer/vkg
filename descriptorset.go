package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

// DescriptorSet is a binding of resources to a descriptor, per a specific DescriptorSetLayout
type DescriptorSet struct {
	Device               *Device
	DescriptorPool       *DescriptorPool
	VKDescriptorSet      vk.DescriptorSet
	VKWriteDiscriptorSet []vk.WriteDescriptorSet
}

func (d *Device) NewDescriptorSet() *DescriptorSet {
	return &DescriptorSet{Device: d}
}

// AddBuffer adds a specific buffer to this descirptor set
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

// AddCombinedImageSampler adds an image layout, image view and sampler to support displaying a texture
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

// Write modifies the descriptor set
func (du *DescriptorSet) Write() {
	for i := range du.VKWriteDiscriptorSet {
		du.VKWriteDiscriptorSet[i].DstSet = du.VKDescriptorSet
	}
	vk.UpdateDescriptorSets(du.Device.VKDevice, uint32(len(du.VKWriteDiscriptorSet)), du.VKWriteDiscriptorSet, 0, nil)
}
