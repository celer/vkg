package vkg

import (
	vk "github.com/vulkan-go/vulkan"
	"unsafe"
)

type IndexSliceUint16 []uint16

func (i IndexSliceUint16) Bytes() []byte {
	size := len(i) * int(unsafe.Sizeof(uint16(1)))
	return ToBytes(unsafe.Pointer(&i[0]), size)
}

func (i IndexSliceUint16) IndexType() vk.IndexType {
	return vk.IndexTypeUint16
}

type IndexSliceUint32 []uint16

func (i IndexSliceUint32) Bytes() []byte {
	size := len(i) * int(unsafe.Sizeof(uint32(1)))
	return ToBytes(unsafe.Pointer(&i[0]), size)
}

func (i IndexSliceUint32) IndexType() vk.IndexType {
	return vk.IndexTypeUint32
}
