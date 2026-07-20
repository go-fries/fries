package ptr_test

import (
	"regexp"
	"testing"

	"github.com/go-fries/fries/ptr/v4"
	"github.com/stretchr/testify/assert"
)

var versionRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)` +
	`(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

func TestVersionSemver(t *testing.T) {
	version := ptr.Version()
	assert.NotNil(t, versionRegex.FindStringSubmatch(version), "version is not semver: %s", version)
}
