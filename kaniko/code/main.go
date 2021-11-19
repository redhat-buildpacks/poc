package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-buildpacks/poc/kaniko/logging"
	util "github.com/redhat-buildpacks/poc/kaniko/util"
	logrus "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	kanikoDir                    = "/kaniko"
	cacheDir                     = "/cache"
	workspaceDir                 = "/workspace"
	defaultDockerFileName        = "Dockerfile"
	DOCKER_FILE_NAME_ENV_NAME    = "DOCKER_FILE_NAME"
	LOGGING_LEVEL_ENV_NAME       = "LOGGING_LEVEL"
	LOGGING_FORMAT_ENV_NAME      = "LOGGING_FORMAT"
	LOGGING_TIMESTAMP_ENV_NAME   = "LOG_TIMESTAMP"

	DefaultLevel              = "info"
	DefaultLogTimestamp       = false
	DefaultLogFormat          = "text"
)

var (
	logLevel	 string // Log level (trace, debug, info, warn, error, fatal, panic)
	logFormat    string // Log format (text, color, json)
	logTimestamp bool   // Timestamp in log output
)

type buildPackConfig struct {
	layerPath       string
	tarPath         string
	cacheDir       	string
	kanikoDir       string
	workspaceDir    string
	dockerFileName	string
	opts            config.KanikoOptions
	newImage		v1.Image
}

func init() {
	logLevel = util.GetValFromEnVar(LOGGING_LEVEL_ENV_NAME)
	if logLevel == "" {
		logLevel = DefaultLevel
	}

	logFormat = util.GetValFromEnVar(LOGGING_FORMAT_ENV_NAME)
	if logFormat == "" {
		logFormat = DefaultLogFormat
	}

	// TODO: Check how to process bool env var
	logTimestamp = DefaultLogTimestamp

	if err := logging.Configure(logLevel, logFormat, logTimestamp); err != nil {
		panic(err)
	}
}

func main() {
	logrus.Info("Starting the Kaniko application to process a Dockerfile ...")

	// Create a buildPackConfig and set the default values
	logrus.Info("Initialize the BuildPackConfig and set the defaults values ...")
	b := newBuildPackConfig()
	b.initDefaults()
	logrus.Infof("Kaniko      dir: %s",b.kanikoDir)
	logrus.Infof("Workspace   dir: %s",b.workspaceDir)
	logrus.Infof("Cache       dir: %s",b.cacheDir)
	logrus.Infof("Dockerfile name: %s",b.dockerFileName)

	// Build the Dockerfile
	logrus.Debugf("Building the %s",b.dockerFileName)
	err := b.buildDockerFile()
	if err != nil {
		panic(err)
	}

	// Save the Config and Manifest files of the new image created
	b.saveImageJSONConfig()
	b.saveImageRawManifest()

	// Log the content of the Kaniko dir
	logrus.Debugf("Reading dir content of: %s", kanikoDir)
	util.ReadFilesFromPath(kanikoDir)

	// Export the layers as tar gzip files under the cache dir
	logrus.Debugf("Export the layers as tar gzip files under the %s ...",b.cacheDir)
	b.copyLayersTarFileToCacheDir(b.newImage)
}

func newBuildPackConfig() *buildPackConfig {
	return &buildPackConfig{
		layerPath: "",
		tarPath: "",
		cacheDir: cacheDir,
		workspaceDir: workspaceDir,
		kanikoDir: kanikoDir,
	}
}

func (b *buildPackConfig) initDefaults() {
	logrus.Debug("Checking if the DOCKER_FILE_NAME env is defined...")
	b.dockerFileName = util.GetValFromEnVar(DOCKER_FILE_NAME_ENV_NAME)
	if b.dockerFileName != "" {
		b.dockerFileName = defaultDockerFileName
	}
	logrus.Debugf("DockerfileName is: %s", b.dockerFileName)

	dockerFilePath := b.workspaceDir + "/" + b.dockerFileName

	b.opts = config.KanikoOptions{
		CacheOptions:   config.CacheOptions{CacheDir: cacheDir},
		DockerfilePath: dockerFilePath,
		IgnoreVarRun:   true,
		NoPush:         true,
		SrcContext:     b.workspaceDir,
		SnapshotMode:   "full",
	}

	logrus.Debug("KanikoOptions defined")
}

func (b *buildPackConfig) buildDockerFile() (err error) {

	logrus.Debugf("Moving to kaniko home dir: %s", b.kanikoDir)
	if err := os.Chdir(b.kanikoDir); err != nil {
		panic(err)
	}

	logrus.Debugf("Building the %s ...", b.dockerFileName)
	b.newImage, err = executor.DoBuild(&b.opts)
	return err
}

func (b *buildPackConfig) copyLayersTarFileToCacheDir(image v1.Image) {
	// Get layers
	layers, err := b.newImage.Layers()
	if err != nil {
		panic(err)
	}
	logrus.Infof("Generated %d layers\n", len(layers))
	for _, layer := range layers {
		digest, err := layer.Digest()
		digest.MarshalText()
		if err != nil {
			panic(err)
		}
		b.layerPath = filepath.Join(b.kanikoDir, digest.String()+".tgz")
		logrus.Infof("Tar layer file: %s\n", b.layerPath)
		err = saveLayer(layer, b.layerPath)
		if err != nil {
			panic(err)
		}
	}

	// Copy the content of the kanikoDir to the cacheDir
	util.Dir(kanikoDir, cacheDir)
}

func (b *buildPackConfig) saveImageRawManifest() {
	rawManifest, err := b.newImage.RawManifest()
	rawManifestFilePath := b.cacheDir + "/manifest.json"
	err = ioutil.WriteFile(rawManifestFilePath, rawManifest, 0644)
	if err != nil {
		panic(err)
	}
	logrus.Debugf("Manifest file of the new image stored at %s",rawManifestFilePath)
}

func (b *buildPackConfig) saveImageJSONConfig() {
	// Get the Image config file
	configJSON, err := b.newImage.ConfigFile()
	if err != nil {
		panic(err)
	}
	configPath := filepath.Join(b.kanikoDir, "config.json")
	c, err := os.Create(configPath)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	err = json.NewEncoder(c).Encode(*configJSON)
	if err != nil {
		panic(err)
	}
	logrus.Debugf("Image JSON config file stored at %s",configPath)

	// Log the image json config
	// TODO: Add a debug opt to log if needed
	// readFileContent(c)
}

func (b *buildPackConfig) untarFile(tgzFile string) (err error) {
	// UnGzip first the tgz file
	gzf, err := unGzip(tgzFile, b.kanikoDir)
	if err != nil {
		logrus.Panicf("Something wrong happened ... %s", err)
	}

	// Open the tar file
	tr := tar.NewReader(gzf)
	// Get each tar segment
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// determine proper file path info
		logrus.Infof("File extracted: %s", hdr.Name)
	}
	return nil
}

func unGzip(gzipFile, tarPath string) (gzf io.Reader, err error) {
	logrus.Infof("Opening the gzip file: %s", gzipFile)
	f, err := os.Open(gzipFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	logrus.Infof("Creating a gzip reader for: %s", f.Name())
	gzf, err = gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	return gzf, nil
}

func saveLayer(layer v1.Layer, path string) error {
	layerReader, err := layer.Compressed()
	if err != nil {
		return err
	}
	l, err := os.Create(path)
	if err != nil {
		return err
	}
	defer l.Close()
	_, err = io.Copy(l, layerReader)
	if err != nil {
		return err
	}
	return nil
}
