#version 450

layout(location = 0) in vec4 fragColor;
layout(location = 1) in vec4 fragBGColor;
layout(location = 2) in vec2 fragTexCoord;
layout(location = 3) in vec2 fragPxRange;
layout(location = 4) in vec2 fragTexRange;

layout(binding = 1) uniform sampler2D texSampler;

layout(location = 0) out vec4 outColor;
layout(location = 1) out float reveal;

void main() {
	vec4 texColor = texture(texSampler, fragTexCoord) * fragColor;
	vec4 unWeightedColor = texColor;
#include "inc_fragment_oit_block.inl"
}