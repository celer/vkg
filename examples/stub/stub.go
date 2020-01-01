package main

import (
	"github.com/celer/vkg"
	"github.com/vulkan-go/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {

	err := glfw.Init()
	orPanic(err)

	if !glfw.VulkanSupported() {
		panic("vulkan is not supported")
	}

	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())

	err = vk.Init()
	orPanic(err)

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(800, 600, "Example app", nil, nil)

	app, err := vkg.NewGraphicsApp("Myapp", vkg.Version{0, 0, 1})
	orPanic(err)

	app.SetWindow(window)
	app.EnableDebugging()

	err = app.Init()
	orPanic(err)

	app.MakeCommandBuffer = func(buffer *vkg.CommandBuffer, frame int) {

		clearValues := make([]vk.ClearValue, 2)

		clearValues[0].SetColor([]float32{0.2, 0.2, 0.2, 1})
		clearValues[1].SetDepthStencil(1, 0)

		buffer.Begin()

		renderPassBeginInfo := vk.RenderPassBeginInfo{
			SType:       vk.StructureTypeRenderPassBeginInfo,
			RenderPass:  app.VKRenderPass,
			Framebuffer: app.Framebuffers[frame],
			RenderArea: vk.Rect2D{
				Offset: vk.Offset2D{
					X: 0, Y: 0,
				},
				Extent: app.GetScreenExtent(),
			},
			ClearValueCount: 2,
			PClearValues:    clearValues,
		}

		vk.CmdBeginRenderPass(buffer.VK(), &renderPassBeginInfo, vk.SubpassContentsInline)
		vk.CmdEndRenderPass(buffer.VK())
		buffer.End()

	}

	err = app.PrepareToDraw()
	orPanic(err)

	for {
		if window.ShouldClose() {
			return
		}
		glfw.PollEvents()
		err := app.DrawFrameSync()
		orPanic(err)
	}
	app.Destroy()

}
