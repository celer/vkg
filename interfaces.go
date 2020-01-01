package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

//Destroyer
type IDestructable interface {
	Destroy()
}

//GraphicsPipelineConfigurer
type IGraphicsPipelineConfig interface {
	VKGraphicsPipelineCreateInfo(screenExtents vk.Extent2D) (vk.GraphicsPipelineCreateInfo, error)
	Destroy()
}

// IAllocator is an interface for generic memroy allocation, it
// can be utilized to provide a more powerful allocation capability
// than the currently implemented linear allocator
//Allocater
type IAllocator interface {
	Free(a *Allocation)
	Allocate(size uint64, align uint64) *Allocation
	Allocations() []*Allocation
	DestroyContents()
	LogDetails()
}

type IAllocatedItem interface {
	Destroy()
	String() string
}

type MappedMemoryRange interface {
	VKMappedMemoryRange() vk.MappedMemoryRange
}

type VKImageProvider interface {
	GetVKImage() vk.Image
}

type VKBufferProvider interface {
	GetVKBufer() vk.Buffer
}

type Descriptor struct {
	Type        vk.DescriptorType
	ShaderStage vk.ShaderStageFlags
	Set         int
	Binding     int
}

type DescriptorBinder interface {
	Descriptor() *Descriptor
}

type VertexDescriptor interface {
	GetBindingDescription() vk.VertexInputBindingDescription
	GetAttributeDescriptions() []vk.VertexInputAttributeDescription
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

// You try to make a idiomatic go interface name out of an object that can represent stuff as bytes - byteabler?
type ByteSourcer interface {
	Bytes() []byte
}

type IndexSourcer interface {
	ByteSourcer
	IndexType() vk.IndexType
}

type VertexSourcer interface {
	ByteSourcer
	VertexDescriptor
}
