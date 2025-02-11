#version 460

layout(location = 0) in vec3 inWorldPosition;
layout(location = 1) in vec3 inWorldNormal;
layout(location = 2) in vec4 inAlbedo;
layout(location = 3) in vec2 inUV;

layout(location = 0) out vec4 outPosition;
layout(location = 1) out vec4 outNormal;
layout(location = 2) out vec4 outAlbedo;

void main() {
	outPosition = vec4(inWorldPosition, 1.0);
	// Normalize again for interpolated normals
	outNormal = vec4(normalize(inWorldNormal), 0.0);
	outAlbedo = inAlbedo;
}
