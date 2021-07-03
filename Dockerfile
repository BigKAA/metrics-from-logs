FROM golang:1.16.5-alpine3.13  AS build-env
ENV CGO_ENABLED 0
ENV COOS linux
ADD . /go/src/mfl
WORKDIR /go/src/mfl
RUN go build main.go

# Final stage
FROM alpine:3.13
LABEL Author="Artur Kryukov <artur@kryukov.biz>"
EXPOSE 8080
WORKDIR /mfl
ADD templates templates
# ADD .env .
COPY --from=build-env /go/src/mfl/main /mfl
ENTRYPOINT /mfl/main
