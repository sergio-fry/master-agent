package version_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"master-agent/internal/version"
)

func TestVersionIsSet(t *testing.T) {
	require.NotEmpty(t, version.Version)
}
