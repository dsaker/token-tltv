//go:build go1.23

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=internal/oapi/cfg.yaml internal/oapi/tltv.yaml
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/translates.go -source=internal/services/translates/translates.go
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/googleclients.go -source=internal/services/translates/googleclients.go
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/audiofile.go -source=internal/services/audiofile/audiofile.go
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/interfaces.go -source=internal/interfaces/interfaces.go

package main
