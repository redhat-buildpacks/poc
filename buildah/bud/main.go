package main

import (
	"context"
	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
	"github.com/redhat-buildpacks/poc/buildah/bud/util"
	"os"
	"path/filepath"
)

func main() {

	ctx := context.TODO()

	if buildah.InitReexec() {
		return
	}
	unshare.MaybeReexecUsingUserNamespace(false)

	b := util.InitOptions()

	os.Setenv("BUILDAH_TEMP_DIR", b.TempDir)
	logrus.Infof("Buildah tempdir: %s",b.TempDir)

	dockerFileName := filepath.Join(b.WorkspaceDir, "Dockerfile")
	logrus.Infof("Dockerfile: %s",dockerFileName)

	// storeOptions, err := storage.DefaultStoreOptions(false,0)

	store, err := storage.GetStore(b.StoreOptions)
	if err != nil {
		logrus.Errorf("error creating buildah storage !",err)
		panic(err)
	}

	imageID, _, err := imagebuildah.BuildDockerfiles(ctx, store, b.BuildOptions, dockerFileName)
	if err != nil {
		logrus.Errorf("Build image failed: %s",err.Error())
	}
	logrus.Infof("Image id: %s",imageID)
	logrus.Info("Image built successfully :-)" )
}