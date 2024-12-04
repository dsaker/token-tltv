package audio

import (
	"embed"
)

// Silence is stored as an embedded FS so the silence mp3 tracks can be added
// to the machines file system on startup of the application with server.go initSilence()
//
//go:embed "silence"
var Silence embed.FS
