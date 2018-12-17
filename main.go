package main

import (
	"bytes"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const dockerfile = `
# Build the binary in docker container
FROM golang:1.10.3 AS build
WORKDIR /go/src/{{ .PackagePath  }}
{{ if .Dep }}
RUN go get github.com/golang/dep/cmd/dep
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure -v -vendor-only
{{ else if .GoMod }}
RUN go mod vendor
{{ else  }}
RUN go get -v ./...
{{ end}}

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/{{.BinaryName}} -ldflags="-w -s" -v {{.MainPackage}}

FROM alpine:3.8 AS final
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/{{.BinaryName}} /bin/{{ .BinaryName }}
`

type Args struct {
	// Root directory name / in new words the module
	PackagePath string
	Dep         bool
	GoMod       bool
	BinaryName  string
	MainPackage string
}

func (a Args) CreateDockerFile() {
	tmpl, err := template.New("Dockerfile").Parse(dockerfile)
	if err != nil {
		log.Fatal("Parsing template failed %v", err)
	}
	var b bytes.Buffer
	err = tmpl.Execute(&b, a)
	if err != nil {
		log.Fatal("Binding values to template failed", err)
	}

	ioutil.WriteFile("Dockerfile", b.Bytes(), 0655)
}

func notgomod() (packagepath, defaultbinaryname string) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	trim := strings.Split(wd, "src")
	packagepath = strings.TrimPrefix(trim[len(trim)-1], "/")
	_, defaultbinaryname = filepath.Split(packagepath)
	return

}

func gomod() string {
	data, err := ioutil.ReadFile("go.mod")
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
	// no implemented
}

func main() {
	var packagepath, defaultbinaryname string

	// Check if go.mod exists
	_, err := os.Stat("go.mod")
	switch os.IsNotExist(err) {
	case true:
		packagepath, defaultbinaryname = notgomod()
	default:
		gomod()
	}

	var binaryname string
	flag.StringVar(&binaryname, "binaryname", defaultbinaryname, "name of the binary which is created")
	flag.Parse()

	args := Args{
		BinaryName:  binaryname,
		Dep:         true,
		MainPackage: packagepath,
		PackagePath: packagepath,
	}

	if _, err := os.Stat("Gopkg.toml"); os.IsNotExist(err) {
		args.Dep = false
	}

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		args.GoMod = false
	}

	args.CreateDockerFile()
}
