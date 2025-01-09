//go:build go1.23

//go:generate go run go.uber.org/mock/mockgen -package mockt -destination=internal/mock/translates/translates.go -source=internal/translates/translates.go
//go:generate go run go.uber.org/mock/mockgen -package mockt -destination=internal/mock/translates/amazonclients.go -source=internal/translates/amazonclients.go
//go:generate go run go.uber.org/mock/mockgen -package mockt -destination=internal/mock/translates/googleclients.go -source=internal/translates/googleclients.go
//go:generate go run go.uber.org/mock/mockgen -package mocka -destination=internal/mock/audiofile/audiofile.go -source=internal/audio/audiofile/audiofile.go
//go:generate go run go.uber.org/mock/mockgen -package mockm -destination=internal/mock/models/models.go -source=internal/models/models.go

package main
