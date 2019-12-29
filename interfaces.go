package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type Descriptor struct {
	Type        vk.DescriptorType
	ShaderStage vk.ShaderStageFlags
	Set         int
	Binding     int
}

type DescriptorBinder interface {
	Descriptor() *Descriptor
}

/*
	Does every object need multiple buffers for all purposes?
	No, a grid for an editor does not
	No, a tool widget does not
*/

type UpdateFrequencyType int

const (
	// If this object is put into device memory it will be tossed from host memory?
	Once UpdateFrequencyType = iota
	// We will potentialy pause the draw frame to update this
	Occasionally
	// We will assume this item needs to be updated almost every frame
	Frequently
)

type AutoAllocatedObject interface {
	//We make the assumption that the stable buffer object
	// won't change much, so we won't pre-allocate multiple
	// buffers for it, instead we will only allocate 1 buffer
	// and treat changes as one off event, which could impact performance
	// we will pause the drawframe to recopy the data
	UpdateFrequency() UpdateFrequencyType
}

type MutableBufferObject interface {
	IsValid() bool
	Invalidate()

	// Lock prior to copying the data
	Lock()
	// Unlock after copying the data
	Unlock()
}

type BufferObject interface {
	Bytes() []byte
}

type IndexSource interface {
	BufferObject
	IndexType() vk.IndexType
}

type VertexSource interface {
	BufferObject
	GetBindingDesciption() vk.VertexInputBindingDescription
	GetAttributeDescriptions() []vk.VertexInputAttributeDescription
}

type UBO interface {
	BufferObject
	DescriptorBinder
}

type Model interface {
	VertexSource
	UBO() UBO
}
