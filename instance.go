package vkg

import (
	"fmt"
	"log"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

// InitializeCompute initializes Vulkan for a compute based task, it doesn't
// enable any graphics capabilties.
func InitializeForComputeOnly() error {
	err := vk.SetDefaultGetInstanceProcAddr()
	if err != nil {
		return err
	}
	err = vk.Init()
	if err != nil {
		return err
	}
	return nil
}

// Version is used to specify versions of components
type Version struct {
	Major int
	Minor int
	Patch int
}

// VKVersion returns a Vulkan compatible version representation
func (v *Version) VKVersion() uint32 {
	return vk.MakeVersion(v.Major, v.Minor, v.Patch)
}

// App is used to provide information about this specific application to Vulkan
type App struct {
	// Name the name of the application
	Name string
	// Engine the name of the engine associated with the application
	EngineName string
	// Version the version of the application
	Version Version
	// APIVersion the expected minimum version of the Vulkan API (i.e. 1.0.0)
	APIVersion Version

	// EnabledLayers the enabled layers
	EnabledLayers []string

	// EnabledExtensions the enabled extensions
	EnabledExtensions []string
}

// SupportedLayers returns a list of supported layers for use by Vulkan
// this may crash if Vulkan has not been initialized previously for use in a compute, or graphics capability
// of some kind
func SupportedLayers() ([]string, error) {
	var instanceLayerLen uint32
	err := vk.Error(vk.EnumerateInstanceLayerProperties(&instanceLayerLen, nil))
	if err != nil {
		return nil, err
	}
	instanceLayer := make([]vk.LayerProperties, instanceLayerLen)
	err = vk.Error(vk.EnumerateInstanceLayerProperties(&instanceLayerLen, instanceLayer))
	if err != nil {
		return nil, err
	}
	layerNames := make([]string, 0)
	for _, layer := range instanceLayer {
		layer.Deref()
		layerNames = append(layerNames,
			vk.ToString(layer.LayerName[:]))
	}
	return layerNames, nil
	return nil, nil
}

// SupportedExtensions returns a list of supported extensions for use by Vulkan
// this may crash if Vulkan has not been initialized previously for use in a compute, or graphics capability
// of some kind
func SupportedExtensions() ([]string, error) {
	var instanceExtLen uint32
	err := vk.Error(vk.EnumerateInstanceExtensionProperties("", &instanceExtLen, nil))
	if err != nil {
		return nil, err
	}
	instanceExt := make([]vk.ExtensionProperties, instanceExtLen)
	err = vk.Error(vk.EnumerateInstanceExtensionProperties("", &instanceExtLen, instanceExt))
	if err != nil {
		return nil, err
	}
	extNames := make([]string, 0)
	for _, ext := range instanceExt {
		ext.Deref()
		extNames = append(extNames,
			vk.ToString(ext.ExtensionName[:]))
	}
	return extNames, nil
}

/*
	VK_LAYER_GOOGLE_threading - check validity of multi-threaded API usage
	VK_LAYER_KHRONOS_validation -
	VK_LAYER_LUNARG_api_dump - print API calls and their parameters and values
	VK_LAYER_LUNARG_core_validation - validate the descriptor set, pipeline state, and dynamic state; validate the interfaces between SPIR-V modules and the graphics pipeline; track and validate GPU memory and its binding to objects and command buffers
	VK_LAYER_LUNARG_device_limits - validate that app properly queries features and obeys feature limitations
	VK_LAYER_LUNARG_device_simulation -
	VK_LAYER_LUNARG_image - validate texture formats and render target formats
	VK_LAYER_LUNARG_monitor -
	VK_LAYER_LUNARG_object_tracker - track all Vulkan objects and flag invalid objects and object memory leaks
	VK_LAYER_LUNARG_screenshot - meta layer which loads all other validation layers
	VK_LAYER_LUNARG_standard_validation - meta layer which loads all other validation layers
	VK_LAYER_LUNARG_swapchain - validate the use of the WSI "swapchain" extensions
	VK_LAYER_LUNARG_vktrace
	VK_LAYER_KHRONOS_validation - The main, comprehensive Khronos validation layer. Vulkan is an Explicit API, enabling direct control over how GPUs actually work. By design, minimal error checking is done inside a Vulkan driver. Applications have full control and responsibility for correct operation. Any errors in how Vulkan is used can result in a crash. The Khronos Valiation Layer can be enabled to assist development by enabling developers to verify their applications correctly use the Vulkan API.


	see: https://vulkan.lunarg.com/doc/view/1.1.130.0/windows/validation_layers.html

*/

func (a *App) EnableDebugging() {
	/*
		a.EnableLayer("VK_LAYER_GOOGLE_threading")
		a.EnableLayer("VK_LAYER_LUNARG_parameter_validation")
		a.EnableLayer("VK_LAYER_LUNARG_device_limits")
		a.EnableLayer("VK_LAYER_LUNARG_object_tracker")
		a.EnableLayer("VK_LAYER_LUNARG_image")
		a.EnableLayer("VK_LAYER_LUNARG_core_validation")
		a.EnableLayer("VK_LAYER_LUNARG_swapchain")
		a.EnableLayer("VK_LAYER_GOOGLE_unique_objects")
	*/

	a.EnableLayer("VK_LAYER_KHRONOS_validation")

	a.EnableExtension("VK_EXT_debug_utils")
	a.EnableExtension("VK_EXT_debug_report")
}

// Enable a specific layer
func (a *App) EnableLayer(layer string) (*App, error) {
	if a.EnabledLayers == nil {
		a.EnabledLayers = make([]string, 0)
	}
	layers, err := SupportedLayers()
	if err != nil {
		return a, fmt.Errorf("error getting supported layers: %w", err)
	}
	for _, l := range layers {
		if l == layer {
			a.EnabledLayers = append(a.EnabledLayers, layer)
			return a, nil
		}
	}
	return a, fmt.Errorf("validation layer '%s' not found", layer)
}

// Enable an extension for use by the application
func (a *App) EnableExtension(extension string) *App {
	if a.EnabledExtensions == nil {
		a.EnabledExtensions = make([]string, 0)
	}
	a.EnabledExtensions = append(a.EnabledExtensions, extension)
	return a
}

//VKApplicationInfo creates a structure representing this application in a Vulkan friendly format
func (a *App) VKApplicationInfo() vk.ApplicationInfo {

	if a.APIVersion.Major < 1 {
		a.APIVersion.Major = 1
	}

	var appInfo = vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		ApiVersion:         a.APIVersion.VKVersion(),
		ApplicationVersion: a.Version.VKVersion(),
		PApplicationName:   safeString(a.Name),
		PEngineName:        safeString(a.EngineName),
	}
	return appInfo
}

