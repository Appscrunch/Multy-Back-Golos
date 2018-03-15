FROM golang:1.9.4

ENV REPO "github.com/Appscrunch/Multy-Back-Golos"

COPY ./ "$GOPATH/src/$REPO"

RUN cd $GOPATH/src/$REPO && \
    make all-with-deps

RUN ls $GOPATH/src/

WORKDIR /go/src/github.com/Appscrunch/Multy-back/cmd

EXPOSE 8080

ENTRYPOINT $GOPATH/src/$REPO/multy-golos
