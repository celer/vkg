package imgui

import (
	"math"

	"github.com/celer/vkg/examples/imgui/app"

	"github.com/inkyblackness/imgui-go"
	"github.com/vulkan-go/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

type UI interface {
	DrawUI()
}

type ImGUIModule struct {
	io       imgui.IO
	renderer *Renderer
	window   *glfw.Window
	context  *imgui.Context

	time             float64
	mouseJustPressed [3]bool

	wantMouse    bool
	wantKeyboard bool

	uis []UI
}

func NewImGUIModule(base *app.AppBase, window *glfw.Window) (*ImGUIModule, error) {

	context := imgui.CreateContext(nil)
	io := imgui.CurrentIO()

	renderer, err := NewRenderer(io, &base.GraphicsApp, 150*1000, 150*1000)
	if err != nil {
		return nil, err
	}

	err = renderer.Init()
	if err != nil {
		return nil, err
	}

	i := &ImGUIModule{
		context:  context,
		io:       io,
		renderer: renderer,
		window:   window,
	}

	i.setKeyMapping()

	return i, nil
}

func (i *ImGUIModule) AddUI(ui UI) {
	if i.uis == nil {
		i.uis = make([]UI, 0)
	}

	i.uis = append(i.uis, ui)
}

func (i *ImGUIModule) NewFrame(base *app.AppBase) {

	i.wantMouse = i.io.WantCaptureMouse()
	i.wantKeyboard = i.io.WantCaptureKeyboard()

	currentTime := glfw.GetTime()
	if i.time > 0 {
		i.io.SetDeltaTime(float32(currentTime - i.time))
	}
	i.time = currentTime

	if i.window.GetAttrib(glfw.Focused) != 0 {
		x, y := i.window.GetCursorPos()
		i.io.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
	} else {
		i.io.SetMousePosition(imgui.Vec2{X: -math.MaxFloat32, Y: -math.MaxFloat32})
	}

	for j := 0; j < len(i.mouseJustPressed); j++ {
		down := i.mouseJustPressed[j] || (i.window.GetMouseButton(glfwButtonIDByIndex[j]) == glfw.Press)
		i.io.SetMouseButtonDown(j, down)
		i.mouseJustPressed[j] = false
	}

}

func (i *ImGUIModule) PostFrame() {

}

func (i *ImGUIModule) Destroy() {
	i.renderer.Destroy()
}

func (i *ImGUIModule) CreateCommandBuffers(renderPass vk.RenderPass, framebuffer vk.Framebuffer, app *app.AppBase) ([]vk.CommandBuffer, error) {
	extent := app.GetScreenExtent()
	i.io.SetDisplaySize(imgui.Vec2{X: float32(extent.Width), Y: float32(extent.Height)})
	imgui.NewFrame()

	for _, ui := range i.uis {
		ui.DrawUI()
	}

	imgui.Render()
	return i.renderer.Render(renderPass, framebuffer, imgui.RenderedDrawData())
}

func (i *ImGUIModule) KeyChange(key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) bool {

	if !i.wantKeyboard {
		return false
	}

	if action == glfw.Press {
		i.io.KeyPress(int(key))
	}
	if action == glfw.Release {
		i.io.KeyRelease(int(key))
	}

	// Modifiers are not reliable across systems
	i.io.KeyCtrl(int(glfw.KeyLeftControl), int(glfw.KeyRightControl))
	i.io.KeyShift(int(glfw.KeyLeftShift), int(glfw.KeyRightShift))
	i.io.KeyAlt(int(glfw.KeyLeftAlt), int(glfw.KeyRightAlt))
	i.io.KeySuper(int(glfw.KeyLeftSuper), int(glfw.KeyRightSuper))

	return true

}

func (i *ImGUIModule) setKeyMapping() {
	// Keyboard mapping. ImGui will use those indices to peek into the io.KeysDown[] array.
	i.io.KeyMap(imgui.KeyTab, int(glfw.KeyTab))
	i.io.KeyMap(imgui.KeyLeftArrow, int(glfw.KeyLeft))
	i.io.KeyMap(imgui.KeyRightArrow, int(glfw.KeyRight))
	i.io.KeyMap(imgui.KeyUpArrow, int(glfw.KeyUp))
	i.io.KeyMap(imgui.KeyDownArrow, int(glfw.KeyDown))
	i.io.KeyMap(imgui.KeyPageUp, int(glfw.KeyPageUp))
	i.io.KeyMap(imgui.KeyPageDown, int(glfw.KeyPageDown))
	i.io.KeyMap(imgui.KeyHome, int(glfw.KeyHome))
	i.io.KeyMap(imgui.KeyEnd, int(glfw.KeyEnd))
	i.io.KeyMap(imgui.KeyInsert, int(glfw.KeyInsert))
	i.io.KeyMap(imgui.KeyDelete, int(glfw.KeyDelete))
	i.io.KeyMap(imgui.KeyBackspace, int(glfw.KeyBackspace))
	i.io.KeyMap(imgui.KeySpace, int(glfw.KeySpace))
	i.io.KeyMap(imgui.KeyEnter, int(glfw.KeyEnter))
	i.io.KeyMap(imgui.KeyEscape, int(glfw.KeyEscape))
	i.io.KeyMap(imgui.KeyA, int(glfw.KeyA))
	i.io.KeyMap(imgui.KeyC, int(glfw.KeyC))
	i.io.KeyMap(imgui.KeyV, int(glfw.KeyV))
	i.io.KeyMap(imgui.KeyX, int(glfw.KeyX))
	i.io.KeyMap(imgui.KeyY, int(glfw.KeyY))
	i.io.KeyMap(imgui.KeyZ, int(glfw.KeyZ))
}

var glfwButtonIndexByID = map[glfw.MouseButton]int{
	glfw.MouseButton1: 0,
	glfw.MouseButton2: 1,
	glfw.MouseButton3: 2,
}

var glfwButtonIDByIndex = map[int]glfw.MouseButton{
	0: glfw.MouseButton1,
	1: glfw.MouseButton2,
	2: glfw.MouseButton3,
}

func (i *ImGUIModule) MouseScrollChange(x, y float64) bool {
	if !i.wantMouse {
		return false
	}

	i.io.AddMouseWheelDelta(float32(x), float32(y))

	return true
}
func (i *ImGUIModule) MouseButtonChange(rawButton glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) bool {
	if !i.wantMouse {
		return false
	}

	buttonIndex, known := glfwButtonIndexByID[rawButton]

	if known && (action == glfw.Press) {
		i.mouseJustPressed[buttonIndex] = true
	}

	return true
}

func (i *ImGUIModule) CharChange(char rune) bool {
	if !i.wantKeyboard {
		return false
	}
	i.io.AddInputCharacters(string(char))
	return true
}
