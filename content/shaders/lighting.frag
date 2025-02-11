#version 460

layout(location = 0) in vec2 inUV;

layout(location = 0) out vec4 outColor;

layout(set = 0, binding = 0) readonly uniform DirectionalLightUniformBuffer {
    vec3 direction;
    vec3 color;
    float intensity;
};

// Descriptor set 0: G-Buffer textures and samplers
layout(set = 0, binding = 1) uniform sampler2D positionTexture;
layout(set = 0, binding = 2) uniform sampler2D normalTexture;
layout(set = 0, binding = 3) uniform sampler2D albedoTexture;
layout(set = 0, binding = 4) uniform sampler2D depthTexture;

void main() {
    // Sample G-Buffer textures using UV coordinates (inUV)
    vec3 worldPosition = texture(positionTexture, inUV).xyz;
    vec3 worldNormal = texture(normalTexture, inUV).xyz;
    vec4 albedoColor = texture(albedoTexture, inUV);
    float depthValue = texture(depthTexture, inUV).r;

    // Perform directional lighting calculation
    vec3 lightDir = normalize(-direction);
    vec3 lightColor = color;
    float lightIntensity = intensity;

    // Simple diffuse lighting (Lambertian)
    float diffuseFactor = max(0.0, dot(worldNormal, lightDir));
    vec3 diffuseLighting = lightColor * lightIntensity * diffuseFactor * albedoColor.rgb;

    // Add ambient, specular, etc. here for a more complete lighting model

    outColor = vec4(diffuseLighting, 1.0);
}
