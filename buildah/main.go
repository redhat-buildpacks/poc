package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/image/v5/image"
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

const (
	graphDriver = "vfs"
	repoType	= "containers-storage"
)

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

	storage := fmt.Sprintf("[%s@%s+%s]", graphDriver, b.StorageRootDir, b.StorageRunRootDir)
	containerStorageName := fmt.Sprintf("%s:%s%s",repoType,storage,imageID)
	ref, err := parseImageSource(ctx,containerStorageName)
	if err != nil {
		logrus.Fatalf("Error parsing the image source: %s", containerStorageName, err)
	}

	src, err := image.FromSource(ctx, nil, ref)
	if err != nil {
		logrus.Fatalf("Error getting the image", err)
	}
	defer ref.Close()
	defer src.Close()

	rawManifest, _, err := src.Manifest(ctx)
	if err != nil {
		logrus.Fatalf("Error while getting the raw manifest", err)
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, rawManifest, "", "    ")
	if err == nil {
		logrus.Infof("Image manifest: %s\n",&buf)
	}

	_, err = src.ConfigBlob(ctx)
	if err != nil {
		logrus.Fatalf("Error parsing ImageConfig", err)
	}

	config, err := src.OCIConfig(ctx)
	if err != nil {
		logrus.Fatalf("Error parsing OCI Config", err)
	}

	out, err := json.MarshalIndent(config, "", "    ")
	if err == nil {
		logrus.Infof("OCI Config: %s\n",string(out))
	}

	layers := src.LayerInfos()
	for _, info := range layers {
		logrus.Infof("Layer sha: %s\n",info.Digest.String())
	}


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

// newSystemContext returns a *types.SystemContext
func newSystemContext() *types.SystemContext {
	return &types.SystemContext{}
}
