FROM golang:1.17-alpine AS build

WORKDIR /cronpose

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" .

FROM scratch

WORKDIR /
COPY --from=build /cronpose/cronpose /cronpose

CMD [ "/cronpose" ]
