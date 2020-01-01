package main

import (
	"os"

	"github.com/celer/vkg/examples/imgui/app"
	"github.com/celer/vkg/examples/imgui/cube"
	gui "github.com/celer/vkg/examples/imgui/imgui"

	"github.com/inkyblackness/imgui-go"
)

type AppUI struct {
	showDemoWindow bool
}

func (a *AppUI) DrawUI() {

	if a.showDemoWindow {
		imgui.ShowDemoWindow(&a.showDemoWindow)
	}

	if imgui.BeginMainMenuBar() {

		if imgui.BeginMenu("File") {
			if imgui.MenuItem("Show Demo Window") {
				a.showDemoWindow = true
			}
			if imgui.MenuItem("Exit") {
				os.Exit(0)
			}
			imgui.EndMenu()
		}

		imgui.EndMainMenuBar()
	}
}

func main() {

	b, err := app.NewAppBase("imgui", 800, 600)
	if err != nil {
		panic(err)
	}

	err = b.Init()
	if err != nil {
		panic(err)
	}

	i, err := gui.NewImGUIModule(b, b.Window)
	if err != nil {
		panic(err)
	}

	c, err := cube.NewCubeModule(b, i)
	if err != nil {
		panic(err)
	}

	b.AddGraphicsModule(c)
	b.AddGraphicsModule(i)
	b.AddInputModule(i)

	i.AddUI(&AppUI{})

	err = b.PrepareToDraw()
	if err != nil {
		panic(err)
	}

	for !b.ShouldClose() {
		b.NewFrame()

		err = b.DrawFrameSync()
		if err != nil {
			panic(err)
		}

		b.PostFrame()
	}
	b.Destroy()

}
