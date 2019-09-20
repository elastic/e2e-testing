module github.com/elastic/metricbeat-tests-poc/cli

go 1.12

require (
	github.com/Flaque/filet v0.0.0-20190209224823-fc4d33cfcf93
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190506211059-b20a14b54661
	github.com/docker/go-units v0.4.0 // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.1.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	github.com/testcontainers/testcontainers-go v0.0.8
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
	golang.org/x/sys v0.0.0-20190712062909-fae7ac547cb7 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/genproto v0.0.0-20190716160619-c506a9f90610 // indirect
	google.golang.org/grpc v1.22.0 // indirect
)

replace github.com/spf13/viper v1.3.2 => github.com/mdelapenya/viper v1.4.1

replace github.com/testcontainers/testcontainers-go v0.0.8 => github.com/mdelapenya/testcontainers-go v0.0.9-compose
