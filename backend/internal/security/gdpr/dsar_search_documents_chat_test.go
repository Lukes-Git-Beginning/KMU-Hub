package gdpr

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSearchByQuery_UserDocumentsAndChatAuthorship_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Docs Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Docs Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Dora", "Dokumentarin", true)
	colleagueID := seedDSARUser(t, pool, tenantOwn, "Kai", "Kollege", true)
	foreignUserID := seedDSARUser(t, pool, tenantOther, "Dora", "Dokumentarin", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	defer testutil.CleanupRow(t, pool, "users", colleagueID)
	defer testutil.CleanupRow(t, pool, "users", foreignUserID)

	now := time.Now().UTC().Truncate(time.Second)
	// t.Cleanup runs after the deferred user/tenant deletes above, so children
	// are removed in reverse seed order via the deferred list below instead.
	var cleanups []func()
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}()
	seed := func(table string, cols map[string]any) uuid.UUID {
		id := testutil.SeedRow(t, pool, table, cols)
		cleanups = append(cleanups, func() { testutil.CleanupRow(t, pool, table, id) })
		return id
	}

	// --- documents ------------------------------------------------------------
	folderOwn := seed("document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Verträge", "space_type": "personal", "space_id": subjectID, "created_by": subjectID,
	})
	folderOther := seed("document_folders", map[string]any{
		"tenant_id": tenantOther, "name": "Fremd", "space_type": "personal", "space_id": foreignUserID, "created_by": foreignUserID,
	})
	newFile := func(tenantID, folderID, ownerID uuid.UUID, name string, deleted bool) uuid.UUID {
		cols := map[string]any{
			"tenant_id": tenantID, "folder_id": folderID, "owner_id": ownerID, "filename": name,
			"mime_type": "application/pdf", "file_size": 2048, "storage_key": "k/" + uuid.NewString(),
			"is_deleted": deleted,
		}
		if deleted {
			cols["deleted_at"] = now
		}
		return seed("document_files", cols)
	}
	fileLive := newFile(tenantOwn, folderOwn, subjectID, "angebot.pdf", false)
	newFile(tenantOwn, folderOwn, subjectID, "entwurf.pdf", true)
	fileColleague := newFile(tenantOwn, folderOwn, colleagueID, "kollege.pdf", false)
	fileForeign := newFile(tenantOther, folderOther, foreignUserID, "fremd.pdf", false)

	seed("document_file_comments", map[string]any{
		"tenant_id": tenantOwn, "file_id": fileColleague, "author_id": subjectID, "content": "Bitte Preis prüfen",
	})
	seed("document_file_comments", map[string]any{
		"tenant_id": tenantOwn, "file_id": fileLive, "author_id": colleagueID, "content": "Kommentar des Kollegen",
	})
	seed("document_file_comments", map[string]any{
		"tenant_id": tenantOther, "file_id": fileForeign, "author_id": foreignUserID, "content": "Fremder Kommentar",
	})
	seed("document_shares", map[string]any{
		"tenant_id": tenantOwn, "entity_type": "file", "entity_id": fileColleague,
		"shared_with_user_id": subjectID, "shared_by": colleagueID, "permission": "read",
	})
	seed("document_shares", map[string]any{
		"tenant_id": tenantOwn, "entity_type": "file", "entity_id": fileLive,
		"shared_with_user_id": colleagueID, "shared_by": subjectID, "permission": "write",
	})
	seed("document_shares", map[string]any{
		"tenant_id": tenantOwn, "entity_type": "file", "entity_id": fileColleague,
		"shared_with_user_id": colleagueID, "shared_by": colleagueID, "permission": "read",
	})

	// --- chat -----------------------------------------------------------------
	channelOwn := seed("channels", map[string]any{"tenant_id": tenantOwn, "name": "projekt-nord", "created_by": colleagueID})
	channelOther := seed("channels", map[string]any{"tenant_id": tenantOther, "name": "fremd-kanal", "created_by": foreignUserID})
	msgOwn := seed("messages", map[string]any{
		"tenant_id": tenantOwn, "channel_id": channelOwn, "content": "Hallo", "created_by": colleagueID,
	})
	msgOther := seed("messages", map[string]any{
		"tenant_id": tenantOther, "channel_id": channelOther, "content": "Hallo fremd", "created_by": foreignUserID,
	})
	newChatFile := func(tenantID, channelID, uploader uuid.UUID, name string, deleted bool) {
		cols := map[string]any{
			"tenant_id": tenantID, "channel_id": channelID, "uploaded_by": uploader, "filename": name,
			"mime_type": "image/png", "file_size": 10, "storage_key": "c/" + uuid.NewString(), "is_deleted": deleted,
		}
		if deleted {
			cols["deleted_at"] = now
		}
		seed("chat_files", cols)
	}
	newChatFile(tenantOwn, channelOwn, subjectID, "baustelle.png", false)
	newChatFile(tenantOwn, channelOwn, subjectID, "geloescht.png", true)
	newChatFile(tenantOwn, channelOwn, colleagueID, "kollege.png", false)
	newChatFile(tenantOther, channelOther, foreignUserID, "fremd.png", false)

	// message_reactions has no id column; the rows go away with their message.
	ctxSys := testutil.WithSystemCtx(context.Background())
	for _, r := range []struct{ tenant, msg, user uuid.UUID }{
		{tenantOwn, msgOwn, subjectID}, {tenantOwn, msgOwn, colleagueID}, {tenantOther, msgOther, foreignUserID},
	} {
		_, err := pool.Exec(ctxSys,
			`INSERT INTO message_reactions (tenant_id, message_id, user_id, emoji) VALUES ($1, $2, $3, $4)`,
			r.tenant, r.msg, r.user, "+1")
		require.NoError(t, err)
	}

	// --- tasks ----------------------------------------------------------------
	taskOwn := seed("tasks", map[string]any{
		"tenant_id": tenantOwn, "task_number": 9001, "title": "Dach decken", "created_by": colleagueID,
	})
	taskOther := seed("tasks", map[string]any{
		"tenant_id": tenantOther, "task_number": 9001, "title": "Fremde Aufgabe", "created_by": foreignUserID,
	})
	newTaskFile := func(tenantID, taskID, uploader uuid.UUID, name string) {
		seed("task_files", map[string]any{
			"tenant_id": tenantID, "task_id": taskID, "uploaded_by": uploader, "filename": name,
			"mime_type": "image/png", "file_size": 5, "storage_key": "t/" + uuid.NewString(),
		})
	}
	newTaskFile(tenantOwn, taskOwn, subjectID, "skizze.png")
	newTaskFile(tenantOwn, taskOwn, colleagueID, "kollege-skizze.png")
	newTaskFile(tenantOther, taskOther, foreignUserID, "fremd-skizze.png")
	seed("task_activities", map[string]any{
		"tenant_id": tenantOwn, "task_id": taskOwn, "actor_id": subjectID, "action": "updated",
		"field_name": "status", "old_value": `"open"`, "new_value": `"done"`,
	})
	seed("task_activities", map[string]any{
		"tenant_id": tenantOwn, "task_id": taskOwn, "actor_id": colleagueID, "action": "created",
	})
	seed("task_activities", map[string]any{
		"tenant_id": tenantOther, "task_id": taskOther, "actor_id": foreignUserID, "action": "created",
	})

	// --- wiki -----------------------------------------------------------------
	newArticle := func(tenantID, authorID uuid.UUID, title string) uuid.UUID {
		return seed("wiki_articles", map[string]any{
			"tenant_id": tenantID, "title": title, "slug": "s-" + uuid.NewString(), "author_id": authorID,
			"content": `{"secret":"ARTICLEBODYTEXT"}`, "published": true,
		})
	}
	articleOwn := newArticle(tenantOwn, subjectID, "Onboarding-Leitfaden")
	articleColleague := newArticle(tenantOwn, colleagueID, "Kollegenartikel")
	articleOther := newArticle(tenantOther, foreignUserID, "Fremder Artikel")
	newAttachment := func(tenantID, articleID, uploader uuid.UUID, ref string) {
		seed("wiki_attachments", map[string]any{
			"tenant_id": tenantID, "article_id": articleID, "file_ref": ref, "mime": "image/png", "size": 7,
			"uploaded_by": uploader,
		})
	}
	newAttachment(tenantOwn, articleColleague, subjectID, "w/a")
	newAttachment(tenantOwn, articleOwn, colleagueID, "w/b")
	newAttachment(tenantOther, articleOther, foreignUserID, "w/c")

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Dokumentarin")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	person := persons[0]

	docs := recordMaps(dsarModule(t, person, "Eigene Dokumente"))
	require.Len(t, docs, 2, "live + deleted file of the subject, never the colleague's")
	deletedAt := map[string]string{}
	for _, d := range docs {
		deletedAt[d["Dateiname"]] = d["Gelöscht am"]
	}
	assert.Empty(t, deletedAt["angebot.pdf"])
	assert.NotEmpty(t, deletedAt["entwurf.pdf"], "deleted file is flagged, not hidden")

	comments := recordMaps(dsarModule(t, person, "Dokument-Kommentare"))
	require.Len(t, comments, 1)
	assert.Equal(t, "Bitte Preis prüfen", comments[0]["Kommentar"])
	assert.Equal(t, "kollege.pdf", comments[0]["Dokument"])

	shares := recordMaps(dsarModule(t, person, "Dokument-Freigaben"))
	require.Len(t, shares, 2, "received + granted share, not the colleague's self-share")
	byRole := map[string]string{}
	for _, s := range shares {
		byRole[s["Rolle"]] = s["Objekt"] + "|" + s["Berechtigung"]
	}
	assert.Equal(t, map[string]string{"Empfänger": "kollege.pdf|read", "Freigebender": "angebot.pdf|write"}, byRole)

	uploads := recordMaps(dsarModule(t, person, "Chat-Uploads"))
	require.Len(t, uploads, 2)
	chatDeleted := map[string]string{}
	for _, u := range uploads {
		assert.Equal(t, "projekt-nord", u["Kanal"])
		chatDeleted[u["Dateiname"]] = u["Gelöscht am"]
	}
	assert.Empty(t, chatDeleted["baustelle.png"])
	assert.NotEmpty(t, chatDeleted["geloescht.png"])

	reactions := recordMaps(dsarModule(t, person, "Chat-Reaktionen"))
	require.Len(t, reactions, 1, "only the subject's own reaction")
	assert.Equal(t, "projekt-nord", reactions[0]["Kanal"])

	taskFiles := recordMaps(dsarModule(t, person, "Aufgaben-Dateien"))
	require.Len(t, taskFiles, 1)
	assert.Equal(t, "skizze.png", taskFiles[0]["Dateiname"])
	assert.Equal(t, "Dach decken", taskFiles[0]["Aufgabe"])

	activities := recordMaps(dsarModule(t, person, "Aufgaben-Aktivitäten"))
	require.Len(t, activities, 1, "only the subject's own change")
	assert.Equal(t, "status", activities[0]["Feld"])
	assert.Equal(t, `"done"`, activities[0]["Neuer Wert"])

	articles := recordMaps(dsarModule(t, person, "Wiki-Artikel"))
	require.Len(t, articles, 1)
	assert.Equal(t, "Onboarding-Leitfaden", articles[0]["Titel"])
	attachments := recordMaps(dsarModule(t, person, "Wiki-Anhänge"))
	require.Len(t, attachments, 1)
	assert.Equal(t, "Kollegenartikel", attachments[0]["Artikel"])

	for _, m := range person.Modules {
		for _, rec := range m.Records {
			for _, f := range rec.Fields {
				assert.NotContains(t, f.Value, "ARTICLEBODYTEXT", "wiki body must not be disclosed")
				assert.NotContains(t, f.Value, "k/", "storage keys must not be disclosed")
			}
		}
	}

	// --- tenant isolation -----------------------------------------------------
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Dokumentarin")
	require.NoError(t, err)
	require.Len(t, foreign, 1)
	for module, key := range map[string]string{
		"Eigene Dokumente": "Dateiname", "Dokument-Kommentare": "Kommentar", "Chat-Uploads": "Dateiname",
		"Aufgaben-Dateien": "Dateiname", "Wiki-Artikel": "Titel",
	} {
		recs := recordMaps(dsarModule(t, foreign[0], module))
		require.Len(t, recs, 1, module)
		assert.Contains(t, recs[0][key], "remd", module+": only the other tenant's own row")
	}
	forged, err := SearchByQuery(ctxOther, pool, tenantOther, "Kollege")
	require.NoError(t, err)
	assert.Empty(t, forged)
}
