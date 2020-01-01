package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

// DescriptorSetLayout describes the layout of a descriptorset
type DescriptorSetLayout struct {
	Device                        *Device
	VKDescriptorSetLayout         vk.DescriptorSetLayout
	VKDescriptorSetLayoutBindings []vk.DescriptorSetLayoutBinding
}

func (d *Device) NewDescriptorSetLayout() *DescriptorSetLayout {
	return &DescriptorSetLayout{Device: d}
}

// AddBinding adds a binding to the descriptor set
func (d *DescriptorSetLayout) AddBinding(binding vk.DescriptorSetLayoutBinding) {
	if d.VKDescriptorSetLayoutBindings == nil {
		d.VKDescriptorSetLayoutBindings = make([]vk.DescriptorSetLayoutBinding, 0)
	}
	d.VKDescriptorSetLayoutBindings = append(d.VKDescriptorSetLayoutBindings, binding)
}

// Destroy destroys this descriptor set layout
func (d *DescriptorSetLayout) Destroy() {
	vk.DestroyDescriptorSetLayout(d.Device.VKDevice, d.VKDescriptorSetLayout, nil)
}

// CreateDescriptorSetLayout creates this descriptor set layout
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
