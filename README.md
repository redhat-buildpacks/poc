# Poc development

This project has been designed in order to validate if we can build an image using
`buildah go lib` and `Dockerfiles`.

As it is needed to use a Linux environment to test the go executable, we will use Vagrant as tool
to launch a Linux VM locally which contains the needed tools (github, podman, buildah, ...) , go framework, ...

Open a terminal where you will be able to bump the VM using the command `vagrant up` and `vagrant ssh`.
Next, you can build the project and launch it there

```bash
cd poc
go build -o out/bud ./bud

sudo ./out/bud
WARN[0000] Failed to decode the keys ["storage.options.override_kernel_check"] from "/etc/containers/storage.conf". 
INFO[0000] Buildah tempdir :                            
INFO[0000] Dockerfile name: %!(EXTRA string=/home/vagrant/poc/Dockerfile, string=/home/vagrant/poc/Dockerfile) 
INFO[0003] Image id: %!(EXTRA string=169271538094c8e40eab0fc6bb106d8b0c4c63739641e4289a1762dadfa35ec8) 
```
The image created is available under the temp buildah folder creates:
```bash
sudo ls -la /root/buildah-poc-2171025412/root/overlay-images/
total 16
drwx------. 4 root root  188 Oct 27 11:45 .
drwx------. 8 root root  155 Oct 27 11:45 ..
drwx------. 2 root root 4096 Oct 27 11:45 169271538094c8e40eab0fc6bb106d8b0c4c63739641e4289a1762dadfa35ec8
drwx------. 2 root root 4096 Oct 27 11:45 cf2a2d19642401ea6af3a51cfc5f5190fca39734409fb2f7f4f4c5173da9d70e
-rw-------. 1 root root 3558 Oct 27 11:45 images.json
-rw-r--r--. 1 root root   64 Oct 27 11:45 images.lock
```
**NOTE**: From your local machine, sync the files with the VM using the command `vagrant rsync`

## MacOS

It is not possible for the moment to develop on a Mac as it is not a real Linux platform !

### Prerequisite

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