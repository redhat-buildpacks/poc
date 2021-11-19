package buildpackconfig

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-buildpacks/poc/kaniko/util"
	"github.com/sirupsen/logrus"
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
)

type BuildPackConfig struct {
	LayerPath       string
	TarPath         string
	CacheDir        string
	KanikoDir       string
	WorkspaceDir    string
	DockerFileName  string
	Opts      		config.KanikoOptions
	NewImage   		v1.Image
	BuildArgs  		[]string
	CnbEnvVars 		map[string]string
}

func NewBuildPackConfig() *BuildPackConfig {
	return &BuildPackConfig{
		LayerPath: "",
		TarPath: "",
		CacheDir: cacheDir,
		WorkspaceDir: workspaceDir,
		KanikoDir: kanikoDir,
	}
}

func (b *BuildPackConfig) InitDefaults() {

	logrus.Debug("Check if DOCKER_FILE_NAME env is defined...")
	b.DockerFileName = util.GetValFromEnVar(DOCKER_FILE_NAME_ENV_NAME)
	if b.DockerFileName == "" {
		b.DockerFileName = defaultDockerFileName
	}
	logrus.Debugf("DockerfileName is: %s", b.DockerFileName)

	logrus.Debug("Checking if CNB_* env var have been declared ...")
	b.CnbEnvVars = util.GetCNBEnvVar()
	logrus.Debugf("CNB ENV var is: %s", b.CnbEnvVars)

	// Convert the CNB ENV vars to Kaniko BuildArgs
	for k, v := range b.CnbEnvVars {
		logrus.Debugf("CNB env key: %s, value: %s", k, v)
		arg := k + "=" + v
		b.BuildArgs = append(b.BuildArgs, arg)
	}

	// setup the path to access the Dockerfile within the workspace dir
	dockerFilePath := b.WorkspaceDir + "/" + b.DockerFileName

	// init the Kaniko options
	b.Opts = config.KanikoOptions{
		CacheOptions:   config.CacheOptions{CacheDir: cacheDir},
		DockerfilePath: dockerFilePath,
		IgnoreVarRun:   true,
		NoPush:         true,
		SrcContext:     b.WorkspaceDir,
		SnapshotMode:   "full",
		BuildArgs:      b.BuildArgs,
	}

	logrus.Debug("KanikoOptions defined")
}

func (b *BuildPackConfig) BuildDockerFile() (err error) {

	logrus.Debugf("Moving to kaniko home dir: %s", b.KanikoDir)
	if err := os.Chdir(b.KanikoDir); err != nil {
		panic(err)
	}

	logrus.Debugf("Building the %s ...", b.DockerFileName)
	b.NewImage, err = executor.DoBuild(&b.Opts)
	return err
}

func (b *BuildPackConfig) CopyLayersTarFileToCacheDir(image v1.Image) {
	// Get layers
	layers, err := b.NewImage.Layers()
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
		b.LayerPath = filepath.Join(b.KanikoDir, digest.String()+".tgz")
		logrus.Infof("Tar layer file: %s\n", b.LayerPath)
		err = saveLayer(layer, b.LayerPath)
		if err != nil {
			panic(err)
		}
	}

	// Copy the content of the kanikoDir to the cacheDir
	util.Dir(b.KanikoDir, b.CacheDir)
}

func (b *BuildPackConfig) SaveImageRawManifest() {
	rawManifest, err := b.NewImage.RawManifest()
	rawManifestFilePath := b.CacheDir + "/manifest.json"
	err = ioutil.WriteFile(rawManifestFilePath, rawManifest, 0644)
	if err != nil {
		panic(err)
	}
	logrus.Debugf("Manifest file of the new image stored at %s",rawManifestFilePath)
}

func (b *BuildPackConfig) SaveImageJSONConfig() {
	// Get the Image config file
	configJSON, err := b.NewImage.ConfigFile()
	if err != nil {
		panic(err)
	}
	configPath := filepath.Join(b.KanikoDir, "config.json")
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

func (b *BuildPackConfig) untarFile(tgzFile string) (err error) {
	// UnGzip first the tgz file
	gzf, err := unGzip(tgzFile, b.KanikoDir)
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

