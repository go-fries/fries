package jet

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ================================================================================================
// PathGenerator generates the path of the service method
// ================================================================================================

var DefaultPathGenerator PathGenerator = NewGeneralPathGenerator()

type PathGenerator interface {
	Generate(service, name string) string
}

// PathGeneratorFunc generates the path of the service method
type PathGeneratorFunc func(service, name string) string

func (f PathGeneratorFunc) Generate(service, name string) string {
	return f(service, name)
}

// GeneralPathGenerator generates the general path of the service method
type GeneralPathGenerator struct{}

func NewGeneralPathGenerator() *GeneralPathGenerator {
	return &GeneralPathGenerator{}
}

var (
	generalServiceRegexp = regexp.MustCompile(`Service$`)
	generalSpaceRegexp   = regexp.MustCompile(`\s+`)
)

func (g *GeneralPathGenerator) Generate(service, name string) string {
	servers := strings.Split(service, "\\")
	path := generalServiceRegexp.ReplaceAllString(servers[len(servers)-1], "")

	if !g.isLower(path) {
		path = g.snake(path)
	}
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	return path + "/" + name
}

func (g *GeneralPathGenerator) snake(s string) string {
	s = generalSpaceRegexp.ReplaceAllString(g.ucwords(s), "")
	s = g.prefixUpperDelimiter(s)
	return strings.ToLower(s)
}

func (g *GeneralPathGenerator) isLower(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func (g *GeneralPathGenerator) ucwords(s string) string {
	return cases.Title(language.Und, cases.NoLower).String(s)
}

func (g *GeneralPathGenerator) prefixUpperDelimiter(s string) string {
	rs := make([]rune, 0, len(s)*2) //nolint:mnd
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			rs = append(rs, '_')
		}
		rs = append(rs, r)
	}
	return string(rs)
}

// FullPathGenerator generates the full path of the service method
type FullPathGenerator struct{}

func NewFullPathGenerator() *FullPathGenerator {
	return &FullPathGenerator{}
}

func (f *FullPathGenerator) Generate(service, name string) string {
	path := strings.ReplaceAll(service, "\\", "/")
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	return path + "/" + name
}

// ================================================================================================
// IDGenerator generates the id of the request
// ================================================================================================

var DefaultIDGenerator IDGenerator = NewUUIDGenerator()

type IDGenerator interface {
	Generate() string
}

type IDGeneratorFunc func() string

func (f IDGeneratorFunc) Generate() string {
	return f()
}

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (u *UUIDGenerator) Generate() string {
	return uuid.New().String()
}
