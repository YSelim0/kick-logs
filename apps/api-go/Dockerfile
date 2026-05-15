FROM golang:1.26.3-alpine AS build

WORKDIR /src

ARG TARGET=api

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/kick-logs ./cmd/${TARGET}

FROM golang:1.26.3-alpine

WORKDIR /app
COPY --from=build /out/kick-logs /app/kick-logs

EXPOSE 8000

ENTRYPOINT ["/app/kick-logs"]
