package main

import (
	"context"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	istorage "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/storage/pkg/unshare"
	"github.com/redhat-buildpacks/poc/buildah/build"
	"github.com/redhat-buildpacks/poc/buildah/logging"
	"github.com/redhat-buildpacks/poc/buildah/model"
	"github.com/redhat-buildpacks/poc/buildah/parse"
	"github.com/redhat-buildpacks/poc/buildah/util"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

const (
	LOGGING_LEVEL_ENV_NAME     = "LOGGING_LEVEL"
	LOGGING_FORMAT_ENV_NAME    = "LOGGING_FORMAT"
	LOGGING_TIMESTAMP_ENV_NAME = "LOGGING_TIMESTAMP"
	EXTRACT_LAYERS_ENV_NAME    = "EXTRACT_LAYERS"
	FILES_TO_SEARCH_ENV_NAME   = "FILES_TO_SEARCH"

	DefaultLevel        = "info"
	DefaultLogTimestamp = false
	DefaultLogFormat    = "text"

	graphDriver = "vfs"
	repoType    = "containers-storage"
)

var (
	logLevel      string   // Log level (trace, debug, info, warn, error, fatal, panic)
	logFormat     string   // Log format (text, color, json)
	logTimestamp  bool     // Timestamp in log output
	extractLayers bool     // Extract layers from tgz files. Default is false
	filesToSearch []string // List of files to search to check if they exist under the updated FS
	opts		  globalOptions
	b             *build.BuildahParameters //
)

type globalOptions struct {
	debug              bool           // Enable debug output
	policyPath         string         // Path to a signature verification policy file
	insecurePolicy     bool           // Use an "allow everything" signature verification policy
	registriesDirPath  string         // Path to a "registries.d" registry configuration directory
	overrideArch       string         // Architecture to use for choosing images, instead of the runtime one
	overrideOS         string         // OS to use for choosing images, instead of the runtime one
	overrideVariant    string         // Architecture variant to use for choosing images, instead of the runtime one
	commandTimeout     time.Duration  // Timeout for the command execution
	registriesConfPath string         // Path to the "registries.conf" file
	tmpDir             string         // Path to use for big temporary files
	metadata           model.Metadata // Metadata file containing the data populated by the Lifecycle builder
}

func initLog() {
	logLevel = util.GetValFromEnVar(LOGGING_LEVEL_ENV_NAME)
	if logLevel == "" {
		logLevel = DefaultLevel
	}

	logFormat = util.GetValFromEnVar(LOGGING_FORMAT_ENV_NAME)
	if logFormat == "" {
		logFormat = DefaultLogFormat
	}

	loggingTimeStampStr := util.GetValFromEnVar(LOGGING_TIMESTAMP_ENV_NAME)
	if loggingTimeStampStr == "" {
		logTimestamp = DefaultLogTimestamp
	} else {
		v, err := strconv.ParseBool(loggingTimeStampStr)
		if err != nil {
			logrus.Fatalf("logTimestamp bool assignment failed %s", err)
		} else {
			logTimestamp = v
		}
	}
	err := logging.Configure(logLevel, logFormat, logTimestamp)
	if err != nil {
		logrus.Fatalf("Error creating logging !", err)
	}
}

func initGlobalVar() {
	extractLayersStr := util.GetValFromEnVar(EXTRACT_LAYERS_ENV_NAME)
	if extractLayersStr == "" {
		logrus.Info("The layered tzg files will NOT be extracted to the home dir ...")
		extractLayers = false
	} else {
		v, err := strconv.ParseBool(extractLayersStr)
		if err != nil {
			logrus.Fatalf("extractLayers bool assignment failed %s", err)
		} else {
			extractLayers = v
			logrus.Info("The layered tar-GZip files will be extracted to the home dir ...")
		}
	}

	filesToSearchStr := util.GetValFromEnVar(FILES_TO_SEARCH_ENV_NAME)
	if filesToSearchStr != "" {
		filesToSearch = strings.Split(filesToSearchStr, ",")
	}
}

// TODO: To be documented
func main() {
	if buildah.InitReexec() {
		return
	}
	unshare.MaybeReexecUsingUserNamespace(true)

	// Configure the Logger with ENV vars or default values
	initLog()

	// Init the variables of the application using the ENV var
	initGlobalVar()

	// TODO: To be reviewed and perhaps merged with initGlobalVar
	opts := initGlobalOptions()

	if _, ok := os.LookupEnv("DEBUG"); ok && (len(os.Args) <= 1 || os.Args[1] != "from-debugger") {
		args := []string{
			"--listen=:2345",
			"--headless=true",
			"--api-version=2",
			"--accept-multiclient",
			"exec",
			"/buildah-app", "from-debugger",
		}
		err := syscall.Exec("/usr/local/bin/dlv", append([]string{"/usr/local/bin/dlv"}, args...), os.Environ())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	// TODO: Check how we could continue to use the debugger as the following code exec a sub-command and by consequence it exits
	//hasCapSysAdmin, err := unshare.HasCapSysAdmin()
	//if err != nil {
	//	logrus.Fatalf("error checking for CAP_SYS_ADMIN: %v", err)
	//}
	//unshare.MaybeReexecUsingUserNamespace(!hasCapSysAdmin)
	// unshare.MaybeReexecUsingUserNamespace(false)

	b = build.InitOptions()
	b.ExtractLayers = extractLayers

	os.Setenv("BUILDAH_TEMP_DIR", b.TempDir)
	logrus.Infof("Buildah tempdir: %s", b.TempDir)

	metadatafileNameToParse := os.Getenv("METADATA_FILE_NAME")

	dockerfileNameToParse := os.Getenv("DOCKERFILE_NAME")
	if dockerfileNameToParse == "" {
		dockerfileNameToParse = "Dockerfile"
	}

	// TODO: Check how to use this function using DLV debugger
	err := reapChildProcesses()
	if err != nil {
		logrus.Fatal(err)
	}

	// Parse the Metadata toml file
	if metadatafileNameToParse != "" {
		// TODO : Create a var to specify the layers path
		if _, err := toml.DecodeFile(filepath.Join(b.WorkspaceDir, "layers", metadatafileNameToParse), &opts.metadata); err != nil {
			logrus.Fatal(err)
		}
		for _, dockerFile := range opts.metadata.Dockerfiles {
			pathToDockerFile := filepath.Join(b.WorkspaceDir, dockerFile.Path)
			logrus.Infof("Dockerfile path: %s", pathToDockerFile)

			// Process now the Dockerfile
			processDockerfile(pathToDockerFile)
		}
	} else {
		// When no metadata.toml file is used, parse the dockerfile directly
		dockerFileName := filepath.Join(b.WorkspaceDir, dockerfileNameToParse)
		logrus.Infof("Dockerfile path: %s", dockerFileName)

		// Process now the Dockerfile
		processDockerfile(dockerFileName)
	}
}

func processDockerfile(pathToDockerFile string) {
	ctx := context.TODO()

	// GetStore attempts to find an already-created Store object matching the
	// specified location and graph driver, and if it can't, it creates and
	// initializes a new Store object, and the underlying storage that it controls.
	store, err := storage.GetStore(b.StoreOptions)
	if err != nil {
		logrus.Fatal("error creating buildah storage !", err)
	}

	// Launch a timer to measure the time needed to parse/copy/extract
	start := time.Now()

	/* Parse the content of the Dockerfile to execute the different commands: FROM, RUN, ...
	   Return the:
	   - imageID: id of the new image created. String of 64 chars.
	     NOTE: The first 12 chars corresponds to the `id` displayed using `sudo buildah --storage-driver vfs images`
	   - digest: image repository name prefixed "localhost/". e.g: localhost/buildpack-buildah:TAG@sha256:64_CHAR_SHA
	*/
	imageID, digest, err := imagebuildah.BuildDockerfiles(ctx, store, b.BuildOptions, pathToDockerFile)
	if err != nil {
		logrus.Fatalf("Build image failed: %s", err)
	}

	logrus.Infof("Image id: %s", imageID)
	logrus.Infof("Image digest: %s", digest.String())

	/* Converts a URL-like image name to a types.ImageReference
		   and create an imageSource
	       NOTE: An imageSource is a service, possibly remote (= slow), to download components of a single image or a named image set (manifest list).
	       This is primarily useful for copying images around; for examining their properties, Image (below)
	*/
	ref, err := istorage.Transport.NewStoreReference(store, nil, imageID)
	if err != nil {
		logrus.Fatalf("Error parsing the image source: %s", imageID, err)
	}

	// Show the content of the Image MANIFEST stored under the local storage
	// ShowRawManifestContent(ref)

	// Show the OCI content of the Image
	//ShowOCIContent(ref)

	logrus.Infof("Image repository id: %s", imageID[0:11])
	logrus.Info("Image built successfully :-)")

	// Let's try to copy the layers from the local storage to the local Cache volume as
	// OCI folder
	ociImageReference, err := CopyImage(ref, imageID)
	if err != nil {
		logrus.Fatalf("Image not copied from local storage to OCI path.", err)
	}

	// Get the path of the new layer file created under OCI:///
	pathOCINewLayer := GetPathLayerTarGZIpfile(ociImageReference, imageID)

	if b.ExtractLayers {
		b.ExtractTGZFile(pathOCINewLayer)
	}

	// Check if files exist
	if len(filesToSearch) > 0 {
		util.FindFiles(filesToSearch)
	}

	// Time elapsed is ...
	logrus.Infof("Time elapsed: %s", time.Since(start))
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

func ShowRawManifestContent(ref types.ImageReference) {
	// Create a FromSource object to read the image content
	src, err := ref.NewImage(context.TODO(), nil)
	if err != nil {
		logrus.Fatalf("Error getting the image", err)
	}
	defer src.Close()

	// Get the Image Manifest and log it as JSON indented string
	// See spec: https://docs.docker.com/registry/spec/manifest-v2-2/#image-manifest
	rawManifest, _, err := src.Manifest(context.TODO())
	if err != nil {
		logrus.Fatalf("Error while getting the raw manifest", err)
	}
	parse.JsonIndent("Image manifest", rawManifest)
}

func ShowOCIContent(ref types.ImageReference) {
	// Create a FromSource object to read the image content
	src, err := ref.NewImage(context.TODO(), nil)
	if err != nil {
		logrus.Fatalf("Error getting the image", err)
	}
	defer src.Close()
	// Get the OCIConfig configuration as per OCI v1 image-spec.
	// Log it as JSON indented string
	config, err := src.OCIConfig(context.TODO())
	if err != nil {
		logrus.Fatalf("Error parsing OCI Config", err)
	}
	parse.JsonMarshal("OCI Config", config)
}

func CopyImage(srcRef types.ImageReference, imageID string) (types.ImageReference, error) {

	policyContext, err := opts.getPolicyContext()
	if err != nil {
		return nil, err
	}
	defer policyContext.Destroy()

	destURL := "oci:///cache/" + imageID[0:11] + ":latest"
	destRef, err := alltransports.ParseImageName(destURL)
	if err != nil {
		return nil, err
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
		return nil, err
	} else {
		logrus.Infof("Image copied to %s", destURL)
	}
	return destRef, nil
}

func GetPathLayerTarGZIpfile(destRef types.ImageReference, imageID string) string {
	src, err := destRef.NewImageSource(context.TODO(), nil)
	if err != nil {
		logrus.Fatalf("Image source cannot be created", err)
	}

	defer func() {
		if err := src.Close(); err != nil {
			logrus.Fatalf("Could not close image", err)
		}
	}()

	// TODO: Should only logged for debugging purpose
	ShowRawManifestContent(destRef)

	img, err := image.FromUnparsedImage(context.TODO(), nil, image.UnparsedInstance(src, nil))
	if err != nil {
		logrus.Fatalf("Error parsing manifest for image", err)
	}
	// Get the layers from the source and log the Layer SHA
	blobs := img.LayerInfos()
	for _, blobInfo := range blobs {
		logrus.Infof("Layer blobInfo: %s\n", blobInfo.Digest.String())
	}

	// Get the last layer from the Layers as it corresponds to our new image
	lastLayer := blobs[len(blobs)-1]
	sha := lastLayer.Digest.Hex()
	logrus.Infof("Last layer: %s", sha)
	pathTarGZipLayer := "/cache/" + imageID[0:11] + "/blobs/sha256/" + sha
	logrus.Infof("Path to the TarGzipLayer file: %s", pathTarGZipLayer)

	return pathTarGZipLayer
}

func initGlobalOptions() *globalOptions {
	return &globalOptions{}
}

// parseManifestFormat parses format parameter for copy and sync command.
// It returns string value to use as manifest MIME type
func parseManifestFormat(manifestFormat string) string {
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

func reapChildProcesses() error {
	procDir, err := os.Open("/proc")
	if err != nil {
		return err
	}

	procDirs, err := procDir.Readdirnames(-1)
	if err != nil {
		return err
	}

	tid := os.Getpid()

	var wg sync.WaitGroup
	for _, dirName := range procDirs {
		pid, err := strconv.Atoi(dirName)
		if err == nil && pid != 1 && pid != tid {
			p, err := os.FindProcess(pid)
			if err != nil {
				continue
			}
			err = p.Signal(syscall.SIGTERM)
			if err != nil {
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.Wait()
			}()
		}
	}
	wg.Wait()
	return nil
}
