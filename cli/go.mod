module github.com/elastic/metricbeat-tests-poc/cli

go 1.12

require (
	github.com/Flaque/filet v0.0.0-20190209224823-fc4d33cfcf93
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/Microsoft/hcsshim v0.8.6 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190506211059-b20a14b54661
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/rogpeppe/go-internal v1.5.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/testcontainers/testcontainers-go v0.0.8
	golang.org/x/crypto v0.0.0-20191029031824-8986dd9e96cf // indirect
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271 // indirect
	golang.org/x/sys v0.0.0-20191029155521-f43be2a4598c // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191028173616-919d9bdd9fe6 // indirect
	google.golang.org/grpc v1.24.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.4 // indirect
)

replace github.com/testcontainers/testcontainers-go v0.0.8 => github.com/mdelapenya/testcontainers-go v0.0.9-compose-1
