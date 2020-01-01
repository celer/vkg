package vkg

import (
	"fmt"
	"image"
	"image/draw"

	// Load the png image loader
	_ "image/png"
	"os"
	"time"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

func (p *ImageResourcePool) StageTextureFromDisk(filename string, cmd *CommandBuffer, queue *Queue) (*ImageResource, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	src, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()

	var extent vk.Extent2D

	extent.Width = uint32(b.Dx())
	extent.Height = uint32(b.Dy())

	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), src, b.Min, draw.Src)

	return p.StageTextureFromImage(m, cmd, queue)
}

func (p *ImageResourcePool) StageTextureFromImage(srcImg *image.RGBA, cmd *CommandBuffer, queue *Queue) (*ImageResource, error) {

	b := srcImg.Bounds()

	var extent vk.Extent2D

	extent.Width = uint32(b.Dx())
	extent.Height = uint32(b.Dy())

	img, err := p.AllocateImage(extent, vk.FormatR8g8b8a8Unorm, vk.ImageTilingOptimal, vk.ImageUsageTransferDstBit|vk.ImageUsageSampledBit)
	if err != nil {
		return nil, err
	}

	img.AllocateStagingResource()
	defer img.FreeStagingResource()

	img.StagingResource.ResourcePool.Memory.Map()

	const c = 0x7fffffff

	mbytes := (*[c]byte)(unsafe.Pointer(&srcImg.Pix[0]))[:len(srcImg.Pix)]

	srb := img.StagingResource.Bytes()
	if srb == nil {
		return nil, fmt.Errorf("unable to map bytes for image data, make sure staging buffer has been mapped")
	}

	copy(srb, mbytes)

	cmd.BeginOneTime()
	cmd.TransitionImageLayout(img, vk.FormatR8g8b8a8Unorm, vk.ImageLayoutUndefined, vk.ImageLayoutTransferDstOptimal)
	cmd.StageImageResource(img)
	cmd.TransitionImageLayout(img, vk.FormatR8g8b8a8Unorm, vk.ImageLayoutTransferDstOptimal, vk.ImageLayoutShaderReadOnlyOptimal)
	cmd.End()

	f, err := p.Device.CreateFence()
	if err != nil {
		return nil, err
	}
	defer f.Destroy()

	err = queue.SubmitWithFence(f, cmd)
	if err != nil {
		return nil, err
	}

	p.Device.WaitForFences(true, 100*time.Second, f)

	return img, nil

}
