package pkg

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/howardjohn/istio-release/pkg/model"
	"github.com/howardjohn/istio-release/pkg/util"

	"istio.io/pkg/log"
)

// Sources will copy all dependencies require, pulling from Github if required, and set up the working tree.
// This includes locally tagging all git repos with the version being built, so that the right version is present in binaries.
func Sources(manifest model.Manifest) error {
	manifest.Directory = SetupWorkDir()

	for _, dependency := range manifest.Dependencies {
		// Fetch the dependency
		if err := util.Clone(dependency, path.Join(manifest.SourceDir(), dependency.Repo)); err != nil {
			return fmt.Errorf("failed to resolve %+v: %v", dependency, err)
		}
		log.Infof("Resolved %v", dependency.Repo)

		// Also copy it to the working directory
		src := path.Join(manifest.SourceDir(), dependency.Repo)
		if err := util.CopyDir(src, manifest.RepoDir(dependency.Repo)); err != nil {
			return fmt.Errorf("failed to copy dependency %v to working directory: %v", dependency.Repo, err)
		}

		// Tag the repo. This allows the build process to look at the git tag for version information
		if err := TagRepo(manifest, manifest.RepoDir(dependency.Repo)); err != nil {
			return fmt.Errorf("failed to tag repo %v: %v", dependency.Repo, err)
		}
	}
	return nil
}

// The release expects a working directory with:
// * sources/ contains all of the sources to build from. These should not be modified
// * work/ initially contains all the sources, but may be modified during the build
// * out/ contains all final artifacts
func SetupWorkDir() string {
	tmpdir, err := ioutil.TempDir(os.TempDir(), "istio-release")
	if err != nil {
		log.Fatalf("failed to create working directory: %v", err)
	}
	if err := os.Mkdir(path.Join(tmpdir, "sources"), 0750); err != nil {
		log.Fatalf("failed to set up working directory: %v", err)
	}
	if err := os.Mkdir(path.Join(tmpdir, "work"), 0750); err != nil {
		log.Fatalf("failed to set up working directory: %v", err)
	}
	if err := os.Mkdir(path.Join(tmpdir, "out"), 0750); err != nil {
		log.Fatalf("failed to set up working directory: %v", err)
	}
	return tmpdir
}

// TagRepo tags a given git repo with the version from the manifest.
func TagRepo(manifest model.Manifest, repo string) error {
	cmd := util.VerboseCommand("git", "tag", manifest.Version)
	cmd.Dir = repo
	return cmd.Run()
}

// StandardizeManifest will convert a manifest to a fixed SHA, rather than a branch
// This allows outputting the exact version used after the build is complete
func StandardizeManifest(manifest *model.Manifest) error {
	for i, dep := range manifest.Dependencies {
		buf := bytes.Buffer{}
		cmd := util.VerboseCommand("git", "rev-parse", "HEAD")
		cmd.Stdout = &buf
		cmd.Dir = manifest.RepoDir(dep.Repo)
		if err := cmd.Run(); err != nil {
			return err
		}
		dep.Sha = strings.TrimSpace(buf.String())
		dep.Branch = ""
		manifest.Dependencies[i] = dep
	}
	return nil
}
