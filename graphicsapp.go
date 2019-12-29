package vkg

import (
	"fmt"
	"log"
	"sync"

	"github.com/vulkan-go/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

const StopCmdBufferConstruction = -1

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

	MaxFramesInFlight int
	CurrentFrame      int

	DefaultNumSwapchainImages int

	presentCompleteSemaphore vk.Semaphore
	renderCompleteSemaphore  vk.Semaphore
	waitFences               []vk.Fence

	screenExtent vk.Extent2D

	InFlightFences []vk.Fence
	ImagesInFlight []vk.Fence

	//Semaphore per frame
	ImageAvailableSemaphore []vk.Semaphore
	RenderFinishedSemaphore []vk.Semaphore

	//This channel is used to signal that we want to
	// start command buffer construction for the specified
	// frame
	makeCmdBuffer chan int

	// This channel is used to signal that the last
	// frame we wanted a command buffer generated
	// for is ready
	cmdBufferReady chan bool

	// Mutex used to indicate that
	// concurrent command generation is in progress
	cmdGeneration sync.Mutex

	Swapchain           *Swapchain
	SwapchainImages     []*Image
	SwapchainImageViews []*ImageView
	DepthImage          *BoundImage
	DepthImageView      *ImageView
	Framebuffers        []vk.Framebuffer

	resized bool

	VKRenderPass vk.RenderPass

	// ConfigureRenderPass is a call back which can be supplied to
	// allow for custimization of the render pass
	ConfigureRenderPass func(renderPass vk.RenderPassCreateInfo)
	MakeCommandBuffer   func(command *CommandBuffer, frame int)
}

func NewApp(name string, version Version) (*GraphicsApp, error) {
	app := &App{Name: name, Version: version}
	p := &GraphicsApp{
		App: app,
	}
	return p, nil
}

func (p *GraphicsApp) PhysicalDevices() ([]*PhysicalDevice, error) {
	if p.Instance == nil {
		return nil, fmt.Errorf("platform hasn't been initialized yet")
	}
	return p.Instance.PhysicalDevices()
}

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

func (p *GraphicsApp) CreateGraphicsPipelineConfig() *GraphicsPipelineConfig {
	return p.Device.CreateGraphicsPipelineConfig()
}

func (p *GraphicsApp) AddGraphicsPipelineConfig(name string, config IGraphicsPipelineConfig) {
	if p.GraphicsPipelineConfigs == nil {
		p.GraphicsPipelineConfigs = make(map[string]IGraphicsPipelineConfig)
	}
	p.GraphicsPipelineConfigs[name] = config
}

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

func (p *GraphicsApp) SupportedExtensions() ([]string, error) {
	return SupportedExtensions()
}

func (p *GraphicsApp) SupportedLayers() ([]string, error) {
	return SupportedLayers()
}

func (p *GraphicsApp) EnableDebugging() bool {
	if p.Instance != nil {
		log.Printf("debugging must be enabled prior to initialization")
		return false
	}

	return p.EnableLayer("VK_LAYER_LUNARG_parameter_validation") &&
		p.EnableLayer("VK_LAYER_LUNARG_core_validation") &&
		p.EnableLayer("VK_LAYER_GOOGLE_threading") &&
		p.EnableLayer("VK_LAYER_LUNARG_standard_validation") &&
		p.EnableExtension("VK_EXT_debug_utils") &&
		p.EnableExtension("VK_EXT_debug_report")

}

func (p *GraphicsApp) NumFramebuffers() int {
	return p.DefaultNumSwapchainImages
}

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

	p.makeCmdBuffer = make(chan int)
	p.cmdBufferReady = make(chan bool)

	go func() {
		for {
			nextImage := <-p.makeCmdBuffer

			if nextImage == StopCmdBufferConstruction {
				return
			}

			p.cmdGeneration.Lock()
			log.Printf("generating buffer for %d", nextImage)
			p.MakeCommandBuffer(p.GraphicsCommandBuffers[nextImage], nextImage)
			log.Printf("done generating buffer")
			p.cmdGeneration.Unlock()
			p.cmdBufferReady <- true
		}
	}()

	return nil

}

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

