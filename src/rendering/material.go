package rendering

import (
	"encoding/json"
	"kaiju/assets"
	vk "kaiju/rendering/vulkan"
	"log/slog"
	"slices"
	"strings"
	"weak"
)

type Material struct {
	Name         string
	shaderInfo   ShaderDataCompiled
	renderPass   *RenderPass
	pipelineInfo ShaderPipelineDataCompiled
	Shader       *Shader
	Textures     []*Texture
	Instances    map[string]*Material
	Root         weak.Pointer[Material]
	Clears       []vk.ClearValue
}

type MaterialTextureData struct {
	Texture string
	Filter  string `options:"StringVkFilter"`
}

type MaterialClearColorData struct {
	IsColor        bool
	R              float32
	G              float32
	B              float32
	A              float32
	IsDepthStencil bool
	Depth          float32
	Stencil        uint32
}

type MaterialData struct {
	Name           string
	Shader         string `options:""` // Blank options uses fallback
	RenderPass     string `options:""` // Blank options uses fallback
	ShaderPipeline string `options:""` // Blank options uses fallback
	Textures       []MaterialTextureData
	Clears         []MaterialClearColorData
}

func (m *Material) CreateInstance(textures []*Texture) *Material {
	instanceKey := strings.Builder{}
	for i := range textures {
		instanceKey.WriteString(textures[i].Key)
		instanceKey.WriteRune(';')
	}
	key := instanceKey.String()
	if found, ok := m.Instances[key]; ok {
		return found
	}
	copy := &Material{}
	*copy = *m
	copy.Textures = slices.Clone(textures)
	m.Instances[key] = copy
	copy.Root = weak.Make(m)
	copy.Instances = nil
	return copy
}

func (d *MaterialTextureData) FilterToVK() TextureFilter {
	switch d.Filter {
	case "Nearest":
		return TextureFilterNearest
	case "Linear":
		return TextureFilterLinear
	case "CubicImg":
		// TODO:  Implement this filter
		fallthrough
	default:
		return TextureFilterLinear
	}
}

func materialUnmarshallData(assets *assets.Database, file string, to any) error {
	s, err := assets.ReadText(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(s), to); err != nil {
		return err
	}
	return nil
}

func (d *MaterialData) Compile(assets *assets.Database, renderer Renderer) (*Material, error) {
	vr := renderer.(*Vulkan)
	c := &Material{
		Name:      d.Name,
		Textures:  make([]*Texture, len(d.Textures)),
		Instances: make(map[string]*Material),
	}
	sd := ShaderData{}
	rp := RenderPassData{}
	sp := ShaderPipelineData{}
	if err := materialUnmarshallData(assets, d.Shader, &sd); err != nil {
		return c, err
	}
	if err := materialUnmarshallData(assets, d.RenderPass, &rp); err != nil {
		return c, err
	}
	if err := materialUnmarshallData(assets, d.ShaderPipeline, &sp); err != nil {
		return c, err
	}
	c.shaderInfo = sd.Compile()
	if pass, ok := vr.renderPassCache[rp.Name]; !ok {
		rpc := rp.Compile(vr)
		if p, ok := rpc.ConstructRenderPass(vr); ok {
			vr.renderPassCache[rp.Name] = p
			c.renderPass = p
		} else {
			slog.Error("failed to load the render pass for the material", "material", d.Name, "renderPass", rp.Name)
		}
	} else {
		c.renderPass = pass
	}
	c.pipelineInfo = sp.Compile()
	shaderConfig, err := assets.ReadText(d.Shader)
	if err != nil {
		return c, err
	}
	var rawSD ShaderData
	if err := json.Unmarshal([]byte(shaderConfig), &rawSD); err != nil {
		return c, err
	}
	c.Shader, _ = vr.caches.ShaderCache().Shader(rawSD.Compile())
	c.Shader.pipelineInfo = &c.pipelineInfo
	c.Shader.renderPass = c.renderPass
	for i := range d.Textures {
		tex, err := vr.caches.TextureCache().Texture(
			d.Textures[i].Texture, d.Textures[i].FilterToVK())
		if err != nil {
			return c, err
		}
		c.Textures[i] = tex
	}
	c.Clears = make([]vk.ClearValue, len(d.Clears))
	for i := range d.Clears {
		dc := &d.Clears[i]
		if dc.IsColor == dc.IsDepthStencil {
			slog.Error("clears for material can't be both color and depth stencil", "index", i, "material", d.Name)
		}
		if dc.IsDepthStencil {
			c.Clears[i].SetDepthStencil(dc.Depth, dc.Stencil)
		} else {
			c.Clears[i].SetColor([]float32{dc.R, dc.G, dc.B, dc.A})
		}
	}
	return c, nil
}
