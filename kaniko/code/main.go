package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	cfg "github.com/redhat-buildpacks/poc/kaniko/buildpackconfig"
	"github.com/redhat-buildpacks/poc/kaniko/logging"
	"github.com/redhat-buildpacks/poc/kaniko/model"
	util "github.com/redhat-buildpacks/poc/kaniko/util"
	logrus "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	logLevel                string   // Log level (trace, debug, info, warn, error, fatal, panic)
	logFormat               string   // Log format (text, color, json)
	logTimestamp            bool     // Timestamp in log output
	extractLayers           bool     // Extract layers from tgz files. Default is false
	filesToSearch           []string // List of files to search to check if they exist under the updated FS
	b						*cfg.BuildPackConfig
	opts					*globalOptions
)

type globalOptions struct {
	metadata                model.Metadata // Metadata file containing the data populated by the Lifecycle builder
	metadatafileNameToParse string         // METADATA.toml file containing the Dockerfiles and args
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

	if err := logging.Configure(logLevel, logFormat, logTimestamp); err != nil {
		panic(err)
	}
}

func main() {
	logrus.Info("Starting Kaniko application to process Dockerfile(s) ...")

	// Create a buildPackConfig and set the default values
	logrus.Info("Initialize the BuildPackConfig and set the defaults values ...")
	b := cfg.NewBuildPackConfig()
	b.InitDefaults()

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
	b.ExtractLayers = extractLayers

	envVal := util.GetValFromEnVar(FILES_TO_SEARCH_ENV_NAME)
	if envVal != "" {
		filesToSearch = strings.Split(envVal, ",")
	}
	b.FilesToSearch = filesToSearch

	// TODO: To be reviewed in order to better manage that section
	opts := initGlobalOptions()
	opts.metadatafileNameToParse = os.Getenv("METADATA_FILE_NAME")

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

	logrus.Infof("Kaniko      dir: %s", b.KanikoDir)
	logrus.Infof("Workspace   dir: %s", b.WorkspaceDir)
	logrus.Infof("Cache       dir: %s", b.CacheDir)
	logrus.Infof("Dockerfile name: %s", b.DockerFileName)
	logrus.Infof("Extract layer files ? %v", extractLayers)
	logrus.Infof("Metadata toml file: %s", opts.metadatafileNameToParse)

	err := reapChildProcesses()
	if err != nil {
		panic(err)
	}

	if opts.metadatafileNameToParse != "" {
		logrus.Infof("Parsing the Metadata toml file to decode it ...")
		if _, err := toml.DecodeFile(filepath.Join(b.WorkspaceDir, "layers", opts.metadatafileNameToParse), &opts.metadata); err != nil {
			logrus.Infof("METADATA toml path: %s",filepath.Join(b.WorkspaceDir, "layers", opts.metadatafileNameToParse))
			logrus.Fatal(err)
		}
		for _, dockerFile := range opts.metadata.Dockerfiles {
			pathToDockerFile := filepath.Join(b.WorkspaceDir, dockerFile.Path)
			logrus.Infof("Dockerfile path: %s", pathToDockerFile)

			// Set up the Build args to be used by Kaniko
			for _, buildArg := range dockerFile.Args.BuildArg {
				arg := buildArg.Key + "=" + buildArg.Value
				logrus.Infof("Build arg: %s",arg)
				b.Opts.BuildArgs = append(b.Opts.BuildArgs, arg)
			}

			// Process now the Dockerfile
			b.ProcessDockerfile(pathToDockerFile)
		}
	} else {
		// When no metadata.toml file is used, parse the dockerfile directly
		pathToDockerFile := filepath.Join(b.WorkspaceDir, b.DockerFileName)
		logrus.Infof("Dockerfile path: %s", pathToDockerFile)

		// Process now the Dockerfile
		b.ProcessDockerfile(pathToDockerFile)
	}
}

func initGlobalOptions() *globalOptions {
	return &globalOptions{}
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