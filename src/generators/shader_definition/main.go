package main

import (
	"bufio"
	"encoding/json"
	"kaiju/klib/string_equations"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	vk "kaiju/rendering/vulkan"
)

type LayoutStructField struct {
	Type string // float, vec3, mat4, etc.
	Name string
}

type Layout struct {
	Location int    // -1 if not set
	Binding  int    // -1 if not set
	Set      int    // -1 if not set
	Type     string // float, vec3, mat4, etc.
	Name     string
	Source   string // in, out, uniform
	Fields   []LayoutStructField
}

type ShaderDefinitionStages struct {
	Vert string // The path to the vertex shader file
	Frag string // The path to the fragment shader file
	Geom string // The path to the geometry shader file
	Tesc string // The path to the tesselation control shader file
	Tese string // The path to the tesselation evaluation shader file
}

type ShaderDefinition struct {
	CullMode    string // front, back, none
	DrawMode    string // lines, points, triangles
	GLSL        ShaderDefinitionStages
	VertLayouts []Layout
	FragLayouts []Layout
	GeomLayouts []Layout
	TescLayouts []Layout
	TeseLayouts []Layout
}

var (
	layoutReg = regexp.MustCompile(`(?s)\s*layout\s*\(([\w\s=\d,]+)\)\s*(?:readonly\s+)?(in|out|uniform)\s+([a-zA-Z0-9]+)\s+([a-zA-Z0-9_]+){0,1}(?:\s*\{(.*?)\})?\s*(\w+){0,1}`)

	formatMapping = map[string]vk.Format{
		"float":  vk.FormatR32Sfloat,
		"vec2":   vk.FormatR32g32Sfloat,
		"vec3":   vk.FormatR32g32b32Sfloat,
		"vec4":   vk.FormatR32g32b32a32Sfloat,
		"mat4":   vk.FormatR32g32b32a32Sfloat, // 4
		"int32":  vk.FormatR32Sint,
		"uint32": vk.FormatR32Uint,
	}
)

func (l *Layout) format() vk.Format { return formatMapping[l.Type] }

type Uniform struct {
	Layout
	descriptorSet  uint32
	descriptorType vk.DescriptorType
	stageFlags     vk.ShaderStageFlags
}

type ShaderSource struct {
	src     string
	file    string
	defines map[string]any
	layouts []Layout
}

func (s *ShaderSource) defineAsString(name string) string {
	if d, ok := s.defines[name]; ok {
		if v, ok := d.(string); ok {
			return v
		} else {
			return strconv.FormatFloat(d.(float64), 'G', 10, 64)
		}
	}
	return name
}

func (s *ShaderSource) processDefineEquation(value string) (float64, error) {
	// Go through and replace all existing defines in the expression
	fields := strings.Fields(value)
	for i := range fields {
		fields[i] = s.defineAsString(fields[i])
	}
	return string_equations.CalculateSimpleStringExpression(strings.Join(fields, " "))
}

func (s *ShaderSource) readDefines() {
	re := regexp.MustCompile(`#define\s+(\w+)\s+([\w\d\s\+\-\*\/]+)`)
	ops := []string{"+", "-", "*", "/"}
	scan := bufio.NewScanner(strings.NewReader(s.src))
	for scan.Scan() {
		line := scan.Text()
		match := re.FindStringSubmatch(line)
		if len(match) == 3 {
			name := match[1]
			value := match[2]
			isEquation := false
			for j := range ops {
				isEquation = isEquation || strings.Contains(value, ops[j])
			}
			if isEquation {
				if v, err := s.processDefineEquation(value); err == nil {
					s.defines[name] = v
				} else {
					log.Fatalf("error processing equation (%s): %s", match[0], err)
				}
			} else {
				if f, err := strconv.ParseFloat(value, 64); err == nil {
					s.defines[name] = f
				} else {
					s.defines[name] = value
				}
			}
		}
	}
}

