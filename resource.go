package vkg

/*
type Resource struct {
	Buffer             *Buffer
	LocalDeviceMemory  *DeviceMemory
	RemoteDeviceMemory *DeviceMemory

	LocalOffset  uint64
	RemoteOffset uint64
}

func (r *Resource) IsLocal() {
	return r.RemoteDeviceMemory == nil
}

func (r *Resource) Destroy() {
}

func main() {

	//app.AllocateMemory(where, what, size, name)
	app.Allocate(Host, Vertex|Index, 100*10, "Pool1")
	app.AllocateWithOptions(Host, Vertex|Index, 100*10, "Pool1",options)


	app.Allocate(Device,Image,1000*100,"RImagePool2")


	app.Memory("RImagePool2").AllocateTextureFromFile(filename) err
	app.Memory("RImagePool2").AllocateTextureFromImage(image.RGBA) err
	t:=app.Memory("RImagePool2").AllocateTextureFromData([]byte string,width int,height int) err
	t.VKImage
	t.VKImageView
	t.VKSampler
	t.Sampler().....
	t.SyncToDevice() -> Syncs right now (synchronously)
	t.CmdsToSync(cmd *CommandBuffer)
	t.Free()


	r,_ := app.Memory("pool1").Allocate(Vertex) -> // Allocates and figures out offset
		app.Memory("pool1").AllocateWithOptions(Vertex)
	r.FillFrom(vs)
	r.FillFrom(is)
	r.VKBuffer -> (Cannot get memory from an allocated resource)
	r.Free()


}
*/
