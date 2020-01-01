package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type Swapchain struct {
	Extent      vk.Extent2D
	Format      vk.Format
	Device      *Device
	VKSwapchain vk.Swapchain
}

func (s *Swapchain) Destroy() {
	vk.DestroySwapchain(s.Device.VKDevice, s.VKSwapchain, nil)
}

func (s *Swapchain) GetImages() ([]*Image, error) {
	var imageCount uint32
	err := vk.Error(vk.GetSwapchainImages(s.Device.VKDevice, s.VKSwapchain, &imageCount, nil))
	if err != nil {
		return nil, err
	}

	swapchainImages := make([]vk.Image, imageCount)
	err = vk.Error(vk.GetSwapchainImages(s.Device.VKDevice, s.VKSwapchain, &imageCount, swapchainImages))

	ret := make([]*Image, imageCount)
	for i, _ := range swapchainImages {
		ret[i] = &Image{}
		ret[i].Device = s.Device
		ret[i].VKImage = swapchainImages[i]
		ret[i].VKFormat = s.Format
	}

	return ret, err
}

type CreateSwapchainOptions struct {
	OldSwapchain              *Swapchain
	ActualSize                vk.Extent2D
	DesiredNumSwapchainImages int
}

func (p *Device) DefaultNumSwapchainImages(surface vk.Surface) (int, error) {
	caps, err := p.PhysicalDevice.GetSurfaceCapabilities(surface)
	if err != nil {
		return 0, err
	}
	caps.Deref()

	return int(caps.MinImageCount) + 1, nil
}

func (p *Device) CreateSwapchain(surface vk.Surface, graphicsQueue, presentQueue *Queue, options *CreateSwapchainOptions) (*Swapchain, error) {

	modes, err := p.PhysicalDevice.GetSurfacePresentModes(surface)
	if err != nil {
		return nil, err
	}

	presentMode := vk.PresentModeFifo
	m := modes.Filter(vk.PresentModeMailbox)
	if len(m) > 0 {
		presentMode = m[0]
	}

	formats, err := p.PhysicalDevice.GetSurfaceFormats(surface)
	if err != nil {
		return nil, err
	}

	var format vk.SurfaceFormat
	formats.Filter(func(f vk.SurfaceFormat) bool {
		f.Deref()
		if f.Format == vk.FormatB8g8r8a8Unorm {
			format = f
			return true
		}
		return false
	})

	caps, err := p.PhysicalDevice.GetSurfaceCapabilities(surface)
	if err != nil {
		return nil, err
	}
	caps.Deref()

	var swapchainSize vk.Extent2D

	caps.CurrentExtent.Deref()
	if caps.CurrentExtent.Width == vk.MaxUint32 {
		if options != nil {
			swapchainSize = options.ActualSize
		} else {
			swapchainSize = caps.MinImageExtent
		}
	} else {
		swapchainSize = caps.CurrentExtent
	}

	//TODO use the math from the examples to calc images
	desiredSwapChainImages := options.DesiredNumSwapchainImages

	if desiredSwapChainImages == 0 {
		desiredSwapChainImages, err = p.DefaultNumSwapchainImages(surface)
		if err != nil {
			return nil, err
		}

	}

	var swapchain vk.Swapchain

	createInfo := &vk.SwapchainCreateInfo{
		SType:           vk.StructureTypeSwapchainCreateInfo,
		Surface:         surface,
		MinImageCount:   uint32(desiredSwapChainImages),
		ImageFormat:     format.Format,
		ImageColorSpace: format.ColorSpace,
		ImageExtent: vk.Extent2D{
			Width:  swapchainSize.Width,
			Height: swapchainSize.Height,
		},
		PresentMode:      presentMode,
		ImageUsage:       vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit),
		ImageArrayLayers: 1,
		Clipped:          vk.True,
		PreTransform:     caps.CurrentTransform,
		CompositeAlpha:   vk.CompositeAlphaOpaqueBit,
		OldSwapchain:     vk.NullSwapchain,
	}

	if options != nil {
		if options.OldSwapchain != nil {
			createInfo.OldSwapchain = options.OldSwapchain.VKSwapchain
		}

	}

	if graphicsQueue.QueueFamily.Index != presentQueue.QueueFamily.Index {
		createInfo.QueueFamilyIndexCount = 2
		createInfo.PQueueFamilyIndices = []uint32{uint32(graphicsQueue.QueueFamily.Index), uint32(presentQueue.QueueFamily.Index)}
		createInfo.ImageSharingMode = vk.SharingModeConcurrent
	} else {
		createInfo.QueueFamilyIndexCount = 0
		createInfo.PQueueFamilyIndices = nil
		createInfo.ImageSharingMode = vk.SharingModeExclusive
	}

	err = vk.Error(vk.CreateSwapchain(p.VKDevice, createInfo, nil, &swapchain))
	if err != nil {
		return nil, err
	}

	var ret Swapchain
	ret.VKSwapchain = swapchain
	ret.Device = p
	ret.Extent = vk.Extent2D{
		Width:  swapchainSize.Width,
		Height: swapchainSize.Height,
	}
	ret.Format = format.Format

	return &ret, nil

}
