package svm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSVM(t *testing.T) {
	svm, err := NewSolidityVersionManager()
	require.NoError(t, err)

	_, err = svm.Resolve("0.8.0")
	require.NoError(t, err)
}
