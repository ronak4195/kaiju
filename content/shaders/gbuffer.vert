#version 460

#include "inc_vertex.inl"

layout(location = 0) out vec3 outWorldPosition;
layout(location = 1) out vec3 outWorldNormal;
layout(location = 2) out vec4 outAlbedo;
layout(location = 3) out vec2 outUV;

void main() {
	// Calculate world position
	vec4 worldPosition = model * vec4(Position, 1.0);
	outWorldPosition = worldPosition.xyz;
	// Calculate world normal
	//mat3 normalMatrix = mat3(transpose(inverse(model))); // Non-uniform scaling
	mat3 normalMatrix = mat3(model); // Uniform scaling, (faster)
	outWorldNormal = normalize(normalMatrix * Normal);
	outAlbedo = Color;
	outUV = UV0;
	// Calculate clip space position
	gl_Position = projection * view * model * vec4(Position, 1.0);
	//gl_Position = projection * view * worldPosition;
}
