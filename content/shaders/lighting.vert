#version 460

layout(location = 0) in vec4 dummy;

layout(location = 0) out vec2 outUV;

void main() {
    // Generate screen-space quad vertices and UVs directly in the vertex shader
    vec2 positions[6];
	vec2 uvs[6];
	positions[0] = vec2(-1.0, 1.0);
	positions[1] = vec2(-1.0, -1.0);
	positions[2] = vec2(1.0, 1.0);
	positions[3] = vec2(1.0, 1.0);
	positions[4] = vec2(-1.0, -1.0);
	positions[5] = vec2(1.0, -1.0);
	uvs[0] = vec2(0.0, 1.0);
	uvs[1] = vec2(0.0, 0.0);
	uvs[2] = vec2(1.0, 1.0);
	uvs[3] = vec2(1.0, 1.0);
	uvs[4] = vec2(0.0, 0.0);
	uvs[5] = vec2(1.0, 0.0);
    gl_Position = vec4(positions[gl_VertexIndex], 0.0, 1.0);
    outUV = uvs[gl_VertexIndex];
}