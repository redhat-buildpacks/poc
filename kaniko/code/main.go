package main

import (
	cfg "github.com/redhat-buildpacks/poc/kaniko/buildpackconfig"
	"github.com/redhat-buildpacks/poc/kaniko/logging"
	util "github.com/redhat-buildpacks/poc/kaniko/util"
	logrus "github.com/sirupsen/logrus"
)

const (
	LOGGING_LEVEL_ENV_NAME       = "LOGGING_LEVEL"
	LOGGING_FORMAT_ENV_NAME      = "LOGGING_FORMAT"

	DefaultLevel              = "info"
	DefaultLogTimestamp       = false
	DefaultLogFormat          = "text"
)

var (
	logLevel	 string // Log level (trace, debug, info, warn, error, fatal, panic)
	logFormat    string // Log format (text, color, json)
	logTimestamp bool   // Timestamp in log output
)

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
	b := cfg.NewBuildPackConfig()
	b.InitDefaults()
	logrus.Infof("Kaniko      dir: %s",b.KanikoDir)
	logrus.Infof("Workspace   dir: %s",b.WorkspaceDir)
	logrus.Infof("Cache       dir: %s",b.CacheDir)
	logrus.Infof("Dockerfile name: %s",b.DockerFileName)

	// Build the Dockerfile
	logrus.Infof("Building the %s",b.DockerFileName)
	err := b.BuildDockerFile()
	if err != nil {
		panic(err)
	}

	// Save the Config and Manifest files of the new image created
	b.SaveImageJSONConfig()
	b.SaveImageRawManifest()

	// Log the content of the Kaniko dir
	logrus.Infof("Reading dir content of: %s", b.KanikoDir)
	util.ReadFilesFromPath(b.KanikoDir)

	// Export the layers from the new Image as tar gzip file under the Kaniko dir
	logrus.Infof("Export the layers as tar gzip files under the %s ...",b.CacheDir)
	b.ExtractLayersFromNewImageToKanikoDir()

	// Copy the files created from the Kaniko dir to the Cache dir
	logrus.Infof("Copy the files created from the Kaniko dir to the %s dir ...",b.CacheDir)
	b.CopyTGZFilesToCacheDir()

	// Find the digest/hash of the Base Image
	baseImageHash := b.FindBaseImageDigest()

	// Explode the layers created under the container / filesystem
	logrus.Info("Extract the content of the tgz file the / filesystem ...")
	b.ExtractTGZFile(baseImageHash)

}