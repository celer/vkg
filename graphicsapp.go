package vkg

import (
	"fmt"
	"log"

	"github.com/vulkan-go/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

var FrameLag = 3

// StopCmdBufferConstruction is used to signal that command buffer construction should stop
const StopCmdBufferConstruction = -1

// GraphicsApp is a utility object which implements many of the core requirements to
// get to a functioning Vulkan app. It will setup the appropriate devices and do many
// of the necissary preprations to begin drawing.
//
// See https://vulkan-tutorial.com/ for a good walkthrough of what this code does.
type GraphicsApp struct {
	Instance *Instance
	App      *App

	Window    *glfw.Window
	VKSurface vk.Surface

	Device         *Device
	PhysicalDevice *PhysicalDevice

	CommandBuffers          []*CommandBuffer
	GraphicsPipelineConfigs map[string]IGraphicsPipelineConfig

	// Generated from GraphicsPipelineConfigs
	GraphicsPipelines map[string]vk.Pipeline

	ResourceManager *ResourceManager

	GraphicsQueue *Queue
	PresentQueue  *Queue
	PipelineCache *PipelineCache

	GraphicsCommandPool    *CommandPool
	GraphicsCommandBuffers []*CommandBuffer

	DefaultNumSwapchainImages int

	presentCompleteSemaphore []vk.Semaphore
	renderCompleteSemaphore  []vk.Semaphore
	waitFences               []vk.Fence

	frameIndex int

	screenExtent vk.Extent2D

	Swapchain           *Swapchain
	SwapchainImages     []*Image
	SwapchainImageViews []*ImageView
	DepthImage          *ImageResource
	DepthImageView      *ImageView
	Framebuffers        []vk.Framebuffer

	resized bool

	VKRenderPass vk.RenderPass

	// ConfigureRenderPass is a call back which can be supplied to
	// allow for custimization of the render pass
	ConfigureRenderPass func(renderPass vk.RenderPassCreateInfo)
	MakeCommandBuffer   func(command *CommandBuffer, frame int)
}

// NewGraphicsApp creates a new graphics app with the given name and version
func NewGraphicsApp(name string, version Version) (*GraphicsApp, error) {
	app := &App{Name: name, Version: version}
	p := &GraphicsApp{
		App: app,
	}
	return p, nil
}

// PhysicalDevices returns a list of physical devices
func (p *GraphicsApp) PhysicalDevices() ([]*PhysicalDevice, error) {
	if p.Instance == nil {
		return nil, fmt.Errorf("platform hasn't been initialized yet")
	}
	return p.Instance.PhysicalDevices()
}

// EnableLayer enables a specific layer of the code
func (p *GraphicsApp) EnableLayer(layer string) bool {
	supportedLayers, err := p.SupportedLayers()
	if err != nil {
		return false
	}

	for _, slayer := range supportedLayers {
		if layer == slayer {
			p.App.EnableLayer(layer)
			return true
		}

	}
	return false
}

// CreateGraphicsPipelineConfig creates a graphic pipeline configuration for customization
func (p *GraphicsApp) CreateGraphicsPipelineConfig() *GraphicsPipelineConfig {
	return p.Device.CreateGraphicsPipelineConfig()
}

// AddGraphicsPipelineConfig adds this graphic pipeline config back into the app
func (p *GraphicsApp) AddGraphicsPipelineConfig(name string, config IGraphicsPipelineConfig) {
	if p.GraphicsPipelineConfigs == nil {
		p.GraphicsPipelineConfigs = make(map[string]IGraphicsPipelineConfig)
	}
	p.GraphicsPipelineConfigs[name] = config
}

// EnableExtension enables a specific extension
func (p *GraphicsApp) EnableExtension(extension string) bool {
	supportedExtensions, err := p.SupportedExtensions()
	if err != nil {
		return false
	}

	for _, sextension := range supportedExtensions {
		if extension == sextension {
			p.App.EnableExtension(extension)
			return true
		}

	}
	return false
}

// SupportedExtensions returns alist of supported extensions
func (p *GraphicsApp) SupportedExtensions() ([]string, error) {
	return SupportedExtensions()
}

// SupportedLayers returns a list of supported layers
func (p *GraphicsApp) SupportedLayers() ([]string, error) {
	return SupportedLayers()
}

// EnableDebugging enables a list of commonly used debugging layers
func (p *GraphicsApp) EnableDebugging() bool {
	if p.Instance != nil {
		return false
	}
	p.App.EnableDebugging()
	return true
}

// NumFramebuffers returns the number of framebuffers that have been created
func (p *GraphicsApp) NumFramebuffers() int {
	return p.DefaultNumSwapchainImages
}

// Init initializes the graphics app
func (p *GraphicsApp) Init() error {
	var initSwapchain bool

	if p.Window != nil {
		initSwapchain = true
	}

	var err error

	p.Instance, err = p.App.CreateInstance()
	if err != nil {
		return err
	}

	if p.Window != nil && p.VKSurface == vk.NullSurface {
		surface, err := p.Window.CreateWindowSurface(p.Instance.VKInstance, nil)
		if err != nil {
			return err
		}
		p.VKSurface = vk.SurfaceFromPointer(surface)
	}

	physicalDevices, err := p.Instance.PhysicalDevices()
	if err != nil {
		return fmt.Errorf("error getting devices: %w", err)
	}

	if physicalDevices == nil && err == nil {
		return fmt.Errorf("no devices found")
	}

	//FIXME this should probably be smarter than this
	pdevice := physicalDevices[0]

	queues, err := pdevice.QueueFamilies()
	if err != nil {
		return fmt.Errorf("unable to load device queue families: %w", err)
	}

	gqueues := queues.FilterGraphicsAndPresent(p.VKSurface)

	if len(gqueues) == 0 {
		return fmt.Errorf("no graphics capable queues found on device: %v", pdevice)
	}

	enabledExtensions := []string{}
	if initSwapchain {
		enabledExtensions = []string{"VK_KHR_swapchain"}
	}

	ldevice, err := pdevice.CreateLogicalDeviceWithOptions(gqueues, &CreateDeviceOptions{
		EnabledExtensions: enabledExtensions,
	})

	if err != nil {
		return fmt.Errorf("unable to create device: %w", err)
	}

	p.Device = ldevice
	p.PhysicalDevice = pdevice

	if len(gqueues) == 1 {
		// Single graphics and present queue
		queue := ldevice.GetQueue(gqueues[0])

		p.GraphicsQueue = queue
		p.PresentQueue = queue
	} else {
		//Seperate graphics and present queue
		pq := gqueues.FilterPresent(p.VKSurface)
		gq := gqueues.FilterGraphics()

		p.GraphicsQueue = ldevice.GetQueue(gq[0])
		p.PresentQueue = ldevice.GetQueue(pq[0])
	}

	p.DefaultNumSwapchainImages, err = p.Device.DefaultNumSwapchainImages(p.VKSurface)
	if err != nil {
		return err
	}

	p.GraphicsCommandPool, err = p.Device.CreateCommandPool(p.GraphicsQueue.QueueFamily)
	if err != nil {
		return err
	}

	p.ResourceManager = p.Device.CreateResourceManager()

	return nil

}

// SetWindow sets the GLFW window for the graphics app
func (p *GraphicsApp) SetWindow(window *glfw.Window) error {

	if p.Instance != nil {
		return fmt.Errorf("window must be set prior to initalizatin")
	}

	p.Window = window

	extensions := p.Window.GetRequiredInstanceExtensions()

	for _, ext := range extensions {
		if !p.EnableExtension(ext) {
			return fmt.Errorf("extension '%s' required to enable glfw is not supported by vulkan", ext)
		}
	}

	p.refreshScreenExtent()

	return nil

}

// PrepareToDraw creates the nescissary objects required to start drawing, it must be called after Init is called and after MakeCommandBuffers is set
func (p *GraphicsApp) PrepareToDraw() error {
	var err error
	err = p.prepareToDraw()
	if err != nil {
		return err
	}
	p.fillCmdBuffers()
	return nil
}

func (p *GraphicsApp) prepareToDraw() error {
	var err error

	if p.MakeCommandBuffer == nil {
		return fmt.Errorf("no function to make command buffers has been configured")
	}

	err = p.createSwapchainAndImages()
	if err != nil {
		return err
	}

	err = p.createRenderer()
	if err != nil {
		return err
	}

	if p.PipelineCache != nil {
		p.PipelineCache.Destroy()
	}

	p.PipelineCache, err = p.Device.CreatePipelineCache()
	if err != nil {
		return err
	}

	err = p.createGraphicsPipelines()
	if err != nil {
		return err
	}

	err = p.createDepthImage()
	if err != nil {
		return err
	}

	err = p.createFramebuffers()
	if err != nil {
		return err
	}

	err = p.createCommandBuffers()
	if err != nil {
		return err
	}

	err = p.createSyncObjects()
	if err != nil {
		return err
	}

	p.frameIndex = 0

	return nil

}

func (p *GraphicsApp) resize(i int) {
	//FIXME minimization

	p.PresentQueue.WaitIdle()
	p.GraphicsQueue.WaitIdle()
	p.Device.WaitIdle()

	p.destroyFramebuffers()
	p.destroyDepthImage()

	for _, c := range p.GraphicsCommandBuffers {
		p.GraphicsCommandPool.FreeBuffer(c)
	}
	p.destroyGraphicsPipelines()
	p.destroyRenderer()

	for _, views := range p.SwapchainImageViews {
		views.Destroy()
	}

	p.Swapchain.Destroy()

	p.refreshScreenExtent()

	p.createSwapchainAndImages()
	p.createRenderer()
	p.createGraphicsPipelines()
	p.createDepthImage()
	p.createFramebuffers()
	p.createCommandBuffers()
	p.fillCmdBuffers()

	p.resized = false

	p.frameIndex = 0
}

func (p *GraphicsApp) unprepareToDraw() {

	vk.WaitForFences(p.Device.VKDevice, uint32(len(p.waitFences)), p.waitFences, vk.True, vk.MaxUint64)

	p.destroyCommandBuffers()

	p.destroySyncObjects()

	p.destroyFramebuffers()

	p.destroyDepthImage()

	p.destroyGraphicsPipelines()

	p.PipelineCache.Destroy()
	p.PipelineCache = nil

	p.destroyRenderer()
	p.destroySwapchainAndImages()

}

func (p *GraphicsApp) fillCmdBuffers() {
	for i := range p.GraphicsCommandBuffers {
		p.MakeCommandBuffer(p.GraphicsCommandBuffers[i], i)
	}
}

func (p *GraphicsApp) recreateSwapchain() error {

	p.unprepareToDraw()

	p.prepareToDraw()

	p.fillCmdBuffers()

	return nil
}

func (p *GraphicsApp) getNextFrameToMakeCmdBufferFor(currentFrame int) int {
	l := p.NumFramebuffers()
	c := (int(currentFrame) + 1) % l
	return c
}

// Resize is used to signal that we need to resize
func (p *GraphicsApp) Resize() {
	p.refreshScreenExtent()
	p.resized = true
}

func (p *GraphicsApp) printFenceStatus() int {
	signaled := 0
	for i := 0; i < FrameLag; i++ {
		s := p.Device.VKGetFenceStatus(p.waitFences[i]) == vk.Success
		if s {
			signaled += 1
		}
		log.Printf("%d FenceStatus %v", i, s)

	}
	log.Printf("\n")
	return signaled
}

func (p *GraphicsApp) clearFenceStatus() {
	for i := 0; i < FrameLag; i++ {
		vk.ResetFences(p.Device.VKDevice, 1, []vk.Fence{p.waitFences[i]})
	}

}

// DrawFrameSync draws one frame at a time to the GPU it does not utilize the GPU
// particularly well but simplifies application development because it insures that
// resources utilized by the created command buffers are not also utilized currently
// by the GPU. The specified call back is utilized to populate the command buffer.
//
// The 'frame' parameter is provided so that the application may utilize multiple
// buffers if desired, although it should not be needed with this frame drawing method
//
// See https://vulkan-tutorial.com/Uniform_buffers/Descriptor_layout_and_buffer for a discussion
// of how memory usage and frame drawing might require a more complex resource allocation
// approach
//
func (p *GraphicsApp) DrawFrameSync() error {
	var imageIndex uint32
	var err error

	res := vk.AcquireNextImage(p.Device.VKDevice, p.Swapchain.VKSwapchain, vk.MaxUint64, p.presentCompleteSemaphore[p.frameIndex], vk.NullFence, &imageIndex)

	if res == vk.ErrorOutOfDate || p.resized {
		p.resize(1)
		return nil
	}
	err = vk.Error(res)

	if err != nil {
		return err
	}

	vk.WaitForFences(p.Device.VKDevice, 1, []vk.Fence{p.waitFences[p.frameIndex]}, vk.True, vk.MaxUint64)
	vk.ResetFences(p.Device.VKDevice, 1, []vk.Fence{p.waitFences[p.frameIndex]})

	p.GraphicsCommandBuffers[int(imageIndex)].Reset()
	p.MakeCommandBuffer(p.GraphicsCommandBuffers[int(imageIndex)], int(imageIndex))

	waitSemaphores := []vk.Semaphore{p.presentCompleteSemaphore[p.frameIndex]}
	signalSemaphores := []vk.Semaphore{p.renderCompleteSemaphore[p.frameIndex]}
	waitStages := []vk.PipelineStageFlags{vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit)}

	submitInfo := []vk.SubmitInfo{{
		SType:                vk.StructureTypeSubmitInfo,
		WaitSemaphoreCount:   1,
		PWaitSemaphores:      waitSemaphores,
		PWaitDstStageMask:    waitStages,
		SignalSemaphoreCount: 1,
		PSignalSemaphores:    signalSemaphores,
		CommandBufferCount:   1,
		PCommandBuffers:      []vk.CommandBuffer{p.GraphicsCommandBuffers[imageIndex].VKCommandBuffer},
	}}

	err = vk.Error(vk.QueueSubmit(p.GraphicsQueue.VKQueue, 1, submitInfo, p.waitFences[p.frameIndex]))
	if err != nil {
		return err
	}

	imageIndices := []uint32{imageIndex}
	presentInfo := vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{p.Swapchain.VKSwapchain},
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    signalSemaphores,
		PImageIndices:      imageIndices,
		PResults:           nil,
	}

	res = vk.QueuePresent(p.GraphicsQueue.VKQueue, &presentInfo)
	if res == vk.ErrorOutOfDate || res == vk.Suboptimal || p.resized {
		p.resize(2)
		return nil
	} else {
		err = vk.Error(res)

		if err != nil {
			return err
		}
	}

	p.frameIndex = int(imageIndex + 1)
	p.frameIndex %= FrameLag

	p.PresentQueue.WaitIdle()
	p.GraphicsQueue.WaitIdle()
	p.Device.WaitIdle()

	return nil
}

