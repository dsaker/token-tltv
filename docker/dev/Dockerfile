FROM golang:latest AS build

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY ./ .
RUN go mod download

## Copy the source code. Note the slash at the end, as explained in
## https://docs.docker.com/reference/dockerfile/#copy
#COPY ../ .

RUN CGO_ENABLED=0 GOOS=linux go build -o /api

FROM alpine:latest

# install ffmpeg
RUN apk update
RUN apk upgrade
RUN apk add --no-cache ffmpeg

# copy static files to container
COPY ./ui ui/
COPY --from=build ./api /app/api

EXPOSE 8080

CMD ["/app/api", "-maximum-number-phrases=10", "-env=dev"]