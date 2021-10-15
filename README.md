# Build an image using Buildah lib

# Centos

Install the needed dependencies: https://github.com/containers/buildah/blob/main/install.md#rhel-centos

# MacOS

## Prerequisite

The following package is needed otherwise the compilation of the application will fail

```bash
# github.com/mtrmac/gpgme
../../golang/pkg/mod/github.com/mtrmac/gpgme@v0.1.2/data.go:4:11: fatal error: 'gpgme.h' file not found
 #include <gpgme.h>
          ^~~~~~~~~
```

It can be installed using brew
```bash
brew install gpgme
```