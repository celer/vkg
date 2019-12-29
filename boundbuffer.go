package vkg

import (
	"log"

	vk "github.com/vulkan-go/vulkan"
)

type HostBoundBuffer struct {
	HostBuffer         *Buffer
	HostMemory         *DeviceMemory
	HostMemoryOffset   uint64
	SharedDeviceMemory bool
	BufferObject       BufferObject
}

type StagedBoundBuffer struct {
	HostBoundBuffer

	DeviceBuffer       *Buffer
	DeviceMemory       *DeviceMemory
	DeviceMemoryOffset uint64
}

func (d *Device) CreateHostIndexBuffer(bo BufferObject, sharingMode vk.SharingMode) (*HostBoundBuffer, error) {
	buffer, dmemory, err := d.CreateAndBindBufferAndMemory(uint64(len(bo.Bytes())), 0, vk.BufferUsageFlags(vk.BufferUsageIndexBufferBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit), sharingMode)

	if err != nil {
		return nil, err
	}

	hbb := &HostBoundBuffer{
		HostBuffer:       buffer,
		HostMemory:       dmemory,
		HostMemoryOffset: 0,
		BufferObject:     bo,
	}

	return hbb, nil
}

func (d *Device) CreateHostVertexBuffer(bo BufferObject, sharingMode vk.SharingMode) (*HostBoundBuffer, error) {
	buffer, dmemory, err := d.CreateAndBindBufferAndMemory(uint64(len(bo.Bytes())), 0, vk.BufferUsageFlags(vk.BufferUsageVertexBufferBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit), sharingMode)

	if err != nil {
		return nil, err
	}

	hbb := &HostBoundBuffer{
		HostBuffer:       buffer,
		HostMemory:       dmemory,
		HostMemoryOffset: 0,
		BufferObject:     bo,
	}

	return hbb, nil
}

func (d *Device) CreateAndBindBufferAndMemory(size uint64, offset uint64, usage vk.BufferUsageFlags, mprops vk.MemoryPropertyFlags, sharing vk.SharingMode) (*Buffer, *DeviceMemory, error) {

	buffer, err := d.CreateBufferWithOptions(size, usage, sharing)
	if err != nil {
		return nil, nil, err
	}
	memory, err := d.AllocateForBuffer(buffer, mprops)
	if err != nil {
		buffer.Destroy()
		return nil, nil, err
	}
	buffer.Bind(memory, offset)
	return buffer, memory, nil
}

func (d *Device) CreateStagedBoundBuffer(bo BufferObject) (*StagedBoundBuffer, error) {
	s := &StagedBoundBuffer{}

	s.BufferObject = bo

	size := uint64(len(bo.Bytes()))

	/*
		bo.Lock()
		bo.Unlock()*/

	buffer, memory, err := d.CreateAndBindBufferAndMemory(size, 0,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
		vk.SharingModeExclusive)

	if err != nil {
		return nil, err
	}

	s.HostBuffer = buffer
	s.HostMemory = memory
	s.HostMemoryOffset = 0

	var usage vk.BufferUsageFlags

	usage = usage | vk.BufferUsageFlags(vk.BufferUsageTransferDstBit)

	if _, ok := s.BufferObject.(VertexSource); ok {
		usage |= vk.BufferUsageFlags(vk.BufferUsageVertexBufferBit)
	}
	if _, ok := s.BufferObject.(IndexSource); ok {
		usage |= vk.BufferUsageFlags(vk.BufferUsageIndexBufferBit)
	}

	buffer, memory, err = d.CreateAndBindBufferAndMemory(size, 0,
		usage,
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
		vk.SharingModeExclusive)

	if err != nil {
		s.Destroy()
		return nil, err
	}

	s.DeviceBuffer = buffer
	s.DeviceMemory = memory
	s.DeviceMemoryOffset = 0

	return s, nil
}

func (s *StagedBoundBuffer) Destroy() {
	s.HostBoundBuffer.Destroy()
	if s.DeviceMemory != nil {
		s.DeviceMemory.Destroy()
	}
	if s.DeviceBuffer != nil {
		s.DeviceBuffer.Destroy()
	}
}

func (cb *CommandBuffer) CopyBuffer(s *StagedBoundBuffer) {
	vk.CmdCopyBuffer(cb.VK(), s.HostBuffer.VKBuffer, s.DeviceBuffer.VKBuffer, 1, []vk.BufferCopy{
		vk.BufferCopy{
			SrcOffset: 0,
			DstOffset: 0,
			Size:      vk.DeviceSize(s.HostBuffer.Size),
		},
	})
}

func (d *Device) CreateHostBoundBuffer(bo BufferObject) (*HostBoundBuffer, error) {
	h := &HostBoundBuffer{BufferObject: bo}

	size := uint64(len(bo.Bytes()))
	/*
		bo.Lock()
		bo.Unlock()
	*/

	var usage vk.BufferUsageFlags

	if _, ok := h.BufferObject.(VertexSource); ok {
		usage |= vk.BufferUsageFlags(vk.BufferUsageVertexBufferBit)
		log.Printf("BoundBuffer: VertexSource")
	}
	if _, ok := h.BufferObject.(IndexSource); ok {
		usage |= vk.BufferUsageFlags(vk.BufferUsageIndexBufferBit)
		log.Printf("BoundBuffer: IndexSource")
	}
	if _, ok := h.BufferObject.(UBO); ok {
		usage |= vk.BufferUsageFlags(vk.BufferUsageUniformBufferBit)
		log.Printf("BoundBuffer: UBO")
	}

	buffer, memory, err := d.CreateAndBindBufferAndMemory(size, 0,
		usage,
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
		vk.SharingModeExclusive)

	if err != nil {
		return nil, err
	}

	h.HostBuffer = buffer
	h.HostMemory = memory

	return h, nil
}

func (h *HostBoundBuffer) Map() error {
	//h.BufferObject.Lock()

	data := h.BufferObject.Bytes()

	pm, err := h.HostMemory.MapWithSize(len(data))
	if err != nil {
		//h.BufferObject.Unlock()
		return err
	}

	const m = 0x7fffffff
	outData := (*[m]byte)(pm)[:len(data)]

	copy(outData, data)

	h.HostMemory.Unmap()
	//h.BufferObject.Unlock()

	return nil
}

func (s *HostBoundBuffer) Destroy() {
	if s.HostMemory != nil {
		s.HostMemory.Destroy()
	}
	if s.HostBuffer != nil {
		s.HostBuffer.Destroy()
	}
}
