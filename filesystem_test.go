package gosolc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadDir(t *testing.T) {
	files, err := readDir("./fixtures/with-relative-deps")
	require.NoError(t, err)

	require.Equal(t, "fixtures/with-relative-deps/Basic.sol", files[0].path)
	require.Equal(t, "fixtures/with-relative-deps/deps/Dependency.sol", files[1].path)
}
