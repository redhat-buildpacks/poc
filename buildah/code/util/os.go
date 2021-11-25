package util

import (
	"github.com/sirupsen/logrus"
	"os"
)

func GetPWD() string {
	dir, err := os.Getwd()
	if err != nil {
		logrus.Fatal("Get current dir failed !",err)
	} else {
		return dir
	}
	return ""
}