func (p *GraphicsApp) PrepareToDraw() error {
	var err error
	p.cmdGeneration.Lock()
	err = p.prepareToDraw()
	p.cmdGeneration.Unlock()
	if err != nil {
		return err
	}
	p.fillCmdBuffers()
	return nil
}

func (p *GraphicsApp) prepareToDraw() error {
	var err error

	//FIXME allow the user to choose, but also choose an appropriate number
	if p.MaxFramesInFlight == 0 {
		p.MaxFramesInFlight = 2
	}

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

	return nil

}

func (p *GraphicsApp) fillCmdBuffers() {

	// Consume any existing ready indicators
	select {
	case <-p.cmdBufferReady:
	default:
	}

	for i := 0; i < p.NumFramebuffers(); i++ {
		p.makeCmdBuffer <- i
		<-p.cmdBufferReady
	}

}

func (p *GraphicsApp) recreateSwapchain() error {
	log.Printf("reseting swap chain")
	// First go ahead and consume any
	// ready indicators
	select {
	case <-p.cmdBufferReady:
	default:
		log.Printf("consumed ready bit")
	}

	for i, _ := range p.GraphicsCommandBuffers {
		log.Printf("command buffer fence %d status %v", i, vk.GetFenceStatus(p.Device.VKDevice, p.waitFences[i]))
	}

	p.cmdGeneration.Lock()

	log.Printf("waiting for idle queue and device")
	vk.DeviceWaitIdle(p.Device.VKDevice)

	//p.OldSwapchain = p.Swapchain
	log.Printf("destroying swap chain and images")
	p.destroySwapchainAndImages()

	log.Printf("preparing to draw")
	p.prepareToDraw()

	p.cmdGeneration.Unlock()

	log.Printf("filling command buffers")
	p.fillCmdBuffers()

	return nil
}

func (p *GraphicsApp) getNextFrameToMakeCmdBufferFor(currentFrame int) int {
	l := p.NumFramebuffers()
	c := (int(currentFrame) + 1) % l
	return c
}

func (p *GraphicsApp) destroySwapchainAndImages() {
	log.Println("Destroying framebuffers")
	for _, c := range p.GraphicsCommandBuffers {
		c.ResetAndRelease()
	}

	for i, _ := range p.Framebuffers {
		vk.DestroyFramebuffer(p.Device.VKDevice, p.Framebuffers[i], nil)
	}
	p.Framebuffers = nil

	log.Println("Destroying command buffers")
	for _, b := range p.GraphicsCommandBuffers {
		p.GraphicsCommandPool.FreeBuffer(b)
	}
	p.GraphicsCommandBuffers = nil

	log.Println("Destroying graphics piplines")
	p.destroyGraphicsPipelines()

	log.Println("Destroying render pass")
	vk.DestroyRenderPass(p.Device.VKDevice, p.VKRenderPass, nil)

	log.Println("Destroying swap chain images")
	for _, views := range p.SwapchainImageViews {
		views.Destroy()
	}
	p.SwapchainImageViews = nil

	p.Swapchain.Destroy()

}

func (p *GraphicsApp) Resize() {
	p.refreshScreenExtent()
	p.resized = true
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

	res := vk.AcquireNextImage(p.Device.VKDevice, p.Swapchain.VKSwapchain, vk.MaxUint64, p.presentCompleteSemaphore, vk.NullFence, &imageIndex)

	if res == vk.ErrorOutOfDate || p.resized {
		p.resized = false
		p.refreshScreenExtent()
		p.recreateSwapchain()
		return nil
	} else {
		err = vk.Error(res)

		if err != nil {
			return err
		}
	}

	// Wait for the command buffer associated with the image to be ready
	vk.WaitForFences(p.Device.VKDevice, 1, []vk.Fence{p.waitFences[imageIndex]}, vk.True, vk.MaxUint64)
	vk.ResetFences(p.Device.VKDevice, 1, []vk.Fence{p.waitFences[imageIndex]})

	p.GraphicsCommandBuffers[int(imageIndex)].Reset()

	p.MakeCommandBuffer(p.GraphicsCommandBuffers[int(imageIndex)], int(imageIndex))

	waitSemaphores := []vk.Semaphore{p.presentCompleteSemaphore}
	signalSemaphores := []vk.Semaphore{p.renderCompleteSemaphore}
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

	err = vk.Error(vk.QueueSubmit(p.GraphicsQueue.VKQueue, 1, submitInfo, p.waitFences[imageIndex]))

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
		p.resized = false
		p.refreshScreenExtent()
		p.recreateSwapchain()
	} else {
		err = vk.Error(res)

		if err != nil {
			return err
		}
	}

	return nil
}

