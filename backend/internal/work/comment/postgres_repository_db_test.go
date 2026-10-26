package comment_test

// postgres_repository_db_test.go runs the task-comment repository against the
// real schema as kmuhub_app (NOSUPERUSER NOBYPASSRLS), with RLS enforced.
// service_test.go only sees the in-memory mockCommentRepo, so every SQL
// statement in postgres_repository.go was unexercised.

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/comment"
)

type commentFixture struct {
	pool    *pgxpool.Pool
	repo    *comment.PostgresRepository
	tenant  uuid.UUID
	project uuid.UUID
	ctx     context.Context
}

func newCommentFixture(t *testing.T) *commentFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Comment Repo Tenant")
	f := &commentFixture{
		pool:   pool,
		repo:   comment.NewPostgresRepository(pool),
		tenant: tenant,
		ctx:    testutil.WithTenantCtx(context.Background(), tenant),
	}
	creator := f.user(t, "Ada", "Lovelace")
	f.project = testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":        tenant,
		"name":             "Comment Repo Project",
		"project_key":      "CR" + uuid.NewString()[:6],
		"next_task_number": 1,
		"created_by":       creator,
	})
	t.Cleanup(func() {
		sys := testutil.WithSystemCtx(context.Background())
		// task_comments cascade from tasks; tasks and projects reference users.
		for _, table := range []string{"tasks", "projects", "users"} {
			if _, err := pool.Exec(sys, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1", table), tenant); err != nil {
				t.Logf("cleanup %s: %v", table, err)
			}
		}
		testutil.CleanupRow(t, pool, "tenants", tenant)
	})
	return f
}

func (f *commentFixture) user(t *testing.T, first, last string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, f.pool, "users", map[string]any{
		"tenant_id":     f.tenant,
		"email":         fmt.Sprintf("wp-comment-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"first_name":    first,
		"last_name":     last,
	})
}

func (f *commentFixture) task(t *testing.T, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, f.pool, "tasks", map[string]any{
		"tenant_id":   f.tenant,
		"project_id":  f.project,
		"title":       "Comment task " + uuid.NewString()[:8],
		"task_number": int(uuid.New().ID())%900000 + 100000,
		"created_by":  createdBy,
	})
}

// standaloneTask seeds a task without a project (tasks.project_id IS NULL).
func (f *commentFixture) standaloneTask(t *testing.T, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, f.pool, "tasks", map[string]any{
		"tenant_id":   f.tenant,
		"title":       "Standalone comment task " + uuid.NewString()[:8],
		"task_number": int(uuid.New().ID())%900000 + 100000,
		"created_by":  createdBy,
	})
}

func (f *commentFixture) add(t *testing.T, taskID, author uuid.UUID, content string, quoted *uuid.UUID, at time.Time) *models.TaskComment {
	t.Helper()
	c := &models.TaskComment{
		ID: uuid.New(), TaskID: taskID, AuthorID: author, Content: content,
		QuotedCommentID: quoted, CreatedAt: at, UpdatedAt: at,
	}
	require.NoError(t, f.repo.Create(f.ctx, c))
	return c
}

func now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

func (f *commentFixture) otherTenantCtx(t *testing.T) context.Context {
	t.Helper()
	other := uuid.New()
	testutil.EnsureTenant(t, f.pool, other, "Comment Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "tenants", other) })
	return testutil.WithTenantCtx(context.Background(), other)
}

func TestCommentRepo_CreateInheritsTenantAndRoundtrips(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "Grace", "Hopper")
	task := f.task(t, author)
	created := now()

	c := f.add(t, task, author, "first comment", nil, created)

	got, err := f.repo.GetByID(f.ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.ID, got.ID)
	assert.Equal(t, task, got.TaskID)
	assert.Equal(t, author, got.AuthorID)
	assert.Equal(t, "Grace Hopper", got.AuthorName)
	assert.Equal(t, "first comment", got.Content)
	assert.Nil(t, got.QuotedCommentID)
	assert.Nil(t, got.QuotedPreview)
	assert.True(t, created.Equal(got.CreatedAt))
	assert.True(t, created.Equal(got.UpdatedAt))

	// The INSERT derives tenant_id from the task; RLS lets it through only if it matches.
	var tenant uuid.UUID
	require.NoError(t, f.pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT tenant_id FROM task_comments WHERE id = $1`, c.ID).Scan(&tenant))
	assert.Equal(t, f.tenant, tenant)
}

func TestCommentRepo_CreateOnStandaloneTask(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "Alan", "Turing")
	task := f.standaloneTask(t, author)

	c := f.add(t, task, author, "comment on a task without project", nil, now())

	got, err := f.repo.GetByID(f.ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, task, got.TaskID)
	assert.Equal(t, "comment on a task without project", got.Content)

	var tenant uuid.UUID
	require.NoError(t, f.pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT tenant_id FROM task_comments WHERE id = $1`, c.ID).Scan(&tenant))
	assert.Equal(t, f.tenant, tenant)

	// A foreign tenant still cannot comment on the standalone task.
	err = f.repo.Create(f.otherTenantCtx(t), &models.TaskComment{
		ID: uuid.New(), TaskID: task, AuthorID: author, Content: "intruder",
		CreatedAt: now(), UpdatedAt: now(),
	})
	require.Error(t, err)
}

func TestCommentRepo_CreateForForeignTaskIsRejected(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "A", "B")
	task := f.task(t, author)

	// Another tenant cannot see the task, so the tenant subselect yields NULL.
	err := f.repo.Create(f.otherTenantCtx(t), &models.TaskComment{
		ID: uuid.New(), TaskID: task, AuthorID: author, Content: "x", CreatedAt: now(), UpdatedAt: now(),
	})
	assert.Error(t, err)

	var n int
	require.NoError(t, f.pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM task_comments WHERE task_id = $1`, task).Scan(&n))
	assert.Zero(t, n)
}

func TestCommentRepo_GetByID_NotFoundAndCrossTenant(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "A", "B")
	c := f.add(t, f.task(t, author), author, "hidden", nil, now())

	_, err := f.repo.GetByID(f.ctx, uuid.New())
	assert.ErrorIs(t, err, comment.ErrNotFound)

	_, err = f.repo.GetByID(f.otherTenantCtx(t), c.ID)
	assert.ErrorIs(t, err, comment.ErrNotFound, "RLS must hide another tenant's comment")
}

func TestCommentRepo_QuotedPreviewTruncatedTo200(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "Q", "Uoter")
	task := f.task(t, author)
	long := f.add(t, task, author, strings.Repeat("x", 250), nil, now())
	short := f.add(t, task, author, "short original", nil, now())

	replyLong := f.add(t, task, author, "re long", &long.ID, now())
	replyShort := f.add(t, task, author, "re short", &short.ID, now())

	got, err := f.repo.GetByID(f.ctx, replyLong.ID)
	require.NoError(t, err)
	require.NotNil(t, got.QuotedCommentID)
	assert.Equal(t, long.ID, *got.QuotedCommentID)
	require.NotNil(t, got.QuotedPreview)
	assert.Len(t, *got.QuotedPreview, 200)

	got, err = f.repo.GetByID(f.ctx, replyShort.ID)
	require.NoError(t, err)
	require.NotNil(t, got.QuotedPreview)
	assert.Equal(t, "short original", *got.QuotedPreview)
}

func TestCommentRepo_List_OrderTotalAndQuotedAuthor(t *testing.T) {
	f := newCommentFixture(t)
	ada := f.user(t, "Ada", "Lovelace")
	bob := f.user(t, "Bob", "Builder")
	task := f.task(t, ada)
	otherTask := f.task(t, ada)
	base := now().Add(-time.Hour)

	third := f.add(t, task, ada, "third", nil, base.Add(3*time.Minute))
	first := f.add(t, task, ada, "first", nil, base.Add(1*time.Minute))
	reply := f.add(t, task, bob, "second reply", &first.ID, base.Add(2*time.Minute))
	f.add(t, otherTask, ada, "other task", nil, base)

	items, total, err := f.repo.List(f.ctx, task, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 3, total, "comments of other tasks must not be counted")
	require.Len(t, items, 3)
	assert.Equal(t, []uuid.UUID{first.ID, reply.ID, third.ID}, []uuid.UUID{items[0].ID, items[1].ID, items[2].ID}, "oldest first")

	assert.Equal(t, "Ada Lovelace", items[0].AuthorName)
	assert.Nil(t, items[0].QuotedAuthorName)
	assert.Equal(t, "Bob Builder", items[1].AuthorName)
	require.NotNil(t, items[1].QuotedAuthorName)
	assert.Equal(t, "Ada Lovelace", *items[1].QuotedAuthorName)
	require.NotNil(t, items[1].QuotedPreview)
	assert.Equal(t, "first", *items[1].QuotedPreview)
}

func TestCommentRepo_List_EmptyAndCrossTenant(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "A", "B")
	task := f.task(t, author)

	items, total, err := f.repo.List(f.ctx, task, 1, 20)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, items)

	f.add(t, task, author, "visible to owner only", nil, now())
	items, total, err = f.repo.List(f.otherTenantCtx(t), task, 1, 20)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, items)
}

func TestCommentRepo_List_PagingAndClamping(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "P", "Ager")
	task := f.task(t, author)
	base := now().Add(-time.Hour)
	ids := make([]uuid.UUID, 0, 5)
	for i := 0; i < 5; i++ {
		ids = append(ids, f.add(t, task, author, fmt.Sprintf("c%d", i), nil, base.Add(time.Duration(i)*time.Minute)).ID)
	}

	page2, total, err := f.repo.List(f.ctx, task, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page2, 2)
	assert.Equal(t, []uuid.UUID{ids[2], ids[3]}, []uuid.UUID{page2[0].ID, page2[1].ID})

	last, _, err := f.repo.List(f.ctx, task, 3, 2)
	require.NoError(t, err)
	require.Len(t, last, 1)
	assert.Equal(t, ids[4], last[0].ID)

	// page < 1 is treated as page 1.
	first, _, err := f.repo.List(f.ctx, task, 0, 2)
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, ids[0], first[0].ID)

	// pageSize < 1 falls back to the default of 20 (all five rows); the upper
	// bound is covered by the oversized-page-size test.
	for _, size := range []int{0, -3} {
		all, _, err := f.repo.List(f.ctx, task, 1, size)
		require.NoError(t, err)
		assert.Len(t, all, 5, "pageSize %d", size)
	}
}

func TestCommentRepo_List_OversizedPageSizeFallsBackToDefault(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "B", "Ig")
	task := f.task(t, author)
	base := now().Add(-time.Hour)
	for i := 0; i < 22; i++ {
		f.add(t, task, author, fmt.Sprintf("c%d", i), nil, base.Add(time.Duration(i)*time.Second))
	}

	// 101 is rejected and replaced by 20 (not clamped to 100): 22 rows give 20, not 22.
	items, total, err := f.repo.List(f.ctx, task, 1, 101)
	require.NoError(t, err)
	assert.Equal(t, 22, total)
	assert.Len(t, items, 20)

	// 100 itself is accepted as given.
	items, _, err = f.repo.List(f.ctx, task, 1, 100)
	require.NoError(t, err)
	assert.Len(t, items, 22)
}

func TestCommentRepo_Update_ChangesContentAndTimestampOnly(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "U", "Pdater")
	task := f.task(t, author)
	created := now().Add(-time.Hour)
	target := f.add(t, task, author, "before", nil, created)
	bystander := f.add(t, task, author, "untouched", nil, created)

	require.NoError(t, f.repo.Update(f.ctx, target.ID, "after"))

	got, err := f.repo.GetByID(f.ctx, target.ID)
	require.NoError(t, err)
	assert.Equal(t, "after", got.Content)
	assert.True(t, created.Equal(got.CreatedAt), "created_at must not change")
	assert.True(t, got.UpdatedAt.After(created), "updated_at must move forward")

	other, err := f.repo.GetByID(f.ctx, bystander.ID)
	require.NoError(t, err)
	assert.Equal(t, "untouched", other.Content)
	assert.True(t, created.Equal(other.UpdatedAt))
}

func TestCommentRepo_Update_CrossTenantIsNoop(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "A", "B")
	c := f.add(t, f.task(t, author), author, "mine", nil, now())

	require.NoError(t, f.repo.Update(f.otherTenantCtx(t), c.ID, "hijacked"))

	got, err := f.repo.GetByID(f.ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "mine", got.Content)
}

func TestCommentRepo_Delete_RemovesOnlyTargetAndNullsQuotes(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "D", "Eleter")
	task := f.task(t, author)
	original := f.add(t, task, author, "original", nil, now())
	reply := f.add(t, task, author, "reply", &original.ID, now())
	keep := f.add(t, task, author, "keep", nil, now())

	require.NoError(t, f.repo.Delete(f.ctx, original.ID))

	_, err := f.repo.GetByID(f.ctx, original.ID)
	assert.ErrorIs(t, err, comment.ErrNotFound)

	got, err := f.repo.GetByID(f.ctx, reply.ID)
	require.NoError(t, err)
	assert.Nil(t, got.QuotedCommentID, "quote reference is cleared (ON DELETE SET NULL)")
	assert.Nil(t, got.QuotedPreview)

	_, err = f.repo.GetByID(f.ctx, keep.ID)
	assert.NoError(t, err)

	// Deleting an unknown id is not an error.
	assert.NoError(t, f.repo.Delete(f.ctx, uuid.New()))
}

func TestCommentRepo_Delete_CrossTenantIsNoop(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "A", "B")
	c := f.add(t, f.task(t, author), author, "survivor", nil, now())

	require.NoError(t, f.repo.Delete(f.otherTenantCtx(t), c.ID))

	_, err := f.repo.GetByID(f.ctx, c.ID)
	assert.NoError(t, err)
}

func TestCommentRepo_CancelledContextFails(t *testing.T) {
	f := newCommentFixture(t)
	author := f.user(t, "A", "B")
	task := f.task(t, author)
	c := f.add(t, task, author, "x", nil, now())

	ctx, cancel := context.WithCancel(f.ctx)
	cancel()

	assert.Error(t, f.repo.Create(ctx, &models.TaskComment{ID: uuid.New(), TaskID: task, AuthorID: author, Content: "y", CreatedAt: now(), UpdatedAt: now()}))
	_, err := f.repo.GetByID(ctx, c.ID)
	require.Error(t, err)
	assert.NotErrorIs(t, err, comment.ErrNotFound)
	_, _, err = f.repo.List(ctx, task, 1, 20)
	assert.Error(t, err)
	assert.Error(t, f.repo.Update(ctx, c.ID, "z"))
	assert.Error(t, f.repo.Delete(ctx, c.ID))
}
