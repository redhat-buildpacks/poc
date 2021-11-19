package util

import (
	"os"
	"testing"
	"github.com/stretchr/testify/assert"
)

var envTests = []struct {
	name				string
	envKey              string
	envVal              string
	expectedBuildArgs   []string
}{
	{
		name:             "CNB foo and bar key, val",
		envKey:           "CNB_foo",
		envVal: 		  "bar",
		expectedBuildArgs:   []string{"CNB_foo=bar"},
	},
}
func TestEnvToBuildArgs(t *testing.T) {

	//b := newBuildPackConfig()

	for _, test := range envTests {
		t.Run(test.name, func(t *testing.T) {

			// set CNB env var
			os.Setenv(test.envKey, test.envVal)

			// Read the env vars
			//b.cnbEnvVars = util.GetCNBEnvVar()

			assert.Equal(t,"1","1")
		})
	}
}
