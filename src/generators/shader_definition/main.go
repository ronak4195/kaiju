package main

import (
	"bufio"
	"kaiju/klib/string_equations"
	"kaiju/rendering"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	vk "kaiju/rendering/vulkan"
)

var (
	layoutInReg      = regexp.MustCompile(`layout\s{0,}\((\w+)\s{0,}=\s{0,}([\w\d\s\+\-\*\/]+)\)\s{0,}in\s+(\w+)\s+(\w+)`)
	layoutOutReg     = regexp.MustCompile(`layout\s{0,}\((\w+)\s{0,}=\s{0,}([\w\d\s\+\-\*\/]+)\)\s{0,}out\s+(\w+)\s+(\w+)`)
	layoutUniformReg = regexp.MustCompile(`layout\s{0,}\((\w+)\s{0,}=\s{0,}([\w\d\s\+\-\*\/]+)\)\s{0,}uniform\s+(\w+)\s+(\w+)`)

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

type Layout struct {
	location uint32
	binding  uint32
	offset   uint32
	typeName string
	name     string
}

func (l *Layout) format() vk.Format { return formatMapping[l.typeName] }

type Uniform struct {
	Layout
	descriptorSet  uint32
	descriptorType vk.DescriptorType
	stageFlags     vk.ShaderStageFlags
}

type ShaderSource struct {
	src       string
	file      string
	defines   map[string]any
	layoutIn  []Layout
	layoutOut []Layout
	uniforms  []Uniform
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
					panic("error processing equation (" + match[0] + ") " + err.Error())
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
				panic("failed to load import file (" + match[1] + "): " + err.Error())
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
		panic("failed to read the file: " + err.Error())
	}

	source.src = readImports(string(data), filepath.Dir(source.file))
	source.readDefines()
	source.layoutIn = source.readLayout(layoutInReg)
	source.layoutOut = source.readLayout(layoutOutReg)
	uniforms := source.readLayout(layoutUniformReg)
	source.uniforms = make([]Uniform, len(uniforms))
	for i := range uniforms {
		source.uniforms[i].Layout = uniforms[i]
	}
	return source
}

func (s *ShaderSource) readLayout(re *regexp.Regexp) []Layout {
	matches := re.FindAllStringSubmatch(s.src, -1)
	layouts := make([]Layout, len(matches))
	for i := range matches {

		val := matches[i][2]
		var intVal uint32
		if v, err := strconv.Atoi(val); err == nil {
			intVal = uint32(v)
		} else {
			if v, err := s.processDefineEquation(val); err == nil {
				intVal = uint32(v)
			} else {
				panic("failed to read layout location (" + matches[i][0] + "): " + err.Error())
			}
		}
		if matches[i][1] == "location" {
			layouts[i].location = intVal
		} else if matches[i][1] == "binding" {
			layouts[i].binding = intVal
		}
		layouts[i].typeName = matches[i][3]
		layouts[i].name = matches[i][4]
	}
	return layouts
}

func main() {
	const tmp = "content/shaders/definitions/basic.json"
	d, err := os.ReadFile(tmp)
	if err != nil {
		panic("failed to read the shader definition file: " + err.Error())
	}
	def, err := rendering.ShaderDefFromJson(string(d))
	if err != nil {
		panic("failed to parse the shader definition file: " + err.Error())
	}
	v := readShaderCode(def.GLSL.Vert)
	f := readShaderCode(def.GLSL.Frag)
	println(v.src)
	println(f.src)
	//f := readShaderCode(def.GLSL.Frag)
}
