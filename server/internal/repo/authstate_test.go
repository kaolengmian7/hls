package repo_test

import (
	"context"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestAuthState_CreateAndGet(t *testing.T) {
	conn := testConn(t)
	r := repo.NewAuthStateRepo(conn)
	ctx := context.Background()

	state := "test-state-" + t.Name()
	if err := r.Create(ctx, state); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	as, err := r.Get(ctx, state)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if as.JWT.Valid {
		t.Fatal("JWT should be NULL after Create()")
	}

	// cleanup
	r.Delete(ctx, state)
}

func TestAuthState_SetJWTAndGet(t *testing.T) {
	conn := testConn(t)
	r := repo.NewAuthStateRepo(conn)
	ctx := context.Background()

	state := "test-state-jwt-" + t.Name()
	r.Create(ctx, state)
	if err := r.SetJWT(ctx, state, "mytoken"); err != nil {
		t.Fatalf("SetJWT() error = %v", err)
	}

	as, err := r.Get(ctx, state)
	if err != nil {
		t.Fatalf("Get() after SetJWT error = %v", err)
	}
	if !as.JWT.Valid || as.JWT.String != "mytoken" {
		t.Fatalf("JWT = %v, want {String:mytoken Valid:true}", as.JWT)
	}

	r.Delete(ctx, state)
}

func TestAuthState_GetExpired(t *testing.T) {
	conn := testConn(t)
	r := repo.NewAuthStateRepo(conn)
	ctx := context.Background()

	// Insert a state that is already expired (expires_at in the past)
	db, _ := conn.RawDB()
	state := "test-state-expired-" + t.Name()
	db.ExecContext(ctx,
		"INSERT INTO auth_states (state, expires_at) VALUES (?, DATE_SUB(NOW(), INTERVAL 1 MINUTE))",
		state)

	_, err := r.Get(ctx, state)
	if err == nil {
		t.Fatal("Get() on expired state should return error (not found after cleanup)")
	}
}
