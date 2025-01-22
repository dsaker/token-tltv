FROM golang:1.23 AS deps

ARG LINKER_FLAGS=$LINKER_FLAGS

WORKDIR /talkliketv
ADD *.mod *.sum ./
RUN go mod download

FROM deps AS dev
ADD . .
EXPOSE 4000
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LINKER_FLAGS" -o=./tltv ./api

CMD ["/talkliketv/tltv"]

FROM scratch AS prod

WORKDIR /
EXPOSE 4000
COPY --from=dev /talkliketv/tltv /
CMD ["/tltv"]