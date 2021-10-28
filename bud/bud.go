package main

import (
	"context"
	"fmt"
	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
	"me.snowdrop/bud/util"
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
	logrus.Infof("Image id : ",imageID)

	builder, err := buildah.ImportBuilderFromImage(context.TODO(), store,buildah.ImportFromImageOptions{Image: imageID})
	if err != nil {
		panic(err)
	}

	imageRef, err := is.Transport.ParseStoreReference(store, imageID)
	if err != nil {
		logrus.Errorf( "no such image %q", imageID)
	}

	imageId, _, _, err := builder.Commit(context.TODO(), imageRef, buildah.CommitOptions{})

	fmt.Printf("Image built! %s\n", imageId)
}