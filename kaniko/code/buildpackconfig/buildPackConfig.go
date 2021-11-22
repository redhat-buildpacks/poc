package buildpackconfig

import (
	"archive/tar"
	"encoding/json"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	image_util "github.com/GoogleContainerTools/kaniko/pkg/image"
	fs_util "github.com/GoogleContainerTools/kaniko/pkg/util"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-buildpacks/poc/kaniko/store"
	"github.com/redhat-buildpacks/poc/kaniko/util"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	homeDir                   = "/"
	kanikoDir                 = "/kaniko"
	cacheDir                  = "/cache"
	workspaceDir              = "/workspace"
	defaultDockerFileName     = "Dockerfile"
	DOCKER_FILE_NAME_ENV_NAME = "DOCKER_FILE_NAME"
	IGNORE_PATHS_ENV_NAME     = "IGNORE_PATHS"
)

var ignorePaths = []string{""}

type BuildPackConfig struct {
	LayerPath      string
	CacheDir       string
	KanikoDir      string
	WorkspaceDir   string
	DockerFileName string
	Opts           config.KanikoOptions
	NewImage       v1.Image
	BuildArgs      []string
	CnbEnvVars     map[string]string
	TarPaths       []store.TarFile
	HomeDir        string
	ExtractLayers  bool
	IgnorePaths    []string
}

func NewBuildPackConfig() *BuildPackConfig {
	return &BuildPackConfig{
		LayerPath:    "",
		CacheDir:     cacheDir,
		WorkspaceDir: workspaceDir,
		KanikoDir:    kanikoDir,
		HomeDir:      homeDir,
	}
}

func (b *BuildPackConfig) InitDefaults() {

	logrus.Debug("Check if DOCKER_FILE_NAME env is defined...")
	b.DockerFileName = util.GetValFromEnVar(DOCKER_FILE_NAME_ENV_NAME)
	if b.DockerFileName == "" {
		b.DockerFileName = defaultDockerFileName
	}
	logrus.Debugf("DockerfileName is: %s", b.DockerFileName)

	logrus.Debug("Check if IGNORE_PATHS env is defined...")

	result := util.GetValFromEnVar(IGNORE_PATHS_ENV_NAME)
	if result == "" {
		b.IgnorePaths = ignorePaths
	} else {
		b.IgnorePaths = strings.Split(result, ",")
	}
	logrus.Debugf("Additional paths to be ignored: %s", b.IgnorePaths)
	for _, p := range b.IgnorePaths {
		fs_util.AddToDefaultIgnoreList(fs_util.IgnoreListEntry{
			Path:            p,
			PrefixMatchOnly: false,
		})
	}

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
		IgnorePaths:    b.IgnorePaths,
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

func (b *BuildPackConfig) ExtractLayersFromNewImageToKanikoDir() {
	// Get layers
	layers, err := b.NewImage.Layers()
	if err != nil {
		panic(err)
	}
	logrus.Infof("Generated %d layers", len(layers))

	tarFiles := []store.TarFile{}

	for _, layer := range layers {
		digest, err := layer.Digest()
		digest.MarshalText()
		if err != nil {
			panic(err)
		}

		tarFile := store.TarFile{
			Name: digest.String(),
			Path: filepath.Join(b.KanikoDir, digest.String()+".tgz"),
		}
		logrus.Infof("Tar layer file: %s", tarFile.Path)

		// Add the tgz file to the list of the tgz files
		tarFiles = append(tarFiles, tarFile)

		// Save the tgz layer file within the Kaniko dir
		err = saveLayer(layer, tarFile.Path)
		if err != nil {
			panic(err)
		}
	}
	b.TarPaths = tarFiles
}

func (b *BuildPackConfig) ExtractTGZFile(baseHash v1.Hash) {
	for _, tarFile := range b.TarPaths {
		if tarFile.Name != baseHash.String() {
			logrus.Infof("Tgz file to be extracted %s", tarFile.Name)
			err := b.untarFile(tarFile.Path, b.HomeDir)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (b *BuildPackConfig) CopyTGZFilesToCacheDir() {
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
	logrus.Debugf("Manifest file of the new image stored at %s", rawManifestFilePath)
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
	logrus.Debugf("Image JSON config file stored at %s", configPath)

	// Log the image json config
	// TODO: Add a debug opt to log if needed
	// readFileContent(c)
}

func (b *BuildPackConfig) untarFile(tgzFilePath string, targetDir string) (err error) {

	// create a Reader of the gzip file
	gzf, err := util.UnGzip(tgzFilePath)
	if err != nil {
		logrus.Panicf("Something wrong happened ... %s", err)
	}

	// Open the tar file from the tgz reader
	tr := tar.NewReader(gzf)
	// Get each tar segment
	for true {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Fatalf("ExtractTarGz: Next() failed: %v", err)
		}

		// the target location where the dir/file should be created
		target := filepath.Join(targetDir, hdr.Name)
		logrus.Debugf("File to be extracted: %s", target)

		if b.ExtractLayers {
			switch hdr.Typeflag {
			case tar.TypeDir:
				if _, err := os.Stat(target); err != nil {
					// TODO: Should we define a const for the permission
					if err := os.Mkdir(target, 0755); err != nil {
						logrus.Fatalf("ExtractTarGz: Mkdir() failed: %s", err.Error())
						return err
					}
				}
			case tar.TypeReg:
				pathExists := util.FileExists(target)
				if pathExists {
					logrus.Debugf("ExtractTarGz: %s exists: %t\n", target, pathExists)
				} else {
					outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
					if err != nil {
						logrus.Fatalf("ExtractTarGz: Create() failed: %s", err.Error())
						return err
					}
					if _, err := io.Copy(outFile, tr); err != nil {
						logrus.Fatalf("ExtractTarGz: Copy() failed: %s", err.Error())
						return err
					}
					// manually close here after each file operation; defering would cause each file close
					// to wait until all operations have completed.
					logrus.Debugf("File extracted to %s", outFile.Name())
					outFile.Close()
				}

			default:
				logrus.Debugf(
					"ExtractTarGz: unknown type: %c in %s",
					hdr.Typeflag,
					hdr.Name)
			}
		}

	}
	return nil
}

func (b *BuildPackConfig) FindBaseImageDigest() v1.Hash {
	var digest v1.Hash

	stages, metaArgs, err := dockerfile.ParseStages(&b.Opts)
	if err != nil {
		panic(err)
	}

	kanikoStages, err := dockerfile.MakeKanikoStages(&b.Opts, stages, metaArgs)
	if err != nil {
		panic(err)
	}

	// Check the baseImage and Log the layer digest
	for _, kanikoStage := range kanikoStages {
		var baseImage v1.Image
		logrus.Infof("Kaniko stage is: %s, index: %d", kanikoStage.BaseName, kanikoStage.Index)

		// Retrieve the SourceImage
		baseImage, err = image_util.RetrieveSourceImage(kanikoStage, &b.Opts)

		// Get the layers of the Base Image
		layers, err := baseImage.Layers()
		if err != nil {
			panic(err)
		}
		for _, l := range layers {
			digest, err = l.Digest()
			if err != nil {
				panic(err)
			}
			logrus.Infof("Layer digest of base image is: %s", digest)
		}
	}
	return digest
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
