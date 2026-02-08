FROM golang:1.25-trixie AS build

COPY . /anything
WORKDIR /anything
RUN go build -o ./anything ./cmd/anythingsrv

FROM debian:trixie-slim AS final

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update
RUN apt-get upgrade -qq -y
RUN apt-get install -qq -y ca-certificates

COPY --from=build /anything/anything /anything
CMD ["/anything"]
