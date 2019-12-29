package vkg

import (
	vk "github.com/vulkan-go/vulkan"
	"io/ioutil"
	"unsafe"
)

type ShaderModule struct {
	Device         *Device
	Description    string
	VKShaderModule vk.ShaderModule
}

func (d *Device) LoadShaderModuleFromFile(file string) (*ShaderModule, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var module vk.ShaderModule
	err = vk.Error(vk.CreateShaderModule(d.VKDevice, &vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint(len(data)),
		PCode:    sliceUint32(data),
	}, nil, &module))

	if err != nil {
		return nil, err
	}

	var ret ShaderModule
	ret.VKShaderModule = module
	ret.Device = d
	return &ret, nil
}

func (s *ShaderModule) VKPipelineShaderStageCreateInfo(stage vk.ShaderStageFlagBits, entryPoint string) vk.PipelineShaderStageCreateInfo {
	var shaderStageCreateInfo = vk.PipelineShaderStageCreateInfo{}
	shaderStageCreateInfo.SType = vk.StructureTypePipelineShaderStageCreateInfo
	shaderStageCreateInfo.Stage = stage
	shaderStageCreateInfo.Module = s.VKShaderModule
	shaderStageCreateInfo.PName = safeString(entryPoint)
	return shaderStageCreateInfo
}

func (s *ShaderModule) Destroy() {
	vk.DestroyShaderModule(s.Device.VKDevice, s.VKShaderModule, nil)
}

func sliceUint32(data []byte) []uint32 {
	const m = 0x7fffffff
	return (*[m / 4]uint32)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&data)).Data))[:len(data)/4]
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
