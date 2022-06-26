// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultVersionParser(t *testing.T) {
	v, err := DefaultParser.Parse("v1.2.3")

	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "v1.2.3", v.String())
	assert.Equal(t, 1, v.Major())
	assert.Equal(t, 2, v.Minor())
	assert.Equal(t, 3, v.Patch())

	badVersions := []string{
		"",
		"1.2.3",
		"vz.2.3",
		"v1.z.3",
		"v1.2.z",
	}
	for _, badVersion := range badVersions {
		_, err := DefaultParser.Parse(badVersion)
		assert.Error(t, err)
	}
}

func TestDefaultApplicationParser(t *testing.T) {
	v, err := DefaultApplicationParser.Parse("axia/1.2.3")

	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "axia/1.2.3", v.String())
	assert.Equal(t, "axia", v.App())
	assert.Equal(t, 1, v.Major())
	assert.Equal(t, 2, v.Minor())
	assert.Equal(t, 3, v.Patch())
	assert.NoError(t, v.Compatible(v))
	assert.False(t, v.Before(v))

	badVersions := []string{
		"",
		"axia/",
		"axia/z.0.0",
		"axia/0.z.0",
		"axia/0.0.z",
	}
	for _, badVersion := range badVersions {
		_, err := DefaultApplicationParser.Parse(badVersion)
		assert.Error(t, err)
	}
}

func TestNewApplicationParser(t *testing.T) {
	p := NewApplicationParser(":", ",")

	v, err := p.Parse("axia:1,2,3")

	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "axia:1,2,3", v.String())
	assert.Equal(t, "axia", v.App())
	assert.Equal(t, 1, v.Major())
	assert.Equal(t, 2, v.Minor())
	assert.Equal(t, 3, v.Patch())
	assert.NoError(t, v.Compatible(v))
	assert.False(t, v.Before(v))
}
