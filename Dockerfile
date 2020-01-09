FROM golang:alpine
RUN apk update && apk add --no-cache git
WORKDIR $GOPATH/src/mypackage/myapp/
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 go build -o /go/bin/main

FROM scratch
ENV DOCKER_API_VERSION='1.40'
COPY --from=0 /go/bin/main /go/bin/main
ADD ./chrome.json /app/chrome.json
ENTRYPOINT ["/go/bin/main"]