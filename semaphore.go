package vkg

import (
	vk "github.com/vulkan-go/vulkan"
)

func (d *Device) VKCreateSemaphore() (vk.Semaphore, error) {
	semaphoreCreateInfo := vk.SemaphoreCreateInfo{
		SType: vk.StructureTypeSemaphoreCreateInfo,
	}

	var sema vk.Semaphore

	err := vk.Error(vk.CreateSemaphore(d.VKDevice, &semaphoreCreateInfo, nil, &sema))

	return sema, err
}
