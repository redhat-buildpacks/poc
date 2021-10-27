package main

import (
	"context"
	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
	"me.snowdrop/bud/util"
	"path/filepath"
)

func main() {

	ctx := context.TODO()

	if buildah.InitReexec() {
		return
	}
	unshare.MaybeReexecUsingUserNamespace(false)

	b := util.InitOptions()

	logrus.Infof("Buildah tempdir : ",b.TempDir)

	dockerFileName := filepath.Join(b.WorkspaceDir, "Dockerfile")
	logrus.Infof("Dockerfile : ",dockerFileName)

	// storeOptions, err := storage.DefaultStoreOptions(false,0)

	store, err := storage.GetStore(b.StoreOptions)
	if err != nil {
		logrus.Errorf("error creating buildah storage !",err)
		panic(err)
	}

	imageID, _, err := imagebuildah.BuildDockerfiles(ctx, store, b.BuildOptions, dockerFileName)
	if err != nil {
		logrus.Errorf("Build image failed: ",err.Error())
	}
	logrus.Infof("Image id: ",imageID)
}
