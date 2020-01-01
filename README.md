# Intro

This repo is very much a work in progress of utilizing Vulkan with go. I've wrapped the vulkan-go/vulkan API to make
them a little more idiomatic, and also provided a bunch of utility classes. 

Here is where I'm at:

  * Works with ImGUI
  * Custom memory allocator see allocator.go
  * Utility class called GraphicsApp which does most of the bootstrapping required to get a vulkan app up and going

If you want to get a good idea of where I'm going checkout examples/imgui

Here is where I expect to go;

  * More documentation
  * More examples
  * Unit tests

I'm hoping to continue pushing on this repo more in the next few weeks. 

Here is a picture of the examples/imgui program:

![Example program](/assets/imgui.png)


