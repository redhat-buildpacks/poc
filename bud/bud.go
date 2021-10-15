package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func main() {

	if buildah.InitReexec() {
		return
	}
	unshare.MaybeReexecUsingUserNamespace(false)

	ctx := context.TODO()
	dirname, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatal( err )
	}
	tempdir, err := ioutil.TempDir(dirname, "buildah-poc-")
	rootDir := filepath.Join(tempdir, "root")
	runrootDir := filepath.Join(tempdir, "runroot")
	contextDir := filepath.Join(tempdir, "context")

	logrus.Infof("Buildah tempdir : ",tempdir)

	currentDir, err := os.Getwd()
	if err != nil {
		logrus.Errorf("unable to choose current working directory as build context")
	}
	dockerFileName := filepath.Join(currentDir, "Dockerfile")
	logrus.Infof("Dockerfile name: ",dockerFileName)

	dateStamp := fmt.Sprintf("%d", time.Now().UnixNano())
	buildahImage := fmt.Sprintf("conformance-buildah:%s-%d", dateStamp, 1)

	var transientMounts []string

	output := &bytes.Buffer{}
	options := define.BuildOptions{
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

	// initialize storage for buildah
	storeOptions := storage.StoreOptions{
		GraphDriverName:     "overlay",
		GraphRoot:           rootDir,
		RunRoot:             runrootDir,
		RootlessStoragePath: rootDir,
	}
	// storeOptions, err := storage.DefaultStoreOptions(false,0)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		logrus.Errorf("error creating buildah storage !",err)
		panic(err)
	}

	imageID, _, err := imagebuildah.BuildDockerfiles(ctx, store, options, dockerFileName)
	if err != nil {
		logrus.Errorf("Build image failed: ",err.Error())
	}
	logrus.Infof("Image id: ",imageID)
}
