FROM golang:1.18.2-alpine3.16 AS build

ENV NOMS_SRC=$GOPATH/src/github.com/ndau/noms
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV DOCKER=1

RUN mkdir -pv $NOMS_SRC
COPY . ${NOMS_SRC}
RUN go test github.com/ndau/noms/...
RUN go install -v github.com/ndau/noms/cmd/noms
RUN cp $GOPATH/bin/noms /bin/noms

# We're going to use our own noms.....
COPY . .

RUN go build ./cmd/noms
RUN mv noms /bin

FROM alpine:3.16.0

COPY --from=build /bin/noms /bin/noms

VOLUME /data
EXPOSE 8000

ENTRYPOINT [ "noms" ]

CMD ["serve", "/data"]
