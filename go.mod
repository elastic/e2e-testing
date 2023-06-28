module github.com/elastic/e2e-testing

go 1.14

replace (
	github.com/cucumber/godog => github.com/laurazard/godog v0.0.0-20220922095256-4c4b17abdae7

	golang.org/x/oauth2 => golang.org/x/oauth2 v0.0.0-20221014153046-6fdb5e3db783

	// For k8s dependencies, we use a replace directive, to prevent them being
	// upgraded to the version specified in containerd, which is not relevant to the
	// version needed.
	// See https://github.com/docker/buildx/pull/948 for details.
	// https://github.com/docker/buildx/blob/v0.9.1/go.mod#L62-L64
	k8s.io/api => k8s.io/api v0.22.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.4
	k8s.io/client-go => k8s.io/client-go v0.22.4
)

require (
	github.com/AlecAivazis/survey/v2 v2.3.6 // indirect
	github.com/Flaque/filet v0.0.0-20201012163910-45f684403088
	github.com/Jeffail/gabs/v2 v2.6.0
	github.com/cenkalti/backoff/v4 v4.2.0
	github.com/containerd/containerd v1.6.21 // indirect
	github.com/cucumber/godog v0.12.4
	github.com/docker/cli v23.0.5+incompatible
	github.com/docker/docker v23.0.5+incompatible
	github.com/elastic/elastic-package v0.36.0
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20210317102009-a9d74cec0186
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/gobuffalo/packr/v2 v2.8.1
	github.com/google/uuid v1.3.0
	github.com/joho/godotenv v1.4.0
	github.com/klauspost/compress v1.15.12 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/shirou/gopsutil/v3 v3.21.10
	github.com/sirupsen/logrus v1.9.0
	github.com/smartystreets/assertions v1.0.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cobra v1.7.0
	github.com/stretchr/testify v1.8.4
	github.com/testcontainers/testcontainers-go v0.21.0
	go.elastic.co/apm v1.13.0
	go.elastic.co/apm/module/apmelasticsearch v1.10.0
	go.elastic.co/apm/module/apmhttp v1.10.0
	golang.org/x/crypto v0.2.0 // indirect
	golang.org/x/oauth2 v0.0.0-20221014153046-6fdb5e3db783 // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/time v0.1.0 // indirect
	google.golang.org/genproto v0.0.0-20221024183307-1bc688fe9f3e // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/gotestsum v1.10.0
	gotest.tools/v3 v3.4.0 // indirect
	howett.net/plist v0.0.0-20201203080718-1454fab16a06 // indirect
	k8s.io/api v0.25.4 // indirect
	k8s.io/apimachinery v0.25.4 // indirect
	k8s.io/client-go v0.25.4 // indirect
	k8s.io/klog/v2 v2.60.1 // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9 // indirect
)
