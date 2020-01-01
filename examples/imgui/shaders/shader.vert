#version 450
#extension GL_ARB_separate_shader_objects : enable


layout(binding = 0) uniform UniformBufferObject {
	mat4 proj;
} ubo;


layout(location = 0) in vec2 inPosition;
layout(location = 1) in vec2 inTexCoord;
layout(location = 2) in uvec4 inColor;

layout(location = 0) out vec4 fragColor;
layout(location = 1) out vec2 fragTexCoord;

void main() {
    gl_Position = ubo.proj * vec4(inPosition.xy,0.0,1.0); 
    fragColor = vec4(inColor[0]/255.0,inColor[1]/255.0,inColor[2]/255.0,inColor[3]/255.0);
    fragTexCoord = inTexCoord;
}
