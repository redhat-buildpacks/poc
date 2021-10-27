package util

import (
	"bytes"
	"fmt"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/storage"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type BuildParameter struct {
	BuildOptions define.BuildOptions
    StoreOptions storage.StoreOptions
	TempDir      string
}

func InitOptions() BuildParameter {
	b := BuildParameter{}

	workspaceDir := os.Getenv("WORKSPACE_DIR")
	logrus.Infof("WORKSPACE DIR: ", workspaceDir)

	graphDriverName := os.Getenv("STORAGE_DRIVER")
	if graphDriverName == "" {
		graphDriverName = "vfs"
	}

	var transientMounts []string
	b.TempDir, _ = ioutil.TempDir(workspaceDir, "buildah-poc-")
	rootDir := filepath.Join(b.TempDir, "root")
	runrootDir := filepath.Join(b.TempDir, "runroot")
	contextDir := filepath.Join(b.TempDir, "context")

	dateStamp := fmt.Sprintf("%d", time.Now().UnixNano())
	buildahImage := fmt.Sprintf("buildpack-buildah:%s-%d", dateStamp, 1)

	output := &bytes.Buffer{}

	// Define image build options
	b.BuildOptions = define.BuildOptions{
		ContextDirectory: contextDir,
		CommonBuildOpts:  &define.CommonBuildOptions{},
		NamespaceOptions: []define.NamespaceOption{{
			Name: string(rspec.NetworkNamespace),
			Host: true,
		}},
		TransientMounts:         transientMounts,
		Output:                  buildahImage,
		OutputFormat:            buildah.Dockerv2ImageManifest,
		Out:                     output,
		Err:                     output,
		Layers:                  true,
		NoCache:                 true,
		RemoveIntermediateCtrs:  true,
		ForceRmIntermediateCtrs: true,
	}

	// Initialize storage for buildah
	b.StoreOptions = storage.StoreOptions{
		GraphDriverName:     graphDriverName,
		GraphRoot:           rootDir,
		RunRoot:             runrootDir,
		RootlessStoragePath: rootDir,
	}
	return b
}

