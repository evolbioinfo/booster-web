GO_EXECUTABLE := go
VERSION := $(shell git describe --abbrev=10 --dirty --always --tags)
DIST_DIRS := find * -type d -exec

all: build install

build: 
	go-bindata-assetfs -pkg static  webapp/static/...
	go-bindata -o bindata_templates.go -pkg templates  webapp/templates/...

	mv bindata_assetfs.go static
	mv bindata_templates.go templates

	${GO_EXECUTABLE} build -o sbsweb -ldflags "-X github.com/fredericlemoine/sbsweb/cmd.Version=${VERSION}" github.com/fredericlemoine/sbsweb

install:
	${GO_EXECUTABLE} install -ldflags "-X github.com/fredericlemoine/sbsweb/cmd.Version=${VERSION}" github.com/fredericlemoine/sbsweb
