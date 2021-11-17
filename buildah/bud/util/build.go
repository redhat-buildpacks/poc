package util

import (
	"bytes"
	"fmt"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/storage"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"
)

type BuildahParameters struct {
	BuildOptions define.BuildOptions
    StoreOptions storage.StoreOptions
	TempDir      string
	WorkspaceDir string
	StorageRootDir    string
	StorageRunRootDir string
	GraphDriverName   string
}

func InitOptions() BuildahParameters {
	b := BuildahParameters{}

	b.WorkspaceDir = os.Getenv("WORKSPACE_DIR")
	if b.WorkspaceDir == "" {
		b.WorkspaceDir = "/workspace/buildah-layers"
	}
	logrus.Infof("WORKSPACE DIR: %s", b.WorkspaceDir)

	b.GraphDriverName = os.Getenv("GRAPH_DRIVER")
	if b.GraphDriverName == "" {
		b.GraphDriverName = "vfs"
	}
	logrus.Infof("GRAPH_DRIVER: %s", b.GraphDriverName)

	b.StorageRootDir = os.Getenv("STORAGE_ROOT_PATH")
	if b.StorageRootDir == "" {
		b.StorageRootDir = "/var/lib/containers/storage"
	}
	logrus.Infof("STORAGE ROOT PATH: %s", b.StorageRootDir)

	b.StorageRunRootDir = os.Getenv("STORAGE_RUN_ROOT_PATH")
	if b.StorageRunRootDir == "" {
		b.StorageRunRootDir = "/var/run/containers/storage"
	}
	logrus.Infof("STORAGE RUN ROOT PATH: %s", b.StorageRunRootDir)

	var transientMounts []string
	b.TempDir = filepath.Join(b.WorkspaceDir,"buildah-layers") // ioutil.TempDir(b.WorkspaceDir, "buildah-poc-")
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
		GraphDriverName:     b.GraphDriverName,
		GraphRoot:           b.StorageRootDir,
		RunRoot:             b.StorageRunRootDir,
		RootlessStoragePath: b.StorageRootDir,
	}
	return b
}

