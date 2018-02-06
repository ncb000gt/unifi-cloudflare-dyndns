deps:
	go get github.com/kardianos/govendor
	govendor fetch github.com/spf13/viper

build: deps
	go build
