package util

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

func GetCNBEnvVar() (map[string]string) {
	kvs := map[string]string{}
	envs := os.Environ()

	for _, env := range envs {
		if strings.Contains(env,"CNB") {
			str := strings.Split(env,"=")
			kvs[str[0]] = str[1]
		}
	}
	return kvs
}

func GetValFromEnVar(envVar string) (val string) {
	val, ok := os.LookupEnv(envVar)
	if !ok {
		logrus.Debugf("%s not set", envVar)
		return ""
	} else {
		logrus.Debugf("%s=%s", envVar, val)
		return val
	}
}

func ReadFileContent(f *os.File) {
	data, err := ioutil.ReadFile(f.Name())
	if err != nil {
		logrus.Errorf("Failed reading data from file: %s", err)
	}
	logrus.Debugf("\nFile Name: %s", f.Name())
	logrus.Debugf("\nData: %s", data)
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

func ReadFilesFromPath(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Println(file.Name(), file.IsDir())
	}
	return nil
}
