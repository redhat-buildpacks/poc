package main

import (
	"context"
	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/image/v5/image"
	imgStorage "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"

	//"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/redhat-buildpacks/poc/buildah/build"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

const maxParallelDownloads = 0

func main() {
	ctx := context.TODO()

	if buildah.InitReexec() {
		return
	}
	unshare.MaybeReexecUsingUserNamespace(false)

	b := build.InitOptions()

	os.Setenv("BUILDAH_TEMP_DIR", b.TempDir)
	logrus.Infof("Buildah tempdir: %s", b.TempDir)

	dockerFileName := filepath.Join(b.WorkspaceDir, "Dockerfile")
	logrus.Infof("Dockerfile: %s", dockerFileName)

	// storeOptions, err := storage.DefaultStoreOptions(false,0)

	store, err := storage.GetStore(b.StoreOptions)
	if err != nil {
		logrus.Errorf("error creating buildah storage !", err)
		panic(err)
	}

	imageID, digest, err := imagebuildah.BuildDockerfiles(ctx, store, b.BuildOptions, dockerFileName)
	if err != nil {
		logrus.Errorf("Build image failed: %s", err.Error())
	}

	logrus.Infof("Image id: %s", imageID)
	logrus.Infof("Image digest: %s", digest.String())

	//rawSource, err := parseImageSource(ctx,name)
	rawSource, err := parseImageReference(ctx,digest.Name())
	if err != nil {
		logrus.Fatalf("Error parsing the image source", err)
	}

	src, err := image.FromSource(ctx, nil, rawSource)
	if err != nil {
		logrus.Fatalf("Error getting the image", err)
	}
	defer rawSource.Close()
	defer src.Close()

	rawManifest, _, err := src.Manifest(ctx)
	if err != nil {
		logrus.Fatalf("Error while getting the raw manifest", err)
	}
	logrus.Infof("Img manifest: %s",rawManifest)


	images, err := store.Images()
	if err != nil {
		logrus.Fatalf("Error reading store of images", err)
	}
	for _, img := range images {
		if (img.ID == imageID) {
			logrus.Infof("Image metadata: %s",img.Metadata)
			logrus.Infof("Top layer: %s",img.TopLayer)
		}
	}

	logrus.Info("Image built successfully :-)")
}

func parseImageSource(ctx context.Context, name string) (types.ImageSource, error) {
	ref, err := alltransports.ParseImageName(name)
	if err != nil {
		return nil, err
	}
	return ref.NewImageSource(ctx, newSystemContext())
}

func parseImageReference(ctx context.Context, name string) (types.ImageSource, error) {
	ref, err := imgStorage.Transport.ParseReference(name)
	if err != nil {
		return nil, err
	}
	return ref.NewImageSource(ctx, newSystemContext())
}

// newSystemContext returns a *types.SystemContext
func newSystemContext() *types.SystemContext {
	return &types.SystemContext{}
}