func (s *ShaderSource) readLayouts() {
	matches := layoutReg.FindAllStringSubmatch(s.src, -1)
	s.layouts = make([]Layout, len(matches))
	for i := range matches {
		name := matches[i][4]
		if name == "" {
			name = matches[i][6]
		}
		s.layouts[i] = Layout{
			Location: -1,
			Binding:  -1,
			Set:      -1,
			Type:     matches[i][3],
			Name:     name,
			Source:   matches[i][2],
		}
		attrs := strings.Split(matches[i][1], ",")
		for j := range attrs {
			parts := strings.Fields(attrs[j])
			val, err := s.processDefineEquation(strings.Join(parts[2:], " "))
			if err != nil {
				log.Fatalf("invalid value for layout (%s): %s", matches[i][0], err)
			}
			switch parts[0] {
			case "location":
				s.layouts[i].Location = int(val)
			case "binding":
				s.layouts[i].Binding = int(val)
			case "set":
				s.layouts[i].Set = int(val)
			}
		}
		if matches[i][5] != "" {
			fields := strings.Split(strings.TrimSpace(matches[i][5]), ";")
			if len(fields) > 0 && fields[len(fields)-1] == "" {
				fields = fields[:len(fields)-1]
			}
			s.layouts[i].Fields = make([]LayoutStructField, len(fields))
			for j := range fields {
				parts := strings.Fields(fields[j])
				s.layouts[i].Fields[j] = LayoutStructField{
					Type: parts[0],
					Name: parts[1],
				}
			}
		}
	}
}

func readImports(inSrc, path string) string {
	src := strings.Builder{}
	scan := bufio.NewScanner(strings.NewReader(inSrc))
	re := regexp.MustCompile(`\s{0,}#include\s+\"([\w\.]+)\"`)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		match := re.FindStringSubmatch(line)
		if len(match) == 2 && match[1] != "" {
			importSrc, err := os.ReadFile(filepath.Join(path, match[1]))
			if err != nil {
				log.Fatalf("failed to load import file (%s): %s", match[1], err)
			}
			src.WriteString(readImports(string(importSrc), path))
		} else {
			src.WriteString(line + "\n")
		}
	}
	return src.String()
}

func readShaderCode(file string) ShaderSource {
	source := ShaderSource{
		file:    "content/" + file,
		defines: make(map[string]any),
	}
	data, err := os.ReadFile(source.file)
	if err != nil {
		log.Fatalf("failed to read the file: %s", err)
	}

	source.src = readImports(string(data), filepath.Dir(source.file))
	source.readDefines()
	source.readLayouts()
	return source
}

func main() {
	const tmp = "content/shaders/definitions/test.json"
	d, err := os.ReadFile(tmp)
	if err != nil {
		log.Fatalf("failed to read the shader definition file: %s", err)
	}
	var def ShaderDefinition
	if err := json.Unmarshal(d, &def); err != nil {
		log.Fatalf("failed to parse the shader definition file: %s", err)
	}
	if def.GLSL.Vert != "" {
		v := readShaderCode(def.GLSL.Vert)
		def.VertLayouts = v.layouts
	}
	if def.GLSL.Frag != "" {
		f := readShaderCode(def.GLSL.Frag)
		def.FragLayouts = f.layouts
	}
	if def.GLSL.Geom != "" {
		f := readShaderCode(def.GLSL.Geom)
		def.GeomLayouts = f.layouts
	}
	if def.GLSL.Tesc != "" {
		f := readShaderCode(def.GLSL.Tesc)
		def.TescLayouts = f.layouts
	}
	if def.GLSL.Tese != "" {
		f := readShaderCode(def.GLSL.Tese)
		def.TeseLayouts = f.layouts
	}
	if out, err := json.Marshal(def); err == nil {
		os.WriteFile(tmp, out, os.ModePerm)
	} else {
		log.Fatalf("failed to serialize the layout for %s", tmp)
	}
}
