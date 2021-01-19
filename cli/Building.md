# Building the binary
We use Gobuffalo's Packr to embbed static resources into the binary, so for that reason we are installing `packr2` in the CI worker.

The `build-cli.sh` script builds the binary based on a few environment variables specifying the target distribution:

```shell 
$ GOOS=("darwin" "linux" "windows") ./.ci/scripts/build-cli.sh
# and/or
$ GOARCH=("386" "amd64") ./.ci/scripts/build-cli.sh
```
