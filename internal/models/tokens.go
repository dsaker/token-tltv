package models

import (
	"cloud.google.com/go/firestore"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"talkliketv.com/tltv/internal/interfaces"
)

func (m *Models) CheckToken(ctx context.Context, token string) error {
	tokenHash := sha256.Sum256([]byte(token))
	hashString := hex.EncodeToString(tokenHash[:])
	tokenDoc, err := m.tokenCollection.Doc(hashString).Get(ctx)
	if err != nil {
		return fmt.Errorf("get token check failed: %w", err)
	}
	var tStruct interfaces.Token
	err = tokenDoc.DataTo(&tStruct)
	if err != nil {
		return fmt.Errorf("token data to struct failed: %w", err)
	}
	if tStruct.UploadUsed {
		return ErrUsedToken
	}
	return err
}

func (m *Models) AddToken(ctx context.Context, token interfaces.Token) error {
	_, err := m.tokenCollection.Doc(token.Hash).Set(ctx, interfaces.FirestoreToken{
		UploadUsed: token.UploadUsed,
		TimesUsed:  token.TimesUsed,
		Created:    token.Created,
	})
	if err != nil {
		return fmt.Errorf("failed adding token: %v", err)
	}
	return err
}

func (m *Models) UpdateTokenField(ctx context.Context, value any, token, path string) error {
	tokenHash := sha256.Sum256([]byte(token))
	hashString := hex.EncodeToString(tokenHash[:])
	tokenDoc := m.tokenCollection.Doc(hashString)
	_, err := tokenDoc.Update(ctx, []firestore.Update{
		{
			Path:  path,
			Value: value,
		},
	})
	if err != nil {
		return fmt.Errorf("token update failed: %w", err)
	}
	return err
}

func (m *Models) DeleteToken(ctx context.Context, hash string) error {
	_, err := m.tokenCollection.Doc(hash).Delete(ctx)
	if err != nil {
		return fmt.Errorf("delete token failed: %w", err)
	}
	return err
}
