package main

import (
	"context"
	"fmt"
	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/redhat-buildpacks/poc/buildah/parse"
	"github.com/redhat-buildpacks/poc/buildah/util"
	"io/ioutil"
	"time"

	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
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
	repoType    = "containers-storage"
)

type globalOptions struct {
	debug              bool                    // Enable debug output
	policyPath         string                  // Path to a signature verification policy file
	insecurePolicy     bool                    // Use an "allow everything" signature verification policy
	registriesDirPath  string                  // Path to a "registries.d" registry configuration directory
	overrideArch       string                  // Architecture to use for choosing images, instead of the runtime one
	overrideOS         string                  // OS to use for choosing images, instead of the runtime one
	overrideVariant    string                  // Architecture variant to use for choosing images, instead of the runtime one
	commandTimeout     time.Duration           // Timeout for the command execution
	registriesConfPath string                  // Path to the "registries.conf" file
	tmpDir             string                  // Path to use for big temporary files
}

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

	// GetStore attempts to find an already-created Store object matching the
	// specified location and graph driver, and if it can't, it creates and
	// initializes a new Store object, and the underlying storage that it controls.
	store, err := storage.GetStore(b.StoreOptions)
	if err != nil {
		logrus.Errorf("error creating buildah storage !", err)
		panic(err)
	}

	/* Parse the content of the Dockerfile to execute the different commands: FROM, RUN, ...
	   Return the:
	   - imageID: id of the new image created. String of 64 chars.
	     NOTE: The first 12 chars corresponds to the `id` displayed using `sudo buildah --storage-driver vfs images`
	   - digest: image repository name prefixed "localhost/". e.g: localhost/buildpack-buildah:TAG@sha256:64_CHAR_SHA
	*/
	imageID, digest, err := imagebuildah.BuildDockerfiles(ctx, store, b.BuildOptions, dockerFileName)
	if err != nil {
		logrus.Errorf("Build image failed: %s", err.Error())
	}

	logrus.Infof("Image id: %s", imageID)
	logrus.Infof("Image digest: %s", digest.String())

	/* Converts a URL-like image name to a types.ImageReference
		   and create an imageSource
	       NOTE: An imageSource is a service, possibly remote (= slow), to download components of a single image or a named image set (manifest list).
	       This is primarily useful for copying images around; for examining their properties, Image (below)
	*/
	ref, err := parseImageSource(ctx, containerStorageName(b, imageID))
	if err != nil {
		logrus.Fatalf("Error parsing the image source: %s", containerStorageName, err)
	}

	// Create a FromSource object to read the image content
	src, err := image.FromSource(ctx, nil, ref)
	if err != nil {
		logrus.Fatalf("Error getting the image", err)
	}
	defer ref.Close()
	defer src.Close()

	// Get the Image Manifest and log it as JSON indented string
	// See spec: https://docs.docker.com/registry/spec/manifest-v2-2/#image-manifest
	rawManifest, _, err := src.Manifest(ctx)
	if err != nil {
		logrus.Fatalf("Error while getting the raw manifest", err)
	}
	parse.JsonIndent("Image manifest",rawManifest)

	// Get the OCIConfig configuration as per OCI v1 image-spec.
	// Log it as JSON indented string
	config, err := src.OCIConfig(ctx)
	if err != nil {
		logrus.Fatalf("Error parsing OCI Config", err)
	}
	parse.JsonMarshal("OCI Config",config)

	// Get the layers from the source and log the Layer SHA
	layers := src.LayerInfos()
	for _, info := range layers {
		logrus.Infof("Layer sha: %s\n", info.Digest.String())
	}

	images, err := store.Images()
	if err != nil {
		logrus.Fatalf("Error reading store of images", err)
	}
	for _, img := range images {
		if img.ID == imageID {
			logrus.Infof("Image metadata: %s", img.Metadata)
			logrus.Infof("Top layer: %s", img.TopLayer)
		}
	}

	logrus.Infof("Image repositry id: %s",imageID[0:11])
	logrus.Info("Image built successfully :-)")

	// Let's try to copy the layers

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

func containerStorageName(b *build.BuildahParameters, imageID string) string {
	storage := fmt.Sprintf("[%s@%s+%s]", graphDriver, b.StorageRootDir, b.StorageRunRootDir)
	return fmt.Sprintf("%s:%s%s", repoType, storage, imageID)
}

// getPolicyContext returns a *signature.PolicyContext based on opts.
func (opts *globalOptions) getPolicyContext() (*signature.PolicyContext, error) {
	var policy *signature.Policy // This could be cached across calls in opts.
	var err error
	if opts.insecurePolicy {
		policy = &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	} else if opts.policyPath == "" {
		policy, err = signature.DefaultPolicy(nil)
	} else {
		policy, err = signature.NewPolicyFromFile(opts.policyPath)
	}
	if err != nil {
		return nil, err
	}
	return signature.NewPolicyContext(policy)
}

func (opts *globalOptions) copyImage(imageID string) {
	initGlobalOptions()

	policyContext, err := opts.getPolicyContext()
	if err != nil {
		logrus.Fatalf("Error loading trust policy: %v", err)
	}
	defer policyContext.Destroy()

	srcURL := "oci://" + util.GetPWD() + "/" + imageID[0:11] + ":latest"
	srcRef, err := alltransports.ParseImageName(srcURL)
	if err != nil {
		logrus.Fatalf("Invalid source name %s: %v", srcURL, err)
	}

	destURL := "oci://" + util.GetPWD() + "/" + imageID[0:11] + ":latest"
	destRef, err := alltransports.ParseImageName(destURL)
	if err != nil {
		logrus.Fatalf("Invalid destination name %s: %v", destURL, err)
	}

	// copy image
	_, err = copy.Image(context.TODO(), policyContext, destRef, srcRef, &copy.Options{
		RemoveSignatures:      false,
		SignBy:                "",
		ReportWriter:          nil,
		SourceCtx:             nil,
		DestinationCtx:        nil,
		ForceManifestMIMEType: parseManifestFormat("oci"),
		ImageListSelection:    copy.CopySystemImage,
		OciDecryptConfig:      nil,
		OciEncryptLayers:      nil,
		OciEncryptConfig:      nil,
	})
	if err != nil {
		logrus.Fatalf("Image not copied :-(")
	}
}

func initGlobalOptions() (*globalOptions) {
	return &globalOptions{}
}

// parseManifestFormat parses format parameter for copy and sync command.
// It returns string value to use as manifest MIME type
func parseManifestFormat(manifestFormat string) (string) {
	switch manifestFormat {
	case "oci":
		return imgspecv1.MediaTypeImageManifest
	case "v2s1":
		return manifest.DockerV2Schema1SignedMediaType
	case "v2s2":
		return manifest.DockerV2Schema2MediaType
	default:
		logrus.Errorf("unknown format %q. Choose one of the supported formats: 'oci', 'v2s1', or 'v2s2'", manifestFormat)
		return ""
	}
}