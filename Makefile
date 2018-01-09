DEP_EXECUTABLE := dep
GO_EXECUTABLE := go
VERSION := $(shell git describe --abbrev=10 --dirty --always --tags)
DIST_DIRS := find * -type d -exec
VERSION_PACKAGE := github.com/fredericlemoine/booster-web/cmd.Version
PACKAGE:=github.com/fredericlemoine/booster-web
NAME := booster-web
ASSETFS=(static/bindata_assetfs.go templates/bindata_templates.go)
VPATH=./static/
all: dep build install

dep:
	${DEP_EXECUTABLE} ensure

build: assetfs
	${GO_EXECUTABLE} build -o ${NAME} -ldflags "-X ${VERSION_PACKAGE}=${VERSION}" ${PACKAGE}

install: assetfs
	rm -f ${GOPATH}/bin/${NAME}
	${GO_EXECUTABLE} install -ldflags "-X ${VERSION_PACKAGE}=${VERSION}" ${PACKAGE}

assetfs:
	go get github.com/jteeuwen/go-bindata/...
	go get github.com/elazarl/go-bindata-assetfs/...
	go-bindata-assetfs -pkg static  webapp/static/...
	mv bindata_assetfs.go static
	go-bindata -o bindata_templates.go -pkg templates  webapp/templates/...
	mv bindata_templates.go templates

test:
	${GO_EXECUTABLE} test ${PACKAGE}/...

.PHONY: assetfs deploy deploydir deploywinamd deploywin386 deploylinuxamd deploylinux386 deploydarwinamd deploydarwin386


deploy: dep deploywinamd deploywin386 deploylinuxamd deploylinux386 deploydarwinamd deploydarwin386
	tar -czvf deploy/${VERSION}.tar.gz --directory="deploy" ${VERSION}

deploydir:
	mkdir -p deploy/${VERSION}

deploywinamd: assetfs deploydir
	env GOOS=windows GOARCH=amd64 ${GO_EXECUTABLE} build -o deploy/${VERSION}/${NAME}_amd64.exe -ldflags "-X ${VERSION_PACKAGE}=${VERSION}" ${PACKAGE}

deploywin386: assetfs deploydir
	env GOOS=windows GOARCH=386 ${GO_EXECUTABLE} build -o deploy/${VERSION}/${NAME}_386.exe -ldflags "-X ${VERSION_PACKAGE}=${VERSION}" ${PACKAGE}

deploylinuxamd: assetfs deploydir
	env GOOS=linux GOARCH=amd64 ${GO_EXECUTABLE} build -o deploy/${VERSION}/${NAME}_amd64_linux -ldflags "-X ${VERSION_PACKAGE}=${VERSION}" ${PACKAGE}

deploylinux386: assetfs deploydir
	env GOOS=linux GOARCH=386 ${GO_EXECUTABLE} build -o deploy/${VERSION}/${NAME}_386_linux -ldflags "-X ${VERSION_PACKAGE}=${VERSION}" ${PACKAGE}

deploydarwinamd: assetfs deploydir
	env GOOS=darwin GOARCH=amd64 ${GO_EXECUTABLE} build -o deploy/${VERSION}/${NAME}_amd64_darwin -ldflags "-X ${VERSION_PACKAGE}=${VERSION}" ${PACKAGE}

deploydarwin386: assetfs deploydir
	env GOOS=darwin GOARCH=386 ${GO_EXECUTABLE} build -o deploy/${VERSION}/${NAME}_386_darwin -ldflags "-X ${VERSION_PACKAGE}=${VERSION}" ${PACKAGE}
