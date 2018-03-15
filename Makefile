NAME = multy-golos

BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
COMMIT = $(shell git rev-parse --short HEAD)
BUILDTIME = $(shell date +%Y-%m-%dT%T%z)

LD_OPTS = -ldflags="-X main.branch=${BRANCH} -X main.commit=${COMMIT} -X main.buildtime=${BUILDTIME} -w"

all:  build run

all-with-deps: setup deps build

run:
	./$(NAME)

setup:
	go get -u github.com/kardianos/govendor

deps:
	govendor sync

build:
	go build $(LD_OPTS) -o $(NAME) .

# Show to-do items per file.
todo:
	@grep \
		--exclude-dir=vendor \
		--exclude=Makefile \
		--text \
		--color \
		-nRo -E ' TODO:.*|SkipNow|nolint:.*' .
.PHONY: todo

dist:
	GOOS=linux GOARCH=amd64 go build $(LD_OPTS)  -o $(NAME) .

test:
	GOOS=linux GOARCH=amd64 go build $(LD_OPTS)  -o test .
