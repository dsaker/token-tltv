package models

import (
	"context"
	firebase "firebase.google.com/go"
	"flag"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"talkliketv.click/tltv/internal/util"
	"testing"
)

func TestTokenGenerate(t *testing.T) {
	if util.Test != "integration" {
		t.Skip("skipping integration test")
	}

	t.Run("generate_tokens_test", func(t *testing.T) {
		token, plaintext, err := GenerateToken()
		require.NoError(t, err)

		collName := strings.Split(t.Name(), "/")[0]

		// Use the application default credentials
		ctx := context.Background()
		conf := &firebase.Config{ProjectID: "token-tltv-test"}
		app, err := firebase.NewApp(ctx, conf)
		require.NoError(t, err)

		client, err := app.Firestore(ctx)
		require.NoError(t, err)
		defer client.Close()

		// get the tokens collection from the database
		_, err = client.Collection(collName).Doc(token.Hash).Set(ctx, FirestoreToken{
			UploadUsed: token.UploadUsed,
			TimesUsed:  token.TimesUsed,
			Created:    token.Created,
		})
		require.NoError(t, err)

		coll := client.Collection(collName)
		tokens := Tokens{Coll: coll}
		err = tokens.CheckToken(ctx, plaintext)
		require.NoError(t, err)

		err = util.DeleteFirestoreCollection(ctx, client, coll)
		require.NoError(t, err)
		require.NoError(t, err)
	})
}

func TestMain(m *testing.M) {
	var platform string
	flag.StringVar(&platform, "platform", "google", "which platform you are using [google|amazon]")
	flag.StringVar(&util.Test, "test", "test", "type of tests to run [unit|integration|end-to-end]")
	var projectId string
	flag.StringVar(&projectId, "project-id", "", "project id for google cloud platform that contains firestore")
	flag.Parse()
	os.Exit(m.Run())
}
