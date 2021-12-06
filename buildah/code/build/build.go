package build

import (
	"archive/tar"
	"bytes"
	"fmt"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/storage"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/redhat-buildpacks/poc/buildah/util"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"time"
)

type BuildahParameters struct {
	BuildOptions      define.BuildOptions
	StoreOptions      storage.StoreOptions
	TempDir           string
	WorkspaceDir      string
	StorageRootDir    string
	StorageRunRootDir string
	GraphDriverName   string
	ExtractLayers     bool
}

func InitOptions() *BuildahParameters {
	b := &BuildahParameters{}

	b.WorkspaceDir = os.Getenv("WORKSPACE_DIR")
	if b.WorkspaceDir == "" {
		b.WorkspaceDir = "/workspace/buildah-layers"
	}
	logrus.Infof("WORKSPACE DIR: %s", b.WorkspaceDir)

	b.GraphDriverName = os.Getenv("GRAPH_DRIVER")
	if b.GraphDriverName == "" {
		b.GraphDriverName = "vfs"
	}
	logrus.Infof("GRAPH_DRIVER: %s", b.GraphDriverName)

	b.StorageRootDir = os.Getenv("STORAGE_ROOT_PATH")
	if b.StorageRootDir == "" {
		b.StorageRootDir = "/var/lib/containers/storage"
	}
	logrus.Infof("STORAGE ROOT PATH: %s", b.StorageRootDir)

	b.StorageRunRootDir = os.Getenv("STORAGE_RUN_ROOT_PATH")
	if b.StorageRunRootDir == "" {
		b.StorageRunRootDir = "/var/run/containers/storage"
	}
	logrus.Infof("STORAGE RUN ROOT PATH: %s", b.StorageRunRootDir)

	var transientMounts []string

	// Buildah context should be the same as the dir where Dockerfiles, files to be copied are located
	contextDir := b.WorkspaceDir
	logrus.Infof("Buildah contextDir: %s", contextDir)

	dateStamp := fmt.Sprintf("%d", time.Now().UnixNano())
	buildahImage := fmt.Sprintf("buildpack-buildah:%s-%d", dateStamp, 1)

	output := &bytes.Buffer{}

	// Define image build options
	b.BuildOptions = define.BuildOptions{
		ContextDirectory: contextDir,
		CommonBuildOpts:  &define.CommonBuildOptions{},
		NamespaceOptions: []define.NamespaceOption{{
			Name: string(rspec.NetworkNamespace),
			Host: true,
		}},
		TransientMounts:         transientMounts,
		Output:                  buildahImage,
		OutputFormat:            buildah.Dockerv2ImageManifest,
		Out:                     output,
		Err:                     output,
		Layers:                  false, // TODO: Check with containers team what the value should be and this option do
		NoCache:                 true,
		RemoveIntermediateCtrs:  true,
		ForceRmIntermediateCtrs: true,
		Isolation:               buildah.IsolationChroot,
		Squash:                  false,
	}

	// Initialize the storage
	b.StoreOptions = storage.StoreOptions{
		GraphDriverName:     b.GraphDriverName,
		GraphRoot:           b.StorageRootDir,
		RunRoot:             b.StorageRunRootDir,
		RootlessStoragePath: b.StorageRootDir,
	}
	return b
}

func (b *BuildahParameters) ExtractTGZFile(path string) {
	logrus.Infof("Tgz file to be extracted %s",path)
	// TODO: Add a var to define the Root FS dir
	err := b.untarFile(path, "/")
	if err != nil {
		panic(err)
	}
}

func (b *BuildahParameters) untarFile(tgzFilePath string, targetDir string) (err error) {

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
