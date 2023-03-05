package gosolc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnique(t *testing.T) {
	cases := []struct {
		input  []string
		output []string
	}{
		{
			[]string{"a", "b"},
			[]string{"a", "b"},
		},
		{
			[]string{"a", "a", "b"},
			[]string{"a", "b"},
		},
	}

	for _, c := range cases {
		output := unique(c.input)
		require.Equal(t, c.output, output)
	}
}

func TestParseSource_Dependencies(t *testing.T) {
	cases := []struct {
		code string
		deps []string
	}{
		{
			`import "../Basic.sol";`,
			[]string{
				"../Basic.sol",
			},
		},
		{
			`import '../Basic.sol';`,
			[]string{
				"../Basic.sol",
			},
		},
	}

	for _, c := range cases {
		deps := parseDependencies(c.code)
		require.Equal(t, c.deps, deps)
	}
}

func TestParseSource_Pragma(t *testing.T) {
	cases := []struct {
		code    string
		pragmas []string
	}{
		{
			`pragma solidity >=0.8.0;`,
			[]string{
				">=0.8.0",
			},
		},
	}

	for _, c := range cases {
		pragmas, err := parsePragma(c.code)
		require.NoError(t, err)

		require.Equal(t, c.pragmas, pragmas)
	}
}

func TestResolveRelativePaths(t *testing.T) {
	cases := []struct {
		path string
		deps []string
		res  []string
	}{
		{
			"path/",
			[]string{
				"./file1",
				"../file2",
			},
			[]string{
				"path/file1",
				"file2",
			},
		},
		{
			"path/",
			[]string{
				"../../file2",
			},
			nil,
		},
	}

	for _, c := range cases {
		deps, err := resolveRelativeImports(c.deps, c.path)
		if err != nil && c.res != nil {
			t.Fatal(err)
		}
		if err == nil && c.res == nil {
			t.Fatal("it should have failed")
		}
		require.Equal(t, c.res, deps)
	}
}
