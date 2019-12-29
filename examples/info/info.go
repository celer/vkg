package main

import (
	"fmt"
	"reflect"

	vkg "github.com/celer/vkg"
	gu "github.com/docker/go-units"
	vk "github.com/vulkan-go/vulkan"
)

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func showDeviceFeatures(features vk.PhysicalDeviceFeatures) {

	features.Deref()

	tf := reflect.TypeOf(features)
	vf := reflect.ValueOf(features)
	for f := 0; f < tf.NumField(); f++ {
		sf := tf.Field(f)
		sv := vf.Field(f)
		if !sf.Anonymous && sf.Type.Kind() == reflect.Uint32 {
			v := false
			if sv.Uint() == 1 {
				v = true
			}
			fmt.Printf("\t\t%s %v\n", sf.Name, v)
		}
	}
}
func memoryHeapFlags(f vk.MemoryHeapFlagBits) string {
	s := ""

	if f&vk.MemoryHeapDeviceLocalBit != 0 {
		s += "vk.MemoryHeapDeviceLocalBit|"
	}
	if f&vk.MemoryHeapMultiInstanceBit != 0 {
		s += "vk.MemoryHeapMultiInstanceBit|"
	}

	if len(s) > 0 {
		s = s[:len(s)-1]
	}

	s += fmt.Sprintf(" (%x)", f)

	return s
}

func memoryPropertyFlags(f vk.MemoryPropertyFlagBits) string {
	s := ""

	if f&vk.MemoryPropertyHostVisibleBit != 0 {
		s += "vk.MemoryPropertyHostVisibleBit|"
	}
	if f&vk.MemoryPropertyHostCoherentBit != 0 {
		s += "vk.MemoryPropertyHostCoherentBit|"
	}
	if f&vk.MemoryPropertyHostCachedBit != 0 {
		s += "vk.MemoryPropertyHostCachedBit|"
	}
	if f&vk.MemoryPropertyDeviceLocalBit != 0 {
		s += "vk.MemoryPropertyDeviceLocalBit|"
	}
	if f&vk.MemoryPropertyLazilyAllocatedBit != 0 {
		s += "vk.MemoryPropertyLazilyAllocatedBit|"
	}
	if f&vk.MemoryPropertyProtectedBit != 0 {
		s += "vk.MemoryPropertyProtectedBit|"
	}

	if len(s) > 0 {
		s = s[:len(s)-1]
	}

	s += fmt.Sprintf(" (%x)", f)

	return s
}

func showDeviceMemory(mem vk.PhysicalDeviceMemoryProperties) {

	mem.Deref()

	fmt.Printf("\n\tType\n")
	fmt.Printf("\t\tHeapIdx\tFlags\n")
	var i uint32
	for i = 0; i < mem.MemoryTypeCount; i++ {
		mt := mem.MemoryTypes[i]
		mt.Deref()
		fmt.Printf("\t\t%d\t%s\n", mt.HeapIndex, memoryPropertyFlags(vk.MemoryPropertyFlagBits(mt.PropertyFlags)))
	}

	fmt.Printf("\n\tHeaps\n")
	for i = 0; i < mem.MemoryHeapCount; i++ {
		h := mem.MemoryHeaps[i]
		h.Deref()
		fmt.Printf("\t\t%s\t%s\n", gu.BytesSize(float64(h.Size)), memoryHeapFlags(vk.MemoryHeapFlagBits(h.Flags)))
	}
}

func showPhysicalDeviceInfo(pd *vkg.PhysicalDevice) {
	fmt.Printf("\n%s\n", pd.DeviceName)
	fmt.Printf("-----------------------------\n")
	fmt.Printf("\n\tQueue Families\n")
	queueFamilies, err := pd.QueueFamilies()
	orPanic(err)
	for _, qf := range queueFamilies {
		fmt.Printf("\t\t%s\n", qf.String())
	}
	fmt.Printf("\n\tFeatures\n")
	showDeviceFeatures(pd.VKPhysicalDeviceFeatures())
	showDeviceMemory(pd.VKPhysicalDeviceMemoryProperties())
	fmt.Printf("\n\tSupported Extensions\n")
	extensions, err := pd.SupportedExtensions()
	for _, ext := range extensions {
		ext.Deref()
		fmt.Printf("\t\t%s (%d)\n", ext.ExtensionName, ext.SpecVersion)
	}

}

func list(title string, data []string, ts int) {
	fmt.Printf("%s\n", title)
	fmt.Printf("-----------------------------\n")
	for _, d := range data {
		for i := ts; i > 0; ts-- {
			fmt.Printf("\t")
		}
		fmt.Printf("\t%s\n", d)
	}
	fmt.Printf("\n")
}

func main() {

	err := vkg.InitializeForComputeOnly()
	orPanic(err)

	extensions, err := vkg.SupportedExtensions()
	orPanic(err)
	list("Extensions", extensions, 0)

	layers, err := vkg.SupportedLayers()
	orPanic(err)
	list("Layers", layers, 0)

	app := &vkg.App{
		Name: "Info",
	}

	instance, err := app.CreateInstance()

	physicalDevices, err := instance.PhysicalDevices()

	for _, physicalDevice := range physicalDevices {
		showPhysicalDeviceInfo(physicalDevice)
	}
}
