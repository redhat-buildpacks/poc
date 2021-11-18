package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	logrus "github.com/sirupsen/logrus"
)

const ( // TODO: derive or pass in
	kanikoDir				   = "/kaniko"
	cacheDir				   = "/cache"
	workspaceDir			   = "/workspace"
	outputDir                  = kanikoDir
	ubuntuBionicTopLayerDigest = "sha256:284055322776031bac33723839acb0db2d063a525ba4fa1fd268a831c7553b26"
)

var (
	layerPath   string
	tarPath     string
)


func main() {
	logrus.Info("Build the Dockerfile, populate a tarball...")
	exportTarball()
}

func exportTarball() {
	// create Kaniko config
	opts := &config.KanikoOptions{ // TODO: see which of these options are truly needed
		CacheOptions:   config.CacheOptions{CacheDir: cacheDir},
		DockerfilePath: workspaceDir + "/Dockerfile",
		IgnoreVarRun:   true,
		NoPush:         true,
		SrcContext:     "dir://" + workspaceDir,
		SnapshotMode:   "full",
	}

	if err := os.Chdir(kanikoDir); err != nil {
		panic(err)
	}

	// Do kaniko build
	image, err := executor.DoBuild(opts)
	if err != nil {
		panic(err)
	}

	// Get the Image config file
	configJSON, err := image.ConfigFile()
	if err != nil {
		panic(err)
	}
	configPath := filepath.Join(outputDir, "config.json")
	c, err := os.Create(configPath)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	err = json.NewEncoder(c).Encode(*configJSON)

	// Print the image json config
	readFileContent(c)

	// Get layers
	layers, err := image.Layers()
	if err != nil {
		panic(err)
	}
	logrus.Infof("Generated %d layers\n", len(layers))
	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			panic(err)
		}
		digestStr := digest.String()
		if digestStr == ubuntuBionicTopLayerDigest {
			continue
		}
		layerPath = filepath.Join(outputDir, digestStr+".tgz")
		logrus.Infof("Tar layer file: %s\n",layerPath)
		err = saveLayer(layer, layerPath)
		if err != nil {
			panic(err)
		}
	}

	logrus.Infof("Reading dir content of: %s",kanikoDir)
	readFilesFromPath(kanikoDir)

	// Copy /kaniko content to the cache
	Dir(kanikoDir,cacheDir)

	// Read the content of the tgz file
	//logrus.Infof("Read the layer tgz file generated: %s",layerPath)
	//err = untarFile(layerPath)
	//if err != nil {
	//	logrus.Panicf("Reading the tgz file failed: %s", err)
	//}
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

func readFileContent(f *os.File) {
	data, err := ioutil.ReadFile(f.Name())
	if err != nil {
		logrus.Errorf("Failed reading data from file: %s", err)
	}
	logrus.Infof("\nFile Name: %s", f.Name())
	logrus.Infof("\nData: %s", data)
}

func untarFile(tgzFile string) (err error) {
	// UnGzip first the tgz file
	gzf, err := unGzip(tgzFile, kanikoDir)
	if err != nil {
		logrus.Panicf("Something wrong happened ... %s",err)
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
		logrus.Infof("File extracted: %s",hdr.Name)
	}
	return nil
}

func unGzip(gzipFile, tarPath string) (gzf io.Reader, err error) {
	logrus.Infof("Opening the gzip file: %s",gzipFile)
	f, err := os.Open(gzipFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	logrus.Infof("Creating a gzip reader for: %s",f.Name())
	gzf, err = gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	return gzf, nil
}

func readFilesFromPath(path string) (error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Println(file.Name(), file.IsDir())
	}
	return nil
}

func Dir(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = Dir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = File(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}

// File copies a single file from src to dst
func File(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}