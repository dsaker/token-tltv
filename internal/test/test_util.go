package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/mock/gomock"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"talkliketv.click/tltv/internal/mock"
	"talkliketv.click/tltv/internal/models"
	"testing"
	"time"
)

var (
	AudioBasePath  = "/tmp/test/audio/"
	GcpTestProject = "token-tltv-test"
)

func RequireMatchAnyExcept(t *testing.T, model any, response any, skip []string, except string, shouldEqual any) {
	v := reflect.ValueOf(response)
	u := reflect.ValueOf(model)

	for i := 0; i < v.NumField(); i++ {
		// Check if field name is the one that should be different
		if v.Type().Field(i).Name == except {
			// Check if type is int32 or int64
			if v.Field(i).CanInt() {
				// check if equal as int64
				require.Equal(t, shouldEqual, v.Field(i).Int())
			} else {
				// if not check if equal as string
				require.Equal(t, shouldEqual, v.Field(i).String())
			}
		} else if slices.Contains(skip, v.Type().Field(i).Name) {
			continue
		} else {
			if v.Field(i).CanInt() {
				require.Equal(t, u.Field(i).Int(), v.Field(i).Int())
			} else {
				require.Equal(t, u.Field(i).String(), v.Field(i).String())
			}
		}
	}
}

const (
	DefaultPause            = 5
	DefaultPattern          = 1
	MaxLanguages            = 75
	MaxVoices               = 95
	ValidLangId             = 16
	alphabet                = "abcdefghijklmnopqrstuvwxyz"
	FirestoreTestCollection = "token-tltv-test"
)

// RandomString generates a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)] //nolint:gosec
		sb.WriteByte(c)
	}

	return sb.String()
}

func RandomPhrase() models.Phrase {
	return models.Phrase{
		ID:   rand.Intn(100), //nolint:gosec
		Text: RandomString(20),
	}
}

// RandomVoice creates a random Voice for testing
func RandomVoice() models.Voice {
	return models.Voice{
		LangId:                 rand.Intn(MaxLanguages), //nolint:gosec
		LanguageCodes:          []string{RandomString(8), RandomString(8)},
		Gender:                 2,
		VoiceName:              RandomString(8),
		NaturalSampleRateHertz: 24000,
	}
}

func RandomTitle(voices map[int]models.Voice) (title models.Title) {
	return models.Title{
		Name:        RandomString(8),
		TitleLangId: ValidLangId,
		ToVoiceId:   voices[rand.Intn(MaxVoices)].ID, //nolint:gosec
		FromVoiceId: voices[rand.Intn(MaxVoices)].ID, //nolint:gosec
		Pause:       DefaultPause,
		Pattern:     DefaultPattern,
	}
}

type MockStubs struct {
	TranslateX             *mock.MockTranslateX
	GoogleTranslateClientX *mock.MockGoogleTranslateClientX
	GoogleTTsClientX       *mock.MockGoogleTTSClientX
	AmazonTranslateClientX *mock.MockAmazonTranslateClientX
	AmazonTTsClientX       *mock.MockAmazonTTSClientX
	AudioFileX             *mock.MockAudioFileX
	TokensX                *mock.MockTokensX
	ModelsX                *mock.MockModelsX
}

// NewMockStubs creates instantiates new instances of all the mock interfaces for testing
func NewMockStubs(ctrl *gomock.Controller) MockStubs {
	return MockStubs{
		TranslateX:             mock.NewMockTranslateX(ctrl),
		GoogleTranslateClientX: mock.NewMockGoogleTranslateClientX(ctrl),
		GoogleTTsClientX:       mock.NewMockGoogleTTSClientX(ctrl),
		AudioFileX:             mock.NewMockAudioFileX(ctrl),
		AmazonTranslateClientX: mock.NewMockAmazonTranslateClientX(ctrl),
		AmazonTTsClientX:       mock.NewMockAmazonTTSClientX(ctrl),
		TokensX:                mock.NewMockTokensX(ctrl),
		ModelsX:                mock.NewMockModelsX(ctrl),
	}
}

type TltvContainer struct {
	testcontainers.Container
	URI string
}

type StdoutLogConsumer struct{}

// Accept prints the log to stdout
func (lc *StdoutLogConsumer) Accept(l testcontainers.Log) {
	fmt.Print(string(l.Content))
}

func StartContainer(ctx context.Context, projectId, saFile string) (*TltvContainer, error) {
	g := StdoutLogConsumer{}

	absPath, err := filepath.Abs(saFile)
	if err != nil {
		log.Fatal(err)
	}

	r, err := os.Open(absPath)
	if err != nil {
		log.Fatal(err)
	}

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../",
			Dockerfile: "/deploy/docker/dev/Dockerfile",
			KeepImage:  true,
		},
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"TEST_PROJECT_ID":                projectId,
			"GOOGLE_APPLICATION_CREDENTIALS": "/secrets/acp/application_default_credentials.json",
		},
		Files: []testcontainers.ContainerFile{
			{
				Reader:            r,
				HostFilePath:      absPath, // will be discarded internally
				ContainerFilePath: "/secrets/acp/application_default_credentials.json",
				FileMode:          0o400,
			},
		},
		WaitingFor: wait.ForHTTP("/").WithPort("8080/tcp"),
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Opts: []testcontainers.LogProductionOption{
				testcontainers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []testcontainers.LogConsumer{&g},
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	var tltvC *TltvContainer
	if container == nil {
		return nil, errors.New("container is nil")
	}

	tltvC = &TltvContainer{Container: container}
	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "8080")
	if err != nil {
		return nil, err
	}

	tltvC.URI = fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())
	return tltvC, nil
}
