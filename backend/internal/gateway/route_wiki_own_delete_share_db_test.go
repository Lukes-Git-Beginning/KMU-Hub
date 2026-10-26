package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/wiki"
	wikiv1 "github.com/kmuhub/kmuhub/proto/wiki/v1"
)

// A member deletes and shares the wiki articles they wrote (decision
// 2026-10-25), not a colleague's; manager keeps doing both tenant-wide. The
// chain is the production one: the real seed through ResolveTokenPermissions,
// the real Auth middleware and route registration, a real wiki gRPC server
// behind TenantInboundUnaryInterceptor and the Postgres repository.

var wikiOwnDeleteShareTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000d3")

func TestWikiOwnDeleteShare_MemberAgainstRealSeed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, wikiOwnDeleteShareTenant, "WikiOwnDeleteShareTenant")
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	wikiv1.RegisterWikiServiceServer(srv, server.NewWikiGRPCServer(wiki.NewService(wiki.NewPostgresRepository(pool))))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("wiki", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	newWikiRoutes(reg).RegisterRoutes(router, middleware.Auth(authSvc))

	tokenFor := func(role string) string {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     wikiOwnDeleteShareTenant,
			"email":         "wiki-own-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "Wiki",
			"last_name":     role,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
		var roleID uuid.UUID
		if err := pool.QueryRow(sysCtx,
			`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID); err != nil {
			t.Fatalf("preset role %q: %v", role, err)
		}
		if _, err := pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
			userID, roleID, wikiOwnDeleteShareTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, wikiOwnDeleteShareTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, wikiOwnDeleteShareTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return token
	}

	do := func(t *testing.T, method, token, path, body string) (int, map[string]any) {
		t.Helper()
		var rd io.Reader
		if body != "" {
			rd = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, "/api/v1"+path, rd)
		req.Header.Set("Authorization", "Bearer "+token)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", uuid.NewString())
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	idOf := func(t *testing.T, body map[string]any, key string) string {
		t.Helper()
		inner, _ := body[key].(map[string]any)
		id, _ := inner["id"].(string)
		if id == "" {
			t.Fatalf("no %s.id in %v", key, body)
		}
		return id
	}
	createArticle := func(token, title string) string {
		t.Helper()
		code, out := do(t, http.MethodPost, token, "/wiki/articles",
			`{"title":"`+title+`","slug":"own-`+uuid.NewString()[:8]+`","published":true}`)
		if code != http.StatusCreated {
			t.Fatalf("create article %q: %d %v", title, code, out)
		}
		id := idOf(t, out, "article")
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "wiki_articles", uuid.MustParse(id)) })
		return id
	}
	attach := func(token, articleID string) string {
		t.Helper()
		code, out := do(t, http.MethodPost, token, "/wiki/articles/"+articleID+"/attachments",
			`{"file_ref":"wiki/`+uuid.NewString()+`.pdf","mime":"application/pdf","size":12}`)
		if code != http.StatusCreated && code != http.StatusOK {
			t.Fatalf("upload attachment: %d %v", code, out)
		}
		return idOf(t, out, "attachment")
	}
	share := func(t *testing.T, token, articleID string) (int, string) {
		t.Helper()
		code, out := do(t, http.MethodPost, token, "/wiki/articles/"+articleID+"/share", `{"permissions":["read"]}`)
		if code != http.StatusCreated {
			return code, ""
		}
		return code, idOf(t, out, "token")
	}
	revokedAt := func(t *testing.T, articleID, tokenID string) any {
		t.Helper()
		var at *time.Time
		if err := pool.QueryRow(sysCtx,
			`SELECT revoked_at FROM wiki_share_tokens WHERE id = $1 AND article_id = $2`, tokenID, articleID).Scan(&at); err != nil {
			t.Fatalf("read token %s: %v", tokenID, err)
		}
		if at == nil {
			return nil
		}
		return *at
	}
	articleExists := func(t *testing.T, id string) bool {
		t.Helper()
		var ok bool
		if err := pool.QueryRow(sysCtx, `SELECT EXISTS(SELECT 1 FROM wiki_articles WHERE id = $1)`, id).Scan(&ok); err != nil {
			t.Fatalf("read article %s: %v", id, err)
		}
		return ok
	}
	attachmentExists := func(t *testing.T, id string) bool {
		t.Helper()
		var ok bool
		if err := pool.QueryRow(sysCtx, `SELECT EXISTS(SELECT 1 FROM wiki_attachments WHERE id = $1)`, id).Scan(&ok); err != nil {
			t.Fatalf("read attachment %s: %v", id, err)
		}
		return ok
	}

	managerToken := tokenFor("manager")
	memberToken := tokenFor("member")
	colleagueToken := tokenFor("member")

	own := createArticle(memberToken, "Eigene Seite")
	ownToDelete := createArticle(memberToken, "Eigene Seite zum Löschen")
	colleagues := createArticle(colleagueToken, "Seite des Kollegen")
	managers := createArticle(managerToken, "Seite des Managers")

	ownAttachment := attach(memberToken, own)
	colleagueAttachment := attach(colleagueToken, colleagues)

	t.Run("member shares their own article and revokes the link", func(t *testing.T) {
		code, tokenID := share(t, memberToken, own)
		if code != http.StatusCreated {
			t.Fatalf("member shares own article = %d, want 201", code)
		}
		if code, out := do(t, http.MethodDelete, memberToken, "/wiki/share/"+tokenID, ""); code != http.StatusNoContent {
			t.Fatalf("member revokes own link = %d %v, want 204", code, out)
		}
		if revokedAt(t, own, tokenID) == nil {
			t.Error("own link still live after a 204 revoke")
		}
	})

	t.Run("member cannot share a colleague's or a manager's article", func(t *testing.T) {
		for _, id := range []string{colleagues, managers} {
			if code, _ := share(t, memberToken, id); code != http.StatusNotFound {
				t.Errorf("member shares foreign article %s = %d, want 404", id, code)
			}
		}
	})

	t.Run("member cannot revoke a link on a colleague's article", func(t *testing.T) {
		code, tokenID := share(t, colleagueToken, colleagues)
		if code != http.StatusCreated {
			t.Fatalf("colleague shares own article = %d, want 201", code)
		}
		if code, out := do(t, http.MethodDelete, memberToken, "/wiki/share/"+tokenID, ""); code != http.StatusNotFound {
			t.Errorf("member revokes colleague's link = %d %v, want 404", code, out)
		}
		if revokedAt(t, colleagues, tokenID) != nil {
			t.Error("colleague's link was cut by a refused revoke")
		}
	})

	t.Run("member deletes attachments on their own article only", func(t *testing.T) {
		if code, out := do(t, http.MethodDelete, memberToken, "/wiki/attachments/"+colleagueAttachment, ""); code != http.StatusNotFound {
			t.Errorf("member deletes colleague's attachment = %d %v, want 404", code, out)
		}
		if !attachmentExists(t, colleagueAttachment) {
			t.Error("colleague's attachment is gone after a refused delete")
		}
		if code, out := do(t, http.MethodDelete, memberToken, "/wiki/articles/"+own+"/attachments/"+ownAttachment, ""); code != http.StatusNoContent {
			t.Errorf("member deletes own attachment = %d %v, want 204", code, out)
		}
		if attachmentExists(t, ownAttachment) {
			t.Error("own attachment survives a 204 delete")
		}
	})

	t.Run("member deletes their own article, not a colleague's or a manager's", func(t *testing.T) {
		for _, id := range []string{colleagues, managers} {
			if code, out := do(t, http.MethodDelete, memberToken, "/wiki/articles/"+id, ""); code != http.StatusNotFound {
				t.Errorf("member deletes foreign article %s = %d %v, want 404", id, code, out)
			}
			if !articleExists(t, id) {
				t.Errorf("foreign article %s is gone after a refused delete", id)
			}
		}
		if code, out := do(t, http.MethodDelete, memberToken, "/wiki/articles/"+ownToDelete, ""); code != http.StatusNoContent {
			t.Fatalf("member deletes own article = %d %v, want 204", code, out)
		}
		if articleExists(t, ownToDelete) {
			t.Error("own article survives a 204 delete")
		}
	})

	t.Run("manager still shares and deletes any article tenant-wide", func(t *testing.T) {
		code, tokenID := share(t, managerToken, colleagues)
		if code != http.StatusCreated {
			t.Fatalf("manager shares colleague's article = %d, want 201", code)
		}
		if code, out := do(t, http.MethodDelete, managerToken, "/wiki/share/"+tokenID, ""); code != http.StatusNoContent {
			t.Errorf("manager revokes link on colleague's article = %d %v, want 204", code, out)
		}
		if code, out := do(t, http.MethodDelete, managerToken, "/wiki/articles/"+colleagues, ""); code != http.StatusNoContent {
			t.Errorf("manager deletes colleague's article = %d %v, want 204", code, out)
		}
	})
}
