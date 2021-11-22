package main

import (
	"fmt"
	cfg "github.com/redhat-buildpacks/poc/kaniko/buildpackconfig"
	"github.com/redhat-buildpacks/poc/kaniko/logging"
	util "github.com/redhat-buildpacks/poc/kaniko/util"
	logrus "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"syscall"
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
)

var (
	logLevel      string   // Log level (trace, debug, info, warn, error, fatal, panic)
	logFormat     string   // Log format (text, color, json)
	logTimestamp  bool     // Timestamp in log output
	extractLayers bool     // Extract layers from tgz files. Defaul is false
	filesToSearch []string // List of files to search to check if they exist under the updated FS
)

func init() {
	envVal := util.GetValFromEnVar(FILES_TO_SEARCH_ENV_NAME)
	if envVal != "" {
		filesToSearch = strings.Split(envVal, ",")
	}

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
			logrus.Info("The layered tzg files will be extracted to the home dir ...")
		}
	}

	if err := logging.Configure(logLevel, logFormat, logTimestamp); err != nil {
		panic(err)
	}
}

func main() {
	if _, ok := os.LookupEnv("DEBUG"); ok && (len(os.Args) <= 1 || os.Args[1] != "from-debugger") {
		args := []string {
			"--listen=:2345",
			"--headless=true",
			"--api-version=2",
			"--accept-multiclient",
			"exec",
			"/kaniko-app", "from-debugger",
		}
		err := syscall.Exec("/usr/local/bin/dlv", append([]string{"/usr/local/bin/dlv"}, args...), os.Environ())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
	logrus.Info("Starting the Kaniko application to process a Dockerfile ...")

	// Create a buildPackConfig and set the default values
	logrus.Info("Initialize the BuildPackConfig and set the defaults values ...")
	b := cfg.NewBuildPackConfig()
	b.InitDefaults()
	b.ExtractLayers = extractLayers

	logrus.Infof("Kaniko      dir: %s", b.KanikoDir)
	logrus.Infof("Workspace   dir: %s", b.WorkspaceDir)
	logrus.Infof("Cache       dir: %s", b.CacheDir)
	logrus.Infof("Dockerfile name: %s", b.DockerFileName)
	logrus.Infof("Extract layer files ? %v", extractLayers)

	// Build the Dockerfile
	logrus.Infof("Building the %s", b.DockerFileName)
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
	logrus.Infof("Export the layers as tar gzip files under the %s ...", b.CacheDir)
	b.ExtractLayersFromNewImageToKanikoDir()

	// Copy the files created from the Kaniko dir to the Cache dir
	logrus.Infof("Copy the files created from the Kaniko dir to the %s dir ...", b.CacheDir)
	b.CopyTGZFilesToCacheDir()

	// Find the digest/hash of the Base Image
	baseImageHash := b.FindBaseImageDigest()

	// Explode the layers created under the container / filesystem
	logrus.Info("Extract the content of the tgz file the / filesystem ...")
	b.ExtractTGZFile(baseImageHash)

	// Check if files exist
	if (len(filesToSearch) > 0) {
		util.FindFiles([]string{"hello.txt", "curl"})
	}
}