func (p *GraphicsApp) destroyGraphicsPipelines() {
	for _, g := range p.GraphicsPipelines {
		vk.DestroyPipeline(p.Device.VKDevice, g, nil)
	}
	p.GraphicsPipelines = nil
}

func (p *GraphicsApp) createGraphicsPipelines() error {

	configs := make([]vk.GraphicsPipelineCreateInfo, len(p.GraphicsPipelineConfigs))
	nameToId := make(map[string]int)
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
		nameToId[name] = i
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
	for name, _ := range p.GraphicsPipelineConfigs {
		p.GraphicsPipelines[name] = graphicsPipelines[nameToId[name]]
	}

	return nil
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

func (p *GraphicsApp) GetScreenExtent() vk.Extent2D {
	return p.screenExtent
}

func (p *GraphicsApp) Destroy() {

	p.makeCmdBuffer <- StopCmdBufferConstruction

	vk.DeviceWaitIdle(p.Device.VKDevice)

	p.destroySwapchainAndImages()

	for i := 0; i < p.MaxFramesInFlight; i++ {
		vk.DestroySemaphore(p.Device.VKDevice, p.RenderFinishedSemaphore[i], nil)
		vk.DestroySemaphore(p.Device.VKDevice, p.ImageAvailableSemaphore[i], nil)
		vk.DestroyFence(p.Device.VKDevice, p.InFlightFences[i], nil)
	}

	p.GraphicsCommandPool.Destroy()

	p.Device.Destroy()

}

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

func (p *GraphicsApp) createDepthImage() error {
	var err error

	//FIXME find correct format
	p.DepthImage, err = p.Device.CreateBoundImage(p.Swapchain.Extent, vk.FormatD32Sfloat, vk.ImageTilingOptimal, vk.ImageUsageFlags(vk.ImageUsageDepthStencilAttachmentBit), vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit))
	if err != nil {
		return err
	}

	p.DepthImageView, err = p.DepthImage.CreateImageViewWithAspectMask(vk.ImageAspectFlags(vk.ImageAspectDepthBit))
	if err != nil {
		return err
	}

	return err
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

func (p *GraphicsApp) createCommandBuffers() error {
	var err error
	p.GraphicsCommandBuffers = make([]*CommandBuffer, len(p.SwapchainImageViews))
	for i, _ := range p.SwapchainImageViews {
		p.GraphicsCommandBuffers[i], err = p.GraphicsCommandPool.AllocateBuffer()
		if err != nil {
			return err
		}
	}
	return nil

}

func (p *GraphicsApp) createSyncObjects() error {
	var err error

	p.presentCompleteSemaphore, _ = p.Device.VKCreateSemaphore()
	p.renderCompleteSemaphore, _ = p.Device.VKCreateSemaphore()

	p.waitFences = make([]vk.Fence, p.NumFramebuffers())
	for i := 0; i < p.NumFramebuffers(); i++ {
		p.waitFences[i], _ = p.Device.VKCreateFence(true)
	}

	p.ImagesInFlight = make([]vk.Fence, len(p.SwapchainImages))
	p.InFlightFences = make([]vk.Fence, p.MaxFramesInFlight)

	p.ImageAvailableSemaphore = make([]vk.Semaphore, p.MaxFramesInFlight)
	p.RenderFinishedSemaphore = make([]vk.Semaphore, p.MaxFramesInFlight)

	for i := 0; i < p.MaxFramesInFlight; i++ {
		p.InFlightFences[i], err = p.Device.VKCreateFence(true)
		if err != nil {
			return err
		}

		p.ImageAvailableSemaphore[i], err = p.Device.VKCreateSemaphore()
		if err != nil {
			return err
		}

		p.RenderFinishedSemaphore[i], err = p.Device.VKCreateSemaphore()
		if err != nil {
			return err
		}

	}
	return nil

}
