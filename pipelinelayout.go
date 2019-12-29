package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type PipelineLayout struct {
	Device           *Device
	VKPipelineLayout vk.PipelineLayout
}

func (p *PipelineLayout) Destroy() {
	vk.DestroyPipelineLayout(p.Device.VKDevice, p.VKPipelineLayout, nil)
}

func (d *Device) CreatePipelineLayout(descriptorSetLayouts ...*DescriptorSetLayout) (*PipelineLayout, error) {
	var pipelineLayoutCreateInfo = vk.PipelineLayoutCreateInfo{}
	pipelineLayoutCreateInfo.SType = vk.StructureTypePipelineLayoutCreateInfo
	pipelineLayoutCreateInfo.SetLayoutCount = uint32(len(descriptorSetLayouts))

	l := make([]vk.DescriptorSetLayout, len(descriptorSetLayouts))
	for i, dsl := range descriptorSetLayouts {
		l[i] = dsl.VKDescriptorSetLayout
	}

	pipelineLayoutCreateInfo.PSetLayouts = l

	var pipelineLayout vk.PipelineLayout

	err := vk.Error(vk.CreatePipelineLayout(d.VKDevice, &pipelineLayoutCreateInfo, nil, &pipelineLayout))
	if err != nil {
		return nil, err
	}

	var ret PipelineLayout

	ret.VKPipelineLayout = pipelineLayout
	ret.Device = d

	return &ret, nil

}
