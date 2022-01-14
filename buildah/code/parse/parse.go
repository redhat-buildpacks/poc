package parse

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
)

func JsonIndent(desc string, in []byte) {
	var buf bytes.Buffer
	err := json.Indent(&buf, in, "", "    ")
	if err == nil {
		logrus.Infof("%s: %s\n", desc, &buf)
	} else {
		logrus.Fatalf("Cannot indent the JSON raw content ! %s",err)
	}
}

func JsonMarshal(desc string, c interface{}) {
	out, err := json.MarshalIndent(c, "", "    ")
	if err == nil {
		logrus.Infof("%s: %s\n", desc, string(out))
	} else {
		logrus.Fatal("Cannot marshal !",err)
	}
}

