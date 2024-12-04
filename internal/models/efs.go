package models

import (
	"embed"
)

// JsonModels are the JSON returned by the platform api's. They are embedded so they can be
// loaded into maps at the startup of the application instead of using a database
//
//go:embed "jsonmodels"
var JsonModels embed.FS
