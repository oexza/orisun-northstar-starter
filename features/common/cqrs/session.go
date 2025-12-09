package cqrs

import (
	"context"
	"errors"
	"fmt"
	"github.com/delaneyj/toolbelt"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/gorilla/sessions"
	"github.com/nats-io/nats.go/jetstream"
	"net/http"
	"time"
)

type SessionService struct {
	kv    jetstream.KeyValue
	store sessions.Store
}

func NewSessionService(ns *embeddednats.Server, store sessions.Store) (*SessionService, error) {
	nc, err := ns.Client()
	if err != nil {
		return nil, fmt.Errorf("error creating nats client: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("error creating jetstream client: %w", err)
	}

	kv, err := js.CreateOrUpdateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket:      "todos",
		Description: "Datastar Todos",
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating key value: %w", err)
	}

	return &SessionService{
		kv:    kv,
		store: store,
	}, nil
}

func (s *SessionService) GetSessionMVC(w http.ResponseWriter, r *http.Request) (string, []byte, error) {
	ctx := r.Context()
	sessionID, err := s.upsertSessionID(r, w)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get session id: %w", err)
	}

	var mvc []byte
	if entry, err := s.kv.Get(ctx, sessionID); err != nil {
		if !errors.Is(err, jetstream.ErrKeyNotFound) {
			return "", nil, fmt.Errorf("failed to get key value: %w", err)
		}

		if err := s.saveMVC(ctx, sessionID, mvc); err != nil {
			return "", nil, fmt.Errorf("failed to save mvc: %w", err)
		}
	} else {
		mvc = entry.Value()
	}
	return sessionID, mvc, nil
}

func (s *SessionService) SaveMVC(ctx context.Context, sessionID string, mvc []byte) error {
	return s.saveMVC(ctx, sessionID, mvc)
}

func (s *SessionService) WatchUpdates(ctx context.Context, sessionID string) (jetstream.KeyWatcher, error) {
	return s.kv.Watch(ctx, sessionID)
}

func (s *SessionService) saveMVC(ctx context.Context, sessionID string, mvc []byte) error {
	if _, err := s.kv.Put(ctx, sessionID, mvc); err != nil {
		return fmt.Errorf("failed to put key value: %w", err)
	}
	return nil
}

func (s *SessionService) upsertSessionID(r *http.Request, w http.ResponseWriter) (string, error) {
	sess, err := s.store.Get(r, "connections")
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	id, ok := sess.Values["id"].(string)

	if !ok {
		id = toolbelt.NextEncodedID()
		sess.Values["id"] = id
		if err := sess.Save(r, w); err != nil {
			return "", fmt.Errorf("failed to save session: %w", err)
		}
	}

	return id, nil
}
