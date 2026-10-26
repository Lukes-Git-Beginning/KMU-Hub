package gateway

import (
	"context"
	"fmt"
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
	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/inventar"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

// A transfer is one "out" plus one "in" in a single transaction, so a member who
// books single movements (inventar:movement:create) transfers too. The actor of
// both legs is the caller, whatever performed_by the body names. Production
// chain: real seed through ResolveTokenPermissions, real Auth middleware and
// route registration, a real inventar gRPC server and the Postgres repository.

var inventarTransferTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000f5")

func TestInventarTransfer_MemberAgainstRealSeed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, inventarTransferTenant, "InventarTransferTenant")
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	inventarv1.RegisterInventarServiceServer(srv, server.NewInventarGRPCServer(inventar.NewService(inventar.NewPostgresRepository(pool))))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("inventar", lis.Addr().String())
	t.Cleanup(reg.Close)

	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_INVENTAR_ENABLED" {
			return "true"
		}
		return ""
	})
	router := chi.NewRouter()
	NewInventarRoutes(reg, flags).RegisterRoutes(router, middleware.Auth(authSvc))

	tokenFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     inventarTransferTenant,
			"email":         "inventar-transfer-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "Transfer",
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
			userID, roleID, inventarTransferTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, inventarTransferTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, inventarTransferTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID, token
	}

	memberID, memberToken := tokenFor("member")
	colleagueID, _ := tokenFor("member")
	_, readonlyToken := tokenFor("readonly")

	item := func(quantity int64) uuid.UUID {
		t.Helper()
		id := testutil.SeedRow(t, pool, "inventory_items", map[string]any{
			"id":        uuid.New(),
			"tenant_id": inventarTransferTenant,
			"name":      "Transfer item",
			"sku":       "TR-" + uuid.NewString()[:8],
			"quantity":  quantity,
			"unit":      "Stk",
		})
		t.Cleanup(func() {
			if _, err := pool.Exec(sysCtx, `DELETE FROM inventory_movements WHERE item_id = $1`, id); err != nil {
				t.Errorf("cleanup movements: %v", err)
			}
			testutil.CleanupRow(t, pool, "inventory_items", id)
		})
		return id
	}
	quantity := func(t *testing.T, id uuid.UUID) int64 {
		t.Helper()
		var q int64
		if err := pool.QueryRow(sysCtx, `SELECT quantity FROM inventory_items WHERE id = $1`, id).Scan(&q); err != nil {
			t.Fatalf("read quantity: %v", err)
		}
		return q
	}
	transfer := func(t *testing.T, from, to uuid.UUID, qty int, token string) *httptest.ResponseRecorder {
		t.Helper()
		body := fmt.Sprintf(`{"from_item_id":%q,"to_item_id":%q,"quantity":%d,"performed_by":%q,"reason":"Umlagerung"}`,
			from, to, qty, colleagueID)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inventar/transfer", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("member transfers stock and is the actor of both legs", func(t *testing.T) {
		from, to := item(5), item(1)
		rec := transfer(t, from, to, 2, memberToken)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("member transfer = %d, want 204, body %s", rec.Code, rec.Body.String())
		}
		if got, want := quantity(t, from), int64(3); got != want {
			t.Fatalf("source quantity = %d, want %d", got, want)
		}
		if got, want := quantity(t, to), int64(3); got != want {
			t.Fatalf("target quantity = %d, want %d", got, want)
		}
		rows, err := pool.Query(sysCtx,
			`SELECT movement_type, performed_by FROM inventory_movements WHERE item_id IN ($1, $2)`, from, to)
		if err != nil {
			t.Fatalf("read movements: %v", err)
		}
		defer rows.Close()
		n := 0
		for rows.Next() {
			var typ string
			var by *uuid.UUID
			if err := rows.Scan(&typ, &by); err != nil {
				t.Fatalf("scan movement: %v", err)
			}
			n++
			if typ != string(inventar.MovementTypeTransfer) {
				t.Errorf("movement type = %q, want transfer", typ)
			}
			if by == nil || *by != memberID {
				t.Errorf("performed_by = %v, want the caller %s (body named %s)", by, memberID, colleagueID)
			}
		}
		if n != 2 {
			t.Fatalf("expected two transfer movements, got %d", n)
		}
	})

	t.Run("member cannot transfer more than the source holds", func(t *testing.T) {
		from, to := item(1), item(0)
		rec := transfer(t, from, to, 2, memberToken)
		if rec.Code < 400 || rec.Code >= 500 {
			t.Fatalf("overdrawn transfer = %d, want 4xx, body %s", rec.Code, rec.Body.String())
		}
		if quantity(t, from) != 1 || quantity(t, to) != 0 {
			t.Fatalf("refused transfer must leave both items unchanged")
		}
	})

	t.Run("readonly is refused", func(t *testing.T) {
		from, to := item(3), item(0)
		rec := transfer(t, from, to, 1, readonlyToken)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("readonly transfer = %d, want 403, body %s", rec.Code, rec.Body.String())
		}
		if quantity(t, from) != 3 {
			t.Fatalf("refused transfer must leave the source at 3")
		}
	})
}