// CreateInstance creates an the Vulkan Instance
func (a *App) CreateInstance() (*Instance, error) {
	appInfo := a.VKApplicationInfo()

	extensions := safeStrings(a.EnabledExtensions)
	layers := safeStrings(a.EnabledLayers)

	createInfo := vk.InstanceCreateInfo{
		SType:                   vk.StructureTypeInstanceCreateInfo,
		PApplicationInfo:        &appInfo,
		EnabledExtensionCount:   uint32(len(extensions)),
		PpEnabledExtensionNames: extensions,
		EnabledLayerCount:       uint32(len(layers)),
		PpEnabledLayerNames:     layers,
	}

	instance := &Instance{}

	err := vk.Error(vk.CreateInstance(&createInfo, nil, &instance.VKInstance))
	if err != nil {
		return nil, err
	}
	vk.InitInstance(instance.VKInstance)

	return instance, nil
}

//PhysicalDevices returns a list of physical devices known to Vulkan
func (i *Instance) PhysicalDevices() ([]*PhysicalDevice, error) {
	var deviceCount uint32
	err := vk.Error(vk.EnumeratePhysicalDevices(i.VKInstance, &deviceCount, nil))
	if err != nil {
		return nil, err
	}

	if deviceCount == 0 {
		return nil, nil
	}

	devices := make([]vk.PhysicalDevice, deviceCount)
	err = (vk.Error(vk.EnumeratePhysicalDevices(i.VKInstance, &deviceCount, devices)))
	if err != nil {
		return nil, err
	}

	ret := make([]*PhysicalDevice, deviceCount)
	for i, device := range devices {
		ret[i] = &PhysicalDevice{}
		ret[i].VKPhysicalDevice = device

		vk.GetPhysicalDeviceProperties(device, &ret[i].VKPhysicalDeviceProperties)

		ret[i].VKPhysicalDeviceProperties.Deref()
		ret[i].DeviceName = fmt.Sprintf("%s", (ret[i].VKPhysicalDeviceProperties.DeviceName))
	}
	return ret, nil

}

func (i *Instance) UseDefaultDebugCallback() {
	i.SetDebugCallback(DefaultDebugCallback)
}

type DebugCallback func(flags vk.DebugReportFlags, objectType vk.DebugReportObjectType,
	object uint64, location uint, messageCode int32, pLayerPrefix string,
	pMessage string, pUserData unsafe.Pointer) vk.Bool32

func (i *Instance) SetDebugCallback(callback vk.DebugReportCallbackFunc) error {
	var debugCallback vk.DebugReportCallback
	ret := vk.CreateDebugReportCallback(i.VKInstance, &vk.DebugReportCallbackCreateInfo{
		SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
		Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit),
		PfnCallback: callback,
	}, nil, &debugCallback)
	return vk.Error(ret)
}

// DefaultDebugCallback - taken from github.com/vulkan-go/asche/
func DefaultDebugCallback(flags vk.DebugReportFlags, objectType vk.DebugReportObjectType,
	object uint64, location uint, messageCode int32, pLayerPrefix string,
	pMessage string, pUserData unsafe.Pointer) vk.Bool32 {

	switch {
	case flags&vk.DebugReportFlags(vk.DebugReportInformationBit) != 0:
		log.Printf("INFORMATION: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportWarningBit) != 0:
		log.Printf("WARNING: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportPerformanceWarningBit) != 0:
		log.Printf("PERFORMANCE WARNING: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportErrorBit) != 0:
		log.Printf("ERROR: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportDebugBit) != 0:
		log.Printf("DEBUG: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	default:
		log.Printf("INFORMATION: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	}
	return vk.Bool32(vk.False)
}

//Instance is an instance of the Vulkan subsystem
type Instance struct {
	//VKInstance is the native Vulkan instance object
	VKInstance vk.Instance
}

func (i *Instance) Destroy() error {
	vk.DestroyInstance(i.VKInstance, nil)
	return nil
}
