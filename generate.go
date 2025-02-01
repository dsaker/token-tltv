//go:build go1.23

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=internal/oapi/cfg.yaml internal/oapi/tltv.yaml
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/translates.go -source=internal/translates/translates.go
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/amazonclients.go -source=internal/translates/amazonclients.go
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/googleclients.go -source=internal/translates/googleclients.go
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/audiofile.go -source=internal/audio/audiofile/audiofile.go
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/tokens.go -source=internal/models/tokens.go
//go:generate go run go.uber.org/mock/mockgen -package mock -destination=internal/mock/models.go -source=internal/models/models.go

package main