func (p *GraphicsApp) createGraphicsPipelines() error {

	configs := make([]vk.GraphicsPipelineCreateInfo, len(p.GraphicsPipelineConfigs))
	nameToID := make(map[string]int)
	i := 0

	if len(p.GraphicsPipelineConfigs) == 0 {
		return nil
	}

	for name, gconfig := range p.GraphicsPipelineConfigs {
		config, err := gconfig.VKGraphicsPipelineCreateInfo(p.GetScreenExtent())
		if err != nil {
			return fmt.Errorf("error generating graphics pipline config '%s' : %w", name, err)
		}
		config.RenderPass = p.VKRenderPass
		configs[i] = config
		nameToID[name] = i
		i++
	}

	graphicsPipelines := make([]vk.Pipeline, len(configs))
	err := vk.Error(vk.CreateGraphicsPipelines(p.Device.VKDevice, p.PipelineCache.VKPipelineCache,
		uint32(len(configs)),
		configs,
		nil,
		graphicsPipelines))

	if err != nil {
		return err
	}

	p.GraphicsPipelines = make(map[string]vk.Pipeline)
	for name := range p.GraphicsPipelineConfigs {
		p.GraphicsPipelines[name] = graphicsPipelines[nameToID[name]]
	}

	return nil
}

func (p *GraphicsApp) destroyGraphicsPipelines() {
	for _, g := range p.GraphicsPipelines {
		vk.DestroyPipeline(p.Device.VKDevice, g, nil)
	}
	p.GraphicsPipelines = nil
}

