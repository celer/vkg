package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

type ImageView struct {
	Device      *Device
	VKImageView vk.ImageView
}

func (i *Image) CreateImageView() (*ImageView, error) {
	return i.CreateImageViewWithAspectMask(vk.ImageAspectFlags(vk.ImageAspectColorBit))
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

func (i *ImageView) Destroy() {
	vk.DestroyImageView(i.Device.VKDevice, i.VKImageView, nil)
}
