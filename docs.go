/*
Package vkg implements an abstraction atop the Vulkan graphics framework for go. Vulkan is a very
powerful graphics and compute framework, which expands upon what is possible with OpenGL and OpenCL,
but at a cost - it is very difficult and complex to use, yet very powerful.

This package provides utility functions and objects which make Vulkan a little more paltable to
gophers, with the drawback that it cannot possibly expose all the power inherent in Vulkan, so
it strives to avoid limiting users when they desire to utilize the full power of Vulkan.

Overview of Vulkan

Vulkan overcomes many of the drawbacks of OpenGL by providing a much less abstracted interface
to GPUs at a pretty hefty cost, all the things OpenGL managed previously are now up to the
implementing application to manage. This documentation is woefully inadequate at fully descrbing
how Vulkan works, and is meant to provide a refersher and perhaps some bread crumbs which
will aid in undersatnding the system.

At a high level vulkan provides a method of submiting work to command queues which execute
various pipeline (which the application has configured) to either display graphics or peform
compute work. An example of a pipeline might be one which displays a set of vertex data
in a specific way, or a compute pipeline which pefroms some massively parallel mathmatical
operation on a large set of data. The challenge then becomes providing the data in a way
which these pipelines can utilize, so the data must be described adqueately enough so that
the GPU can work with the data in an optimal way. To allow application developers to better
utilize the underlying hardware, many decisions about how this data is managed is now left
up to the application developer, rather than manged by a framework like OpenGL.

This means the application developer is now responsible for deciding where the data which
is to be processed by the pipeline will reside, what the format of the data is and how to
make sure the data arrives at the appopriate time to best utilize the underlying hardware
(this is a daunting task and application specific). So a game developed with Vulkan will
have different needs and priorites than a scientific computing application developed with
Vulkan, or even a 3D editor.

This package can provide some assitance in getting started with vulkan but cannot possibly
provide apporpaite abstractions for all possible use cases, or else it would fall into
the trap of OpenGL - doing too much in a sub optimal way.

Back to a discussion of Vulkan, we have command queues, which are feeding pipelines,
which are using data provided by the application.


Native Vulkan terms
	Instance 	the vulkan runtime instance
	PhysicalDevice	the physical hardware device
	LogicalDevice	a representation of the device which is the target of most of the vulkan apis.
	Pipeline	a description of how to process data on the GPU
	Queue 		a queue which work (comand buffers) may be submitted to
	DeviceMemory	a allocation of memory on the host or device for use by buffers and images
	Buffer		a description of some bit of data (vertex, index, or other)
	Image		a description of some image
	ImageView	a way of descibing how an image is utilized or viewed
	DescriptorSet 	a mapping of data for use by shaders
	DescriptorSetLayout a description of what data is in the descriptor set
	Swapchain	a grouping of images which are used to display graphical data


When thinking about how a graphics application with Vulkan works a very high level sequences of might be:

	1. Intialize the vulkan instance
	2. Setup the swapchain and framebuffers
	3. Allocate buffers and device memory
		3a. Allocate a buffer and memory on the host for vertex data
		3b. Allocate a staging buffer in host memory to send texture data to the device
		3c. Allocate a image in the device memory to recieve the texture data
	4. Load data
		4a. Load vertex into the host allocated memory by 'mapping' the memory
		4b. Load texture data into a host allocated staging buffer
	5. Query the physical device for queues which work may be submitted to
	6. Submit a command buffer to a device queue to copy the image data loaded in 3b. to the image created in 3c
	7. Create a descriptor set which describes how data is related in the application
	8. Configure and create a graphics pipeline (and provide it the descriptor set)
	9. Start drawing frames:
		10. Fill a command buffer with the pipeline and vertex details to draw
		11. Draw the image and display it.

About this package

This package provides a basic set of APIs which wrap some of the Vulkan APIs to make them a bit
less painful to work with, the trade off being that many of the native Vulkan APIs expose options
which are not exposed in the APIs provided by the package. To mitigate the drawback of this approach
native vulkan structures are exposed in all the objects prefixed with the 'VK' in the name - so
applications aren't limited by what this package provides.

This package goes beyond what Vulkan natively provides to make Vulkan a bit easier to work with:

GraphicsApp:
	a basic graphics application framework which managed setup and frame drawing
ResourceManager:
	a resource manager which manages memory allocation and can assist with staging of resources
Utility interfaces:
	for using and describing data


*/
package vkg