func (p *GraphicsApp) refreshScreenExtent() {
	if p.Window != nil {
		extent := vk.Extent2D{}
		width, height := p.Window.GetFramebufferSize()
		extent.Width = uint32(width)
		extent.Height = uint32(height)
		p.screenExtent = extent
	}

}

// GetScreenExtent gets the current screen extents
func (p *GraphicsApp) GetScreenExtent() vk.Extent2D {
	return p.screenExtent
}

// Destroy tears down the graphics application
func (p *GraphicsApp) Destroy() {

	vk.DeviceWaitIdle(p.Device.VKDevice)

	p.destroyGraphicsPipelines()

	for _, g := range p.GraphicsPipelineConfigs {
		g.Destroy()
	}

	if p.PipelineCache != nil {
		p.PipelineCache.Destroy()
	}

	p.ResourceManager.Destroy()

	p.destroyDepthImage()

	p.destroySwapchainAndImages()

	p.destroySyncObjects()

	p.GraphicsCommandPool.Destroy()

	vk.DestroySurface(p.Instance.VKInstance, p.VKSurface, nil)

	p.Device.Destroy()

	p.Instance.Destroy()

}

// VKRenderPassCreateInfo is a utility function which creates the render pass info, the implementing application
// can implement the ConfigureRenderPass function to customize the render pass
func (p *GraphicsApp) VKRenderPassCreateInfo() vk.RenderPassCreateInfo {
	attachmentDescriptions := []vk.AttachmentDescription{{
		Format:         p.Swapchain.Format,
		Samples:        vk.SampleCount1Bit,
		LoadOp:         vk.AttachmentLoadOpClear,
		StoreOp:        vk.AttachmentStoreOpStore,
		StencilLoadOp:  vk.AttachmentLoadOpDontCare,
		StencilStoreOp: vk.AttachmentStoreOpDontCare,
		InitialLayout:  vk.ImageLayoutUndefined,
		FinalLayout:    vk.ImageLayoutPresentSrc,
	},
		{
			Format:         vk.FormatD32Sfloat,
			Samples:        vk.SampleCount1Bit,
			LoadOp:         vk.AttachmentLoadOpClear,
			StoreOp:        vk.AttachmentStoreOpDontCare,
			StencilLoadOp:  vk.AttachmentLoadOpDontCare,
			StencilStoreOp: vk.AttachmentStoreOpDontCare,
			InitialLayout:  vk.ImageLayoutUndefined,
			FinalLayout:    vk.ImageLayoutDepthStencilAttachmentOptimal,
		},
	}

	depthAttachmentRef := vk.AttachmentReference{
		Attachment: 1,
		Layout:     vk.ImageLayoutDepthStencilAttachmentOptimal,
	}

	colorAttachments := []vk.AttachmentReference{{
		Attachment: 0,
		Layout:     vk.ImageLayoutColorAttachmentOptimal,
	}}

	subpassDescriptions := []vk.SubpassDescription{{
		PipelineBindPoint:       vk.PipelineBindPointGraphics,
		ColorAttachmentCount:    1,
		PColorAttachments:       colorAttachments,
		PDepthStencilAttachment: &depthAttachmentRef,
	}}

	dependency := vk.SubpassDependency{
		SrcSubpass:    vk.SubpassExternal,
		DstSubpass:    0,
		SrcStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		SrcAccessMask: 0,
		DstStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		DstAccessMask: vk.AccessFlags(vk.AccessColorAttachmentReadBit | vk.AccessColorAttachmentWriteBit),
	}

	renderPassCreateInfo := vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		AttachmentCount: 2,
		PAttachments:    attachmentDescriptions,
		SubpassCount:    1,
		PSubpasses:      subpassDescriptions,
		DependencyCount: 1,
		PDependencies:   []vk.SubpassDependency{dependency},
	}

	return renderPassCreateInfo

}

