package vkg

import (
	"image"
	"image/draw"
	"os"
	"unsafe"

	_ "image/jpeg"
	_ "image/png"

	vk "github.com/vulkan-go/vulkan"
)

type Image struct {
	Device   *Device
	VKImage  vk.Image
	VKFormat vk.Format
}

func (d *Image) GetMemoryRequirements() vk.MemoryRequirements {
	var memRequirements vk.MemoryRequirements
	vk.GetImageMemoryRequirements(d.Device.VKDevice, d.VKImage, &memRequirements)
	return memRequirements
}

func (d *Device) CreateImage(extent vk.Extent2D, format vk.Format, tiling vk.ImageTiling, usage vk.ImageUsageFlags) (*Image, error) {
	var imageInfo = vk.ImageCreateInfo{}
	imageInfo.SType = vk.StructureTypeImageCreateInfo
	imageInfo.ImageType = vk.ImageType2d
	imageInfo.Extent.Width = extent.Width
	imageInfo.Extent.Height = extent.Height
	imageInfo.Extent.Depth = 1
	imageInfo.MipLevels = 1
	imageInfo.ArrayLayers = 1
	imageInfo.Format = format
	imageInfo.Tiling = tiling
	imageInfo.InitialLayout = vk.ImageLayoutUndefined
	imageInfo.Usage = usage
	imageInfo.Samples = vk.SampleCount1Bit
	imageInfo.SharingMode = vk.SharingModeExclusive

	var image vk.Image

	err := vk.Error(vk.CreateImage(d.VKDevice, &imageInfo, nil, &image))
	if err != nil {
		return nil, err
	}

	var ret Image

	ret.Device = d
	ret.VKImage = image
	ret.VKFormat = format

	return &ret, nil
}

type BoundImage struct {
	Image
	DeviceMemory *DeviceMemory
}

type StagedBoundImage struct {
	BoundImage
	HostBuffer       *Buffer
	HostMemory       *DeviceMemory
	HostOffset       int
	HostMemoryOffset uint64
	Width            int
	Height           int
}

type LocalImage struct {
	img *image.RGBA
}

func (l *LocalImage) Bytes() []byte {
	const m = 0x7fffffff
	return (*[m]byte)(unsafe.Pointer(&l.img.Pix[0]))[:len(l.img.Pix)]
}

func LoadImageFromDisk(file string) (*LocalImage, error) {
	imageFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer imageFile.Close()

	src, _, err := image.Decode(imageFile)
	if err != nil {
		return nil, err
	}

	b := src.Bounds()
	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), src, b.Min, draw.Src)

	return &LocalImage{m}, nil
}

func (d *Device) StageRGBAImageFromMemory(img unsafe.Pointer, width, height int) (*StagedBoundImage, error) {

	size := uint64(width * height * 4)

	buffer, memory, err := d.CreateAndBindBufferAndMemory(size, 0,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
		vk.SharingModeExclusive)
	if err != nil {
		return nil, err
	}

	const m = 0x7FFFFFFF
	p := (*[m]byte)(img)[:size]

	memory.MapCopyUnmap(p)

	bi, err := d.CreateBoundImage(
		vk.Extent2D{Width: uint32(width), Height: uint32(height)},
		vk.FormatR8g8b8a8Unorm,
		vk.ImageTilingOptimal,
		vk.ImageUsageFlags(vk.ImageUsageTransferDstBit|vk.ImageUsageSampledBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit))

	if err != nil {
		return nil, err
	}

	si := &StagedBoundImage{
		HostMemory: memory,
		HostBuffer: buffer,
		HostOffset: 0,
	}
	si.Device = d
	si.VKImage = bi.VKImage
	si.DeviceMemory = bi.DeviceMemory
	si.VKFormat = bi.VKFormat
	si.Width = width
	si.Height = height

	return si, nil
}

func (d *Device) StageImageFromDisk(file string) (*StagedBoundImage, error) {

	img, err := LoadImageFromDisk(file)
	if err != nil {
		return nil, err
	}

	size := uint64(len(img.Bytes()))

	buffer, memory, err := d.CreateAndBindBufferAndMemory(size, 0,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
		vk.SharingModeExclusive)
	if err != nil {
		return nil, err
	}

	memory.MapCopyUnmap(img.Bytes())

	bounds := img.img.Bounds()

	bi, err := d.CreateBoundImage(
		vk.Extent2D{Width: uint32(bounds.Dx()), Height: uint32(bounds.Dy())},
		vk.FormatR8g8b8a8Unorm,
		vk.ImageTilingOptimal,
		vk.ImageUsageFlags(vk.ImageUsageTransferDstBit|vk.ImageUsageSampledBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit))

	if err != nil {
		return nil, err
	}

	si := &StagedBoundImage{
		HostMemory: memory,
		HostBuffer: buffer,
		HostOffset: 0,
	}
	si.Device = d
	si.VKImage = bi.VKImage
	si.DeviceMemory = bi.DeviceMemory
	si.VKFormat = bi.VKFormat
	si.Width = bounds.Dx()
	si.Height = bounds.Dy()

	return si, nil
}

