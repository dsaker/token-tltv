package testflags

import (
	"flag"
	"testing"
)

// TestFlags contains common flags used across test packages
var (
	// TestType stores the type of test being run
	TestType string
	// Platform stores the platform being used
	Platform string
	// ProjectID stores the GCP project ID
	ProjectID string
	// Headless determines if browser tests run headlessly
	Headless bool
	// SAFile stores the service account file path
	SAFile string
	// Local determines if tests run locally
	Local bool
)

// ParseFlags parses common command line flags for tests
func ParseFlags() {
	// Only define flags if they haven't been defined yet
	if flag.Lookup("test") == nil {
		flag.StringVar(&TestType, "test", "unit", "type of tests to run [unit|integration|end-to-end]")
	}
	if flag.Lookup("platform") == nil {
		flag.StringVar(&Platform, "platform", "google", "which platform you are using [google|amazon]")
	}
	if flag.Lookup("project-id") == nil {
		flag.StringVar(&ProjectID, "project-id", "", "project id for google cloud platform that contains firestore")
	}
	if flag.Lookup("headless") == nil {
		flag.BoolVar(&Headless, "headless", true, "if true browser will be headless")
	}
	if flag.Lookup("sa-file") == nil {
		flag.StringVar(&SAFile, "sa-file", "", "path to service account file with permissions to run tests")
	}
	if flag.Lookup("local") == nil {
		flag.BoolVar(&Local, "local", false, "if true end-to-end tests will be run in local mode")
	}
}

// RunTests runs the tests with the parsed flags
func RunTests(m *testing.M) int {
	return m.Run()
}