func (p *GraphicsApp) createRenderer() error {
	renderPassCreateInfo := p.VKRenderPassCreateInfo()

	if p.ConfigureRenderPass != nil {
		p.ConfigureRenderPass(renderPassCreateInfo)
	}

	var renderPass vk.RenderPass

	err := vk.Error(vk.CreateRenderPass(p.Device.VKDevice, &renderPassCreateInfo, nil, &renderPass))
	if err != nil {
		return err
	}

	p.VKRenderPass = renderPass

	return nil

}

func (p *GraphicsApp) destroyRenderer() {
	vk.DestroyRenderPass(p.Device.VKDevice, p.VKRenderPass, nil)
	p.VKRenderPass = vk.NullRenderPass
	return
}

func (p *GraphicsApp) createSwapchainAndImages() error {

	extent := p.GetScreenExtent()

	options := &CreateSwapchainOptions{
		ActualSize:                extent,
		DesiredNumSwapchainImages: p.DefaultNumSwapchainImages,
	}

	/*
		if p.OldSwapchain != nil {
			options = &CreateSwapchainOptions{}
			options.OldSwapchain = p.Swapchain
		}*/

	swapchain, err := p.Device.CreateSwapchain(p.VKSurface, p.GraphicsQueue, p.PresentQueue, options)
	if err != nil {
		return err
	}
	p.Swapchain = swapchain

	/*
		if p.OldSwapchain != nil {
			p.OldSwapchain.Destroy()
			p.OldSwapchain = nil
		}*/

	images, err := swapchain.GetImages()
	if err != nil {
		return err
	}
	p.SwapchainImages = images

	p.SwapchainImageViews = make([]*ImageView, len(images))
	for i, image := range images {
		view, err := image.CreateImageView()
		if err != nil {
			return err
		}
		p.SwapchainImageViews[i] = view
	}
	return nil
}

