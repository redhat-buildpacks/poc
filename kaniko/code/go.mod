module github.com/redhat-buildpacks/poc/kaniko

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/GoogleContainerTools/kaniko v1.7.1-0.20220114205832-76624697df87
	github.com/google/go-containerregistry v0.4.1-0.20210128200529-19c2b639fab1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
)

replace (
    // These match the docker/docker's dependencies configured in:
    // https://github.com/moby/moby/blob/v20.10.12/vendor.conf
	github.com/moby/buildkit v0.9.3 => github.com/moby/buildkit v0.8.3
	github.com/opencontainers/runc v1.0.3 => github.com/opencontainers/runc v1.0.0-rc92
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/tonistiigi/fsutil v0.0.0-20190819224149-3d2716dd0a4d => github.com/tonistiigi/fsutil v0.0.0-20191018213012-0f039a052ca1
)

go 1.16
