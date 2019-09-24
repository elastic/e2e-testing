# Building the binary
We use Gobuffalo's Packr to embbed static resources into the binary, so for that reason we have replaced the default Go build image with one with `packr2` already installed: https://github.com/drud/golang-build-container.

The `build.sh` script builds the binary based on a few environment variables specifying the target distribution:

```shell 
$ GOOS=("darwin" "linux" "windows") ./build.sh
# and/or
$ GOARCH=("386" "amd64") ./build.sh
```