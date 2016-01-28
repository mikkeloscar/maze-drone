package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	baseDir = "build_home_test"
)

var (
	pkgRepo = &Repo{
		name:    "repo",
		url:     baseDir + "/repo",
		workdir: baseDir + "/repo",
	}
	builder = &Builder{
		repo:    pkgRepo,
		workdir: baseDir + "/sources",
	}
	aurSrc = &AUR{baseDir + "/sources"}
)

func TestUpdateBuild(t *testing.T) {
	if os.Getenv("DOCKER_TEST") != "1" {
		return
	}

	_, _, err := setupBuildDirs(baseDir)
	assert.NoError(t, err, "should not fail")

	err = builder.update()
	assert.NoError(t, err, "should not fail")

	// cleanup
	err = os.RemoveAll(baseDir)
	assert.NoError(t, err, "should not fail")
}

func TestupdatePkgSrc(t *testing.T) {
	if os.Getenv("DOCKER_TEST") != "1" {
		return
	}

	pkgs, err := aurSrc.Get([]string{"wlc-git"})
	assert.NoError(t, err, "should not fail")

	_, _, err = setupBuildDirs(baseDir)
	assert.NoError(t, err, "should not fail")

	_, err = builder.updatePkgSrc(pkgs[0])
	assert.NoError(t, err, "should not fail")

	// cleanup
	err = os.RemoveAll(baseDir)
	assert.NoError(t, err, "should not fail")
}

func TestBuildPkg(t *testing.T) {
	if os.Getenv("DOCKER_TEST") != "1" {
		return
	}

	pkgs, err := aurSrc.Get([]string{"imgur"})
	assert.NoError(t, err, "should not fail")

	_, _, err = setupBuildDirs(baseDir)
	assert.NoError(t, err, "should not fail")

	_, err = builder.buildPkg(pkgs[0])
	assert.NoError(t, err, "should not fail")

	// cleanup
	err = os.RemoveAll(baseDir)
	assert.NoError(t, err, "should not fail")
}

func TestBuildPkgs(t *testing.T) {
	if os.Getenv("DOCKER_TEST") != "1" {
		return
	}

	_, _, err := setupBuildDirs(baseDir)
	assert.NoError(t, err, "should not fail")

	_, err = builder.BuildNew([]string{"imgur"}, aurSrc)
	assert.NoError(t, err, "should not fail")

	// cleanup
	err = os.RemoveAll(baseDir)
	assert.NoError(t, err, "should not fail")
}