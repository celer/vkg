package app

import (
	"fmt"
	"log"
	"runtime"

	"github.com/celer/vkg"
	"github.com/vulkan-go/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

type IGraphicsModule interface {
	NewFrame(base *AppBase)
	PostFrame()
	Destroy()
	CreateCommandBuffers(renderPass vk.RenderPass, framebuffer vk.Framebuffer, app *AppBase) ([]vk.CommandBuffer, error)
}

type IInputModule interface {
	KeyChange(key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) bool
	MouseScrollChange(x, y float64) bool
	MouseButtonChange(rawButton glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) bool
	CharChange(char rune) bool
}

type AppBase struct {
	vkg.GraphicsApp

	GraphicsModules []IGraphicsModule
	InputModules    []IInputModule

	priorCommandBuffers []vk.CommandBuffer
}

func NewAppBase(appName string, width, height int) (*AppBase, error) {
	runtime.LockOSThread()

	err := glfw.Init()
	if err != nil {
		return nil, fmt.Errorf("unable to intialize glfw: %w", err)
	}

	if !glfw.VulkanSupported() {
		return nil, fmt.Errorf("vulkan is unsupported")
	}

	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())

	err = vk.Init()
	if err != nil {
		return nil, fmt.Errorf("unable to initialize vulkan: %w", err)
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(width, height, appName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create window: %w", err)
	}

	app, err := vkg.NewGraphicsApp(appName, vkg.Version{0, 0, 1})
	if err != nil {
		return nil, fmt.Errorf("unable to initialize vulkan app: %w", err)
	}

	base := &AppBase{GraphicsApp: *app}

	base.SetWindow(window)
	base.EnableDebugging()

	return base, nil
}

func (b *AppBase) charChange(window *glfw.Window, char rune) {
	for _, i := range b.InputModules {
		if i.CharChange(char) {
			break
		}
	}
}

func (b *AppBase) keyChange(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	for _, i := range b.InputModules {
		if i.KeyChange(key, scancode, action, mods) {
			break
		}
	}

}

func (b *AppBase) mouseScrollChange(window *glfw.Window, x, y float64) {
	for _, i := range b.InputModules {
		if i.MouseScrollChange(x, y) {
			break
		}
	}
}

func (b *AppBase) mouseButtonChange(window *glfw.Window, rawButton glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	for _, i := range b.InputModules {
		if i.MouseButtonChange(rawButton, action, mods) {
			break
		}
	}

}

func (b *AppBase) Init() error {

	err := b.GraphicsApp.Init()
	if err != nil {
		return fmt.Errorf("unable to initialize vulkan instance: %w", err)
	}

	b.MakeCommandBuffer = func(buffer *vkg.CommandBuffer, frame int) {
		b.makeCommandBuffers(buffer, frame)
	}

	b.Window.SetMouseButtonCallback(b.mouseButtonChange)
	b.Window.SetScrollCallback(b.mouseScrollChange)
	b.Window.SetKeyCallback(b.keyChange)
	b.Window.SetCharCallback(b.charChange)

	return nil
}

func (b *AppBase) AddGraphicsModule(g IGraphicsModule) {
	if b.GraphicsModules == nil {
		b.GraphicsModules = make([]IGraphicsModule, 0)
	}
	b.GraphicsModules = append(b.GraphicsModules, g)
}

func (b *AppBase) AddInputModule(i IInputModule) {
	if b.InputModules == nil {
		b.InputModules = make([]IInputModule, 0)
	}
	b.InputModules = append(b.InputModules, i)
}

func (b *AppBase) makeCommandBuffers(buffer *vkg.CommandBuffer, frame int) {

	clearValues := make([]vk.ClearValue, 2)

	clearValues[0].SetColor([]float32{0.2, 0.2, 0.2, 1})
	clearValues[1].SetDepthStencil(1, 0)

	buffer.Begin()

	renderPassBeginInfo := vk.RenderPassBeginInfo{
		SType:       vk.StructureTypeRenderPassBeginInfo,
		RenderPass:  b.VKRenderPass,
		Framebuffer: b.Framebuffers[frame],
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{
				X: 0, Y: 0,
			},
			Extent: b.GetScreenExtent(),
		},
		ClearValueCount: 2,
		PClearValues:    clearValues,
	}

	vk.CmdBeginRenderPass(buffer.VK(), &renderPassBeginInfo, vk.SubpassContentsSecondaryCommandBuffers)

	if len(b.priorCommandBuffers) > 0 {
		vk.FreeCommandBuffers(b.Device.VKDevice, b.GraphicsCommandPool.VKCommandPool, uint32(len(b.priorCommandBuffers)), b.priorCommandBuffers)
	}

	buffers := make([]vk.CommandBuffer, 0)
	for _, g := range b.GraphicsModules {
		cmds, err := g.CreateCommandBuffers(b.VKRenderPass, b.Framebuffers[frame], b)
		if err != nil {
			log.Printf("error generating command buffer: %v", err)
		}
		buffers = append(buffers, cmds...)
	}
	b.priorCommandBuffers = buffers

	if len(buffers) > 0 {
		vk.CmdExecuteCommands(buffer.VK(), uint32(len(buffers)), buffers)
	}

	vk.CmdEndRenderPass(buffer.VK())
	buffer.End()
}

func (b *AppBase) Destroy() {
	for _, g := range b.GraphicsModules {
		g.Destroy()
	}
	b.GraphicsApp.Destroy()

}

func (b *AppBase) ShouldClose() bool {
	return b.Window.ShouldClose()
}

func (b *AppBase) NewFrame() {
	glfw.PollEvents()

	for _, g := range b.GraphicsModules {
		g.NewFrame(b)
	}

}

func (b *AppBase) PostFrame() {
	glfw.PollEvents()

	for _, g := range b.GraphicsModules {
		g.PostFrame()
	}

}