func (cb *CommandBuffer) TransitionImageLayout(s *StagedBoundImage, format vk.Format, oldLayout, newLayout vk.ImageLayout) {
	var barrier = vk.ImageMemoryBarrier{}
	barrier.SType = vk.StructureTypeImageMemoryBarrier
	barrier.OldLayout = oldLayout
	barrier.NewLayout = newLayout
	barrier.SrcQueueFamilyIndex = vk.QueueFamilyIgnored
	barrier.DstQueueFamilyIndex = vk.QueueFamilyIgnored
	barrier.Image = s.VKImage
	barrier.SubresourceRange.AspectMask = vk.ImageAspectFlags(vk.ImageAspectColorBit)
	barrier.SubresourceRange.BaseMipLevel = 0
	barrier.SubresourceRange.LevelCount = 1
	barrier.SubresourceRange.BaseArrayLayer = 0
	barrier.SubresourceRange.LayerCount = 1
	barrier.SrcAccessMask = 0 // TODO
	barrier.DstAccessMask = 0 // TODO

	var sourceStage, destStage vk.PipelineStageFlags

	if oldLayout == vk.ImageLayoutUndefined && newLayout == vk.ImageLayoutTransferDstOptimal {
		barrier.SrcAccessMask = 0
		barrier.DstAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)

		sourceStage = vk.PipelineStageFlags(vk.PipelineStageTopOfPipeBit)
		destStage = vk.PipelineStageFlags(vk.PipelineStageTransferBit)

	} else if oldLayout == vk.ImageLayoutTransferDstOptimal && newLayout == vk.ImageLayoutShaderReadOnlyOptimal {
		barrier.SrcAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)
		barrier.DstAccessMask = vk.AccessFlags(vk.AccessShaderReadBit)

		sourceStage = vk.PipelineStageFlags(vk.PipelineStageTransferBit)
		destStage = vk.PipelineStageFlags(vk.PipelineStageFragmentShaderBit)
	}

	vk.CmdPipelineBarrier(cb.VK(), sourceStage, destStage, 0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{barrier})

}

func (cb *CommandBuffer) CopyImage(s *StagedBoundImage) {
	vk.CmdCopyBufferToImage(cb.VK(), s.HostBuffer.VKBuffer, s.VKImage, vk.ImageLayoutTransferDstOptimal, 1, []vk.BufferImageCopy{
		vk.BufferImageCopy{
			BufferOffset:      0,
			BufferRowLength:   0,
			BufferImageHeight: 0,
			ImageSubresource: vk.ImageSubresourceLayers{
				AspectMask:     vk.ImageAspectFlags(vk.ImageAspectColorBit),
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			ImageOffset: vk.Offset3D{},
			ImageExtent: vk.Extent3D{
				Width: uint32(s.Width), Height: uint32(s.Height), Depth: 1,
			},
		},
	})
}

func (d *Device) CreateBoundImage(extent vk.Extent2D, format vk.Format, tiling vk.ImageTiling, usage vk.ImageUsageFlags, props vk.MemoryPropertyFlags) (*BoundImage, error) {
	i, err := d.CreateImage(extent, format, tiling, usage)
	if err != nil {
		return nil, err
	}

	mr := i.GetMemoryRequirements()

	mr.Deref()

	mem, err := d.Allocate(int(mr.Size), mr.MemoryTypeBits, props)
	if err != nil {
		return nil, err
	}

	boundImage := &BoundImage{}

	boundImage.Device = d
	boundImage.VKImage = i.VKImage
	boundImage.DeviceMemory = mem
	boundImage.VKFormat = i.VKFormat

	vk.BindImageMemory(d.VKDevice, i.VKImage, mem.VKDeviceMemory, 0)

	return boundImage, nil

}

func (i *Image) Destroy() {
	vk.DestroyImage(i.Device.VKDevice, i.VKImage, nil)
}

type ImageView struct {
	Device      *Device
	VKImageView vk.ImageView
}

func (i *ImageView) Destroy() {
	vk.DestroyImageView(i.Device.VKDevice, i.VKImageView, nil)
}

func (i *Image) CreateImageViewWithAspectMask(mask vk.ImageAspectFlags) (*ImageView, error) {
	createImage := &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    i.VKImage,
		ViewType: vk.ImageViewType2d,
		Format:   i.VKFormat,
		Components: vk.ComponentMapping{
			R: vk.ComponentSwizzleR,
			G: vk.ComponentSwizzleG,
			B: vk.ComponentSwizzleB,
			A: vk.ComponentSwizzleA,
		},
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: mask,
			LevelCount: 1,
			LayerCount: 1,
		},
	}

	var view vk.ImageView

	err := vk.Error(vk.CreateImageView(i.Device.VKDevice, createImage, nil, &view))
	if err != nil {
		return nil, err
	}
	var ret ImageView
	ret.Device = i.Device
	ret.VKImageView = view

	return &ret, nil

}

func (i *Image) CreateImageView() (*ImageView, error) {
	return i.CreateImageViewWithAspectMask(vk.ImageAspectFlags(vk.ImageAspectColorBit))
}
