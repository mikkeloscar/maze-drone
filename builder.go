package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/mikkeloscar/gopkgbuild"
)

// BuiltPkg defines a built package and optional signature file.
type BuiltPkg struct {
	Pkg       string
	Signature string
}

func (b *BuiltPkg) String() string {
	if b.Signature != "" {
		return fmt.Sprintf("%s (%s)", path.Base(b.Pkg), path.Base(b.Signature))
	}

	return path.Base(b.Pkg)
}

// Builder is used to build arch packages.
type Builder struct {
	workdir string
	repo    *Repo
	config  ArchBuild
}

// BuildNew checks what packages to build based on related repo and builds
// those that have been updated.
func (b *Builder) BuildNew(pkgs []string, aur *AUR) ([]*BuiltPkg, error) {
	// make sure environment is up to date
	err := b.update()
	if err != nil {
		return nil, err
	}

	// get packages that should be built
	srcPkgs, err := b.getBuildPkgs(pkgs, aur)
	if err != nil {
		return nil, err
	}

	if len(srcPkgs) == 0 {
		log.Print("All packages up to date, nothing to build")
		return nil, nil
	}

	buildPkgs, err := b.buildPkgs(srcPkgs)
	if err != nil {
		return nil, err
	}

	successLog(buildPkgs)
	return buildPkgs, nil
}

// Write packages built to the log.
func successLog(pkgs []*BuiltPkg) {
	var buf bytes.Buffer
	buf.WriteString("Built packages:")
	for _, pkg := range pkgs {
		buf.WriteString("\n * ")
		buf.WriteString(pkg.String())
	}

	log.Print(buf.String())
}

// Update build environment.
func (b *Builder) update() error {
	log.Printf("Updating packages")
	return runCmd(b.workdir, nil, "sudo", "pacman", "--sync", "--refresh", "--sysupgrade", "--noconfirm")
}

// Get a sorted list of packages to build.
func (b *Builder) getBuildPkgs(pkgs []string, aur *AUR) ([]*SrcPkg, error) {
	log.Printf("Fetching build sources+dependencies for %s", strings.Join(pkgs, ", "))
	pkgSrcs, err := aur.Get(pkgs)
	if err != nil {
		return nil, err
	}

	// Get a list of devel packages (-{bzr,git,svn,hg}) where an extra
	// version check is needed.
	updates := make([]*SrcPkg, 0, len(pkgSrcs))

	for _, pkgSrc := range pkgSrcs {
		if pkgSrc.PKGBUILD.IsDevel() {
			updates = append(updates, pkgSrc)
		}
	}

	err = b.updatePkgSrcs(updates)
	if err != nil {
		return nil, err
	}

	return b.repo.GetUpdated(pkgSrcs)
}

// update package sources.
func (b *Builder) updatePkgSrcs(pkgs []*SrcPkg) error {
	for _, pkg := range pkgs {
		_, err := b.updatePkgSrc(pkg)
		if err != nil {
			return err
		}
	}

	return nil
}

// Check and update if a newer source exist for the package.
func (b *Builder) updatePkgSrc(pkg *SrcPkg) (*SrcPkg, error) {
	p := pkg.PKGBUILD
	if len(p.Pkgnames) > 1 || p.Pkgnames[0] != p.Pkgbase {
		log.Printf("Checking for new version of %s:(%s)", p.Pkgbase, strings.Join(p.Pkgnames, ", "))
	} else {
		log.Printf("Checking for new version of %s", p.Pkgbase)
	}

	err := runCmd(pkg.Path, nil, "makepkg", "--nobuild", "--nodeps", "--noconfirm")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("mksrcinfo")
	cmd.Dir = pkg.Path
	if err != nil {
		return nil, err
	}

	filePath := path.Join(pkg.Path, ".SRCINFO")

	pkgb, err := pkgbuild.ParseSRCINFO(filePath)
	if err != nil {
		return nil, err
	}

	pkg.PKGBUILD = pkgb

	return pkg, nil
}

// Build a list of packages.
func (b *Builder) buildPkgs(pkgs []*SrcPkg) ([]*BuiltPkg, error) {
	buildPkgs := make([]*BuiltPkg, 0, len(pkgs))

	for _, pkg := range pkgs {
		pkgPaths, err := b.buildPkg(pkg)
		if err != nil {
			return nil, err
		}

		buildPkgs = append(buildPkgs, pkgPaths...)
	}

	return buildPkgs, nil
}

// Build package and return a list of resulting package archives.
func (b *Builder) buildPkg(pkg *SrcPkg) ([]*BuiltPkg, error) {
	p := pkg.PKGBUILD
	if len(p.Pkgnames) > 1 || p.Pkgnames[0] != p.Pkgbase {
		log.Printf("Building package %s:(%s)", p.Pkgbase, strings.Join(p.Pkgnames, ", "))
	} else {
		log.Printf("Building package %s", p.Pkgbase)
	}

	var env []string
	if b.config.Packager != "" {
		env = os.Environ()
		env = append(env, fmt.Sprintf("PACKAGER=%s", b.config.Packager))
	}

	err := runCmd(pkg.Path, env, "makepkg", "--install", "--syncdeps", "--noconfirm")
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(pkg.Path)
	if err != nil {
		return nil, err
	}

	pkgs := make([]*BuiltPkg, 0, 1)

	for _, f := range files {
		if strings.HasSuffix(f.Name(), "pkg.tar.xz") {
			builtPkg := &BuiltPkg{
				Pkg: path.Join(pkg.Path, f.Name()),
			}
			pkgs = append(pkgs, builtPkg)
		}
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), "pkg.tar.xz.sig") {
			for _, p := range pkgs {
				if path.Base(p.Pkg) == f.Name()[:len(f.Name())-4] {
					p.Signature = path.Join(pkg.Path, f.Name())
				}
			}
		}
	}

	return pkgs, nil
}
