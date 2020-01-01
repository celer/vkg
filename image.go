package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

// Image is analogous to a buffer, it is essentially a designation that a resource is an image.
type Image struct {
	Device   *Device
	VKImage  vk.Image
	VKFormat vk.Format
	Size     uint64
	Extent   vk.Extent2D
}

// CreateImageWithOptions creates an image with some commonly used options
func (d *Device) CreateImageWithOptions(extent vk.Extent2D, format vk.Format, tiling vk.ImageTiling, usage vk.ImageUsageFlagBits) (*Image, error) {
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
	imageInfo.Usage = vk.ImageUsageFlags(usage)
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
	ret.Extent = extent

	return &ret, nil
}

// VKMemoryRequirements return the memory requirements for the images
func (d *Image) VKMemoryRequirements() vk.MemoryRequirements {
	var memRequirements vk.MemoryRequirements
	vk.GetImageMemoryRequirements(d.Device.VKDevice, d.VKImage, &memRequirements)
	return memRequirements
}

// Destroy destroys the image
func (d *Image) Destroy() {
	if d.VKImage != vk.NullImage {
		vk.DestroyImage(d.Device.VKDevice, d.VKImage, nil)
		d.VKImage = vk.NullImage
	}
}
