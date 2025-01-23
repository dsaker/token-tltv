FROM golang:latest AS build

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /tltv

FROM alpine:latest
RUN apk update
RUN apk upgrade
RUN apk add --no-cache ffmpeg

COPY --from=build /tltv /
EXPOSE 8080

CMD ["/tltv"]