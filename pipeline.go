package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type ComputePipeline struct {
	VKPipeline                      vk.Pipeline
	VKPipelineShaderStageCreateInfo vk.PipelineShaderStageCreateInfo
	VKPipelineLayout                vk.PipelineLayout
}

type PipelineCache struct {
	VKPipelineCache vk.PipelineCache
}

func (d *Device) CreatePipelineCache() (*PipelineCache, error) {
	var pipelineCacheCreate = vk.PipelineCacheCreateInfo{}
	pipelineCacheCreate.SType = vk.StructureTypePipelineCacheCreateInfo

	var pipelineCache vk.PipelineCache

	err := vk.Error(vk.CreatePipelineCache(d.VKDevice, &pipelineCacheCreate, nil, &pipelineCache))
	if err != nil {
		return nil, err
	}

	var ret PipelineCache
	ret.VKPipelineCache = pipelineCache
	return &ret, nil
}

func (c *ComputePipeline) SetPipelineLayout(layout *PipelineLayout) {
	c.VKPipelineLayout = layout.VKPipelineLayout
}

func (c *ComputePipeline) SetShaderStage(entryPoint string, shaderModule *ShaderModule) {
	c.VKPipelineShaderStageCreateInfo = shaderModule.VKPipelineShaderStageCreateInfo(vk.ShaderStageComputeBit, entryPoint)
}

func (d *Device) CreateComputePipelines(pc *PipelineCache, cp ...*ComputePipeline) error {

	pipelines := make([]vk.Pipeline, len(cp))

	ci := make([]vk.ComputePipelineCreateInfo, len(cp))

	for i, p := range cp {
		var pipelineCreateInfo = vk.ComputePipelineCreateInfo{}
		pipelineCreateInfo.SType = vk.StructureTypeComputePipelineCreateInfo
		pipelineCreateInfo.Stage = p.VKPipelineShaderStageCreateInfo
		pipelineCreateInfo.Layout = p.VKPipelineLayout
		ci[i] = pipelineCreateInfo
	}

	err := vk.Error(vk.CreateComputePipelines(
		d.VKDevice, pc.VKPipelineCache,
		1, ci,
		nil, pipelines))

	if err != nil {
		return err
	}

	for i, _ := range pipelines {
		cp[i].VKPipeline = pipelines[i]
	}

	return nil

}