func (p *GraphicsApp) destroySwapchainAndImages() {

	for _, views := range p.SwapchainImageViews {
		views.Destroy()
	}
	p.SwapchainImageViews = nil

	/*
		for _, image := range p.SwapchainImages {
			image.Destroy()
		}
		p.SwapchainImages = nil*/

	p.Swapchain.Destroy()

}

func (p *GraphicsApp) createDepthImage() error {
	var err error

	p.DepthImage, err = p.ResourceManager.NewImageResourceWithOptions(p.Swapchain.Extent, vk.FormatD32Sfloat, vk.ImageTilingOptimal, vk.ImageUsageDepthStencilAttachmentBit, vk.SharingModeExclusive, vk.MemoryPropertyDeviceLocalBit)

	p.DepthImageView, err = p.DepthImage.CreateImageViewWithAspectMask(vk.ImageAspectFlags(vk.ImageAspectDepthBit))
	if err != nil {
		return err
	}

	return err
}

func (p *GraphicsApp) destroyDepthImage() error {
	p.DepthImage.Destroy()
	p.DepthImageView.Destroy()
	return nil
}

func (p *GraphicsApp) createFramebuffers() error {
	p.Framebuffers = make([]vk.Framebuffer, len(p.SwapchainImageViews))
	for i, view := range p.SwapchainImageViews {
		attachments := []vk.ImageView{
			view.VKImageView,
			p.DepthImageView.VKImageView,
		}
		fbCreateInfo := vk.FramebufferCreateInfo{
			SType:           vk.StructureTypeFramebufferCreateInfo,
			RenderPass:      p.VKRenderPass,
			Layers:          1,
			AttachmentCount: uint32(len(attachments)),
			PAttachments:    attachments,
			Width:           p.Swapchain.Extent.Width,
			Height:          p.Swapchain.Extent.Height,
		}
		err := vk.Error(vk.CreateFramebuffer(p.Device.VKDevice, &fbCreateInfo, nil, &p.Framebuffers[i]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *GraphicsApp) destroyFramebuffers() {
	for i := range p.Framebuffers {
		vk.DestroyFramebuffer(p.Device.VKDevice, p.Framebuffers[i], nil)
	}
	p.Framebuffers = nil
}

func (p *GraphicsApp) createCommandBuffers() error {
	var err error
	p.GraphicsCommandBuffers = make([]*CommandBuffer, len(p.SwapchainImageViews))
	for i := range p.SwapchainImageViews {
		p.GraphicsCommandBuffers[i], err = p.GraphicsCommandPool.AllocateBuffer(vk.CommandBufferLevelPrimary)
		if err != nil {
			return err
		}
	}
	return nil

}

func (p *GraphicsApp) destroyCommandBuffers() {
	for _, c := range p.GraphicsCommandBuffers {
		p.GraphicsCommandPool.FreeBuffer(c)
	}
}

func (p *GraphicsApp) destroySyncObjects() error {

	for i := 0; i < FrameLag; i++ {
		p.Device.VKDestroySemaphore(p.presentCompleteSemaphore[i])
		p.Device.VKDestroySemaphore(p.renderCompleteSemaphore[i])
	}

	for _, fence := range p.waitFences {
		p.Device.VKDestroyFence(fence)
	}

	return nil

}

func (p *GraphicsApp) createSyncObjects() error {

	p.presentCompleteSemaphore = make([]vk.Semaphore, FrameLag)
	p.renderCompleteSemaphore = make([]vk.Semaphore, FrameLag)

	for i := 0; i < FrameLag; i++ {
		p.presentCompleteSemaphore[i], _ = p.Device.VKCreateSemaphore()
		p.renderCompleteSemaphore[i], _ = p.Device.VKCreateSemaphore()
	}

	p.waitFences = make([]vk.Fence, FrameLag)
	for i := 0; i < FrameLag; i++ {
		p.waitFences[i], _ = p.Device.VKCreateFence(true)
	}

	return nil

}
