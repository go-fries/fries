package font

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

const (
	defaultDPI     = 72
	opaqueAlpha    = 0xff
	hexShortLength = 6
	hexLongLength  = 8
)

// LoadFace 加载指定路径的字体。
func LoadFace(path string, size float64) (font.Face, error) {
	if path == "" {
		return nil, fmt.Errorf("font path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read font: %w", err)
	}
	tt, err := truetype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	face := truetype.NewFace(tt, &truetype.Options{Size: size, DPI: defaultDPI})
	return face, nil
}

// DefaultFace 返回可用的默认字体（ASCII 友好，不保证中文）。
func DefaultFace() font.Face {
	return basicfont.Face7x13
}

// LoadFaceWithFallback 依次尝试加载指定路径与常见中文字体，失败则回退到默认字体。
// 返回的字符串为实际使用的字体路径（若为内置默认字体则为空）。
func LoadFaceWithFallback(size float64, preferred ...string) (font.Face, string, error) {
	candidates := deduplicate(append(preferred, collectCommonCandidates()...))
	loadErrs := make([]string, 0, len(candidates))
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		face, err := LoadFace(p, size)
		if err == nil {
			return face, p, nil
		}
		loadErrs = append(loadErrs, fmt.Sprintf("%s: %v", p, err))
	}

	if env := strings.TrimSpace(os.Getenv("GGM_FONT_PATH")); env != "" {
		if face, err := LoadFace(env, size); err == nil {
			return face, env, nil
		}
	}

	if len(loadErrs) > 0 {
		return DefaultFace(), "", fmt.Errorf("font fallback to default; tried: %s", strings.Join(loadErrs, "; "))
	}
	return DefaultFace(), "", fmt.Errorf("font fallback to default; no usable font found")
}

func collectCommonCandidates() []string {
	return []string{
		strings.TrimSpace(os.Getenv("GGM_FONT_PATH")),
		filepath.Join("testdata", "NotoSansSC-Regular.otf"),
		filepath.Join("testdata", "NotoSansSC-Regular.ttf"),
		"/usr/share/fonts/truetype/noto/NotoSansSC-Regular.otf",
		"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
		"/System/Library/Fonts/PingFang.ttc",
		"/System/Library/Fonts/STHeiti Light.ttc",
		`C:\Windows\Fonts\msyh.ttc`,
	}
}

func deduplicate(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		result = append(result, p)
	}
	return result
}

// ParseHexColorWithDefault 转换 hex 颜色字符串。
func ParseHexColorWithDefault(hex string, fallback color.Color) color.Color {
	c, err := parseHexColor(hex)
	if err != nil {
		return fallback
	}
	return c
}

func parseHexColor(s string) (color.RGBA, error) {
	c := color.RGBA{A: opaqueAlpha}
	if s == "" {
		return c, fmt.Errorf("empty color")
	}
	if s[0] == '#' {
		s = s[1:]
	}
	var err error
	switch len(s) {
	case hexShortLength:
		_, err = fmt.Sscanf(s, "%02x%02x%02x", &c.R, &c.G, &c.B)
	case hexLongLength:
		_, err = fmt.Sscanf(s, "%02x%02x%02x%02x", &c.R, &c.G, &c.B, &c.A)
	default:
		return c, fmt.Errorf("invalid color: %s", s)
	}
	return c, err
}
