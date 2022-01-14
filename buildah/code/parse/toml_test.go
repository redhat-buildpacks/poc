package parse_test

import (
	"github.com/BurntSushi/toml"
	"github.com/redhat-buildpacks/poc/buildah/model"
	"reflect"
	"testing"
)


func TestDecodeMetadataFile(t *testing.T) {	var tomlMetadata = `
[[buildpacks]]
api = "0.7"
homepage = "https://github.com/buildpacks/samples/tree/main/extensions/curl"
id = "samples/curl"
version = "0.0.1"
extension = true

[[buildpacks]]
api = "0.7"
homepage = "https://github.com/buildpacks/samples/tree/main/extensions/rebasable"
id = "samples/rebasable"
version = "0.0.1"
extension = true

[[dockerfiles]]
extension_id = "samples/curl"
path = "/layers/samples_curl/Dockerfile"
build = true
run = true

[[dockerfiles.args]]
name = "some_arg"
value = "some-arg-build-value"

[[dockerfiles.args]]
name = "some_arg"
value = "some-arg-launch-value"

[[dockerfiles]]
extension_id = "samples/rebasable"
path = "/cnb/ext/samples_rebasable/0.0.1/Dockerfile"
build = true
run = true
`

	expected := model.Metadata{[]model.Dockerfile{
		{Path: "/layers/samples_curl/Dockerfile",Build: true,Run: true,ExtensionID: "samples/curl",Args: []model.DockerfileArg{
			{Key: "some_arg", Value: "some-arg-build-value"},
			{Key: "some_arg", Value: "some-arg-launch-value"},
		}},
		{Path: "/cnb/ext/samples_rebasable/0.0.1/Dockerfile",Build: true,Run: true,ExtensionID: "samples/rebasable",Args: nil},
	}}
	var got model.Metadata
	if _, err := toml.Decode(tomlMetadata, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("\n%#v\n!=\n%#v\n", expected, got)
	}
}