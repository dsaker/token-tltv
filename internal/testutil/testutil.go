package testutil

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"log"
	"math/rand"
	"os"
	"reflect"
	"slices"
	"strings"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/mock"
	"testing"
	"time"
)

var (
	AudioBasePath  = "/tmp/test/audio/"
	ParseBasePath  = "/tmp/test/parse/"
	GcpTestProject = "token-tltv-test"
	ErrUnexpected  = errors.New("unexpected error")
)

const (
	FiveSentences = "This is the first sentence.\nThis is the second sentence.\nThis is the third sentence.\nThis is the fourth sentence.\nThis is the fifth sentence.\n"
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

func RandomPhrase() interfaces.Phrase {
	return interfaces.Phrase{
		ID:   rand.Intn(100), //nolint:gosec
		Text: RandomString(20),
	}
}

// RandomVoice creates a random Voice for testing
func RandomVoice() interfaces.Voice {
	return interfaces.Voice{
		Language:               RandomString(8),
		LanguageCode:           RandomString(8),
		SsmlGender:             interfaces.MALE,
		Name:                   RandomString(8),
		NaturalSampleRateHertz: 24000,
	}
}

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

func RandomTitle() (title interfaces.Title) {
	return interfaces.Title{
		Name:      RandomString(8),
		TitleLang: RandomString(8),
		ToVoice:   RandomString(8),
		FromVoice: RandomString(8),
		Pause:     DefaultPause,
		Pattern:   DefaultPattern,
	}
}

type MockStubs struct {
	TranslateX             *mock.MockTranslateX
	GoogleTranslateClientX *mock.MockGoogleTranslateClientX
	GoogleTTsClientX       *mock.MockGoogleTTSClientX
	AudioFileX             *mock.MockAudioFileX
	ModelsX                *mock.MockModelsStore
}

// NewMockStubs creates instantiates new instances of all the mock interfaces for testing
func NewMockStubs(ctrl *gomock.Controller) MockStubs {
	return MockStubs{
		TranslateX:             mock.NewMockTranslateX(ctrl),
		GoogleTranslateClientX: mock.NewMockGoogleTranslateClientX(ctrl),
		GoogleTTsClientX:       mock.NewMockGoogleTTSClientX(ctrl),
		AudioFileX:             mock.NewMockAudioFileX(ctrl),
		ModelsX:                mock.NewMockModelsStore(ctrl),
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
	g := &StdoutLogConsumer{}

	r, err := os.Open("/Users/dustysaker/secrets/token-tltv-test-898847de130d.json")
	if err != nil {
		log.Fatal(err)
	}

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "..",
			Dockerfile: "deploy/docker/dev/Dockerfile",
			KeepImage:  true,
			BuildArgs: map[string]*string{
				"BUILDKIT_INLINE_CACHE": ptr("1"),
			},
			BuildLogWriter: os.Stdout, // Add this to see build logs
		},

		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"PROJECT_ID":                     projectId,
			"GOOGLE_APPLICATION_CREDENTIALS": "/secrets/acp/application_default_credentials.json",
		},
		Files: []testcontainers.ContainerFile{
			{
				Reader:            r,
				HostFilePath:      saFile,
				ContainerFilePath: "/secrets/acp/application_default_credentials.json",
				FileMode:          0o400,
			},
		},
		WaitingFor: wait.ForHTTP("/").WithPort("8080/tcp"),
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Opts: []testcontainers.LogProductionOption{
				testcontainers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []testcontainers.LogConsumer{g},
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

func ptr[T any](v T) *T {
	return &v
}

func DeleteCollection(ctx context.Context, client *firestore.Client, collectionName string, batchSize int) error {
	col := client.Collection(collectionName)
	bulkwriter := client.BulkWriter(ctx)

	for {
		iter := col.Limit(batchSize).Documents(ctx)
		numDeleted := 0

		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return fmt.Errorf("error iterating documents: %v", err)
			}
			_, err = bulkwriter.Delete(doc.Ref)
			if err != nil {
				return err
			}
			numDeleted++
		}

		if numDeleted == 0 {
			break
		}

		bulkwriter.Flush()
	}

	bulkwriter.End()

	fmt.Printf("Deleted collection '%s'\n", collectionName)
	return nil
}
