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

func TestSearchByQuery_UserInboxAndMailAccounts_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Inbox Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Inbox Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Ingrid", "Postfachinhaberin", true)
	colleagueID := seedDSARUser(t, pool, tenantOwn, "Karl", "Kollege", true)
	// A different tenant's user of the same name must not be reachable.
	foreignUserID := seedDSARUser(t, pool, tenantOther, "Ingrid", "Postfachinhaberin", true)

	defer testutil.CleanupRow(t, pool, "users", subjectID)
	defer testutil.CleanupRow(t, pool, "users", colleagueID)
	defer testutil.CleanupRow(t, pool, "users", foreignUserID)

	received := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	ownedID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":    tenantOwn,
		"user_id":      subjectID,
		"channel":      "email",
		"source_id":    "src-" + uuid.NewString(),
		"sender_name":  "Externer Absender",
		"sender_email": "extern@absender.invalid",
		"subject":      "Angebot Nr. 4711",
		"preview":      "Bitte um Rückruf wegen des Angebots",
		"received_at":  received,
	})
	// Someone else's mailbox, but assigned to the subject.
	assignedID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":   tenantOwn,
		"user_id":     colleagueID,
		"assigned_to": subjectID,
		"channel":     "chat",
		"source_id":   "src-" + uuid.NewString(),
		"sender_name": "Kunde Zwei",
		"subject":     "Zugewiesene Anfrage",
		"preview":     "Hallo, können Sie helfen?",
		"received_at": received.Add(-time.Hour),
	})
	// The colleague's own message must never show up for the subject.
	colleagueOnlyID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":   tenantOwn,
		"user_id":     colleagueID,
		"channel":     "email",
		"source_id":   "src-" + uuid.NewString(),
		"sender_name": "Fremder Absender",
		"subject":     "Nur für Karl",
		"preview":     "geheim",
		"received_at": received,
	})
	// Same user id in another tenant's inbox: tenant filter must hide it.
	foreignMsgID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":   tenantOther,
		"user_id":     foreignUserID,
		"channel":     "email",
		"source_id":   "src-" + uuid.NewString(),
		"sender_name": "Anderer Tenant",
		"subject":     "Fremder Tenant",
		"preview":     "darf nie erscheinen",
		"received_at": received,
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", ownedID)
	defer testutil.CleanupRow(t, pool, "inbox_messages", assignedID)
	defer testutil.CleanupRow(t, pool, "inbox_messages", colleagueOnlyID)
	defer testutil.CleanupRow(t, pool, "inbox_messages", foreignMsgID)

	// Reply by the subject and an inbound line by the external party.
	replyID := testutil.SeedRow(t, pool, "inbox_thread_messages", map[string]any{
		"tenant_id": tenantOwn, "message_id": ownedID, "author_id": subjectID,
		"direction": "outbound", "body": "Ich melde mich morgen bei Ihnen.",
	})
	inboundID := testutil.SeedRow(t, pool, "inbox_thread_messages", map[string]any{
		"tenant_id": tenantOwn, "message_id": ownedID, "author_name": "Externer Absender",
		"direction": "inbound", "body": "Fremder Text des Absenders",
	})
	defer testutil.CleanupRow(t, pool, "inbox_thread_messages", replyID)
	defer testutil.CleanupRow(t, pool, "inbox_thread_messages", inboundID)

	teamInboxID := testutil.SeedRow(t, pool, "team_inboxes", map[string]any{
		"tenant_id": tenantOwn, "name": "Vertrieb Postfach", "created_by": colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "team_inboxes", teamInboxID)
	_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO team_inbox_members (team_inbox_id, user_id, role, tenant_id) VALUES ($1, $2, 'admin', $3)`,
		teamInboxID, subjectID, tenantOwn)
	require.NoError(t, err)

	accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"tenant_id":          tenantOwn,
		"user_id":            subjectID,
		"email_address":      "ingrid@firma.invalid",
		"display_name":       "Ingrid P.",
		"imap_host":          "imap.firma.invalid",
		"smtp_host":          "smtp.firma.invalid",
		"username":           "ingrid-login",
		"password_encrypted": "SUPER-SECRET-CIPHERTEXT",
	})
	defer testutil.CleanupRow(t, pool, "email_accounts", accountID)
	signatureID := testutil.SeedRow(t, pool, "email_signatures", map[string]any{
		"tenant_id": tenantOwn, "user_id": subjectID, "name": "Standard",
		"html_content": "<p>Ingrid P., Tel. 0170 1234567</p>",
	})
	defer testutil.CleanupRow(t, pool, "email_signatures", signatureID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Postfachinhaberin")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	person := persons[0]

	inbox := dsarModule(t, person, "Posteingang")
	roles := map[string]string{}
	for _, r := range recordMaps(inbox) {
		roles[r["Betreff"]] = r["Rolle"]
	}
	assert.Equal(t, map[string]string{
		"Angebot Nr. 4711":    "Postfach-Inhaber",
		"Zugewiesene Anfrage": "Zugewiesen",
	}, roles, "owned and assigned entries only, never the colleague's own mail")
	for _, r := range recordMaps(inbox) {
		if r["Betreff"] == "Angebot Nr. 4711" {
			assert.Equal(t, "extern@absender.invalid", r["Absender-E-Mail"])
			assert.Equal(t, "Bitte um Rückruf wegen des Angebots", r["Vorschau"])
			assert.NotEmpty(t, r["Empfangen"])
		}
	}

	replies := recordMaps(dsarModule(t, person, "Posteingang-Antworten"))
	require.Len(t, replies, 1, "only the subject's own reply, not the inbound line")
	assert.Equal(t, "Ich melde mich morgen bei Ihnen.", replies[0]["Text"])
	assert.Equal(t, "Angebot Nr. 4711", replies[0]["Betreff"])

	teams := recordMaps(dsarModule(t, person, "Team-Posteingänge"))
	require.Len(t, teams, 1)
	assert.Equal(t, "Vertrieb Postfach", teams[0]["Team-Posteingang"])
	assert.Equal(t, "admin", teams[0]["Rolle"])

	accounts := dsarModule(t, person, "E-Mail-Konten")
	require.Len(t, accounts.Records, 1)
	acc := recordMaps(accounts)[0]
	assert.Equal(t, "ingrid@firma.invalid", acc["E-Mail-Adresse"])
	assert.Equal(t, "ingrid-login", acc["Benutzername"])
	assert.Equal(t, "imap.firma.invalid", acc["IMAP-Server"])
	for _, rec := range accounts.Records {
		for _, f := range rec.Fields {
			assert.NotContains(t, f.Value, "SUPER-SECRET", "stored credentials must never be disclosed")
		}
	}

	sigs := recordMaps(dsarModule(t, person, "E-Mail-Signaturen"))
	require.Len(t, sigs, 1)
	assert.Contains(t, sigs[0]["Inhalt"], "0170 1234567")

	// --- tenant isolation ---------------------------------------------------
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Postfachinhaberin")
	require.NoError(t, err)
	require.Len(t, foreign, 1, "the other tenant's own user of the same name")
	assert.NotContains(t, moduleNames(foreign[0]), "E-Mail-Konten")
	assert.NotContains(t, moduleNames(foreign[0]), "E-Mail-Signaturen")
	assert.NotContains(t, moduleNames(foreign[0]), "Team-Posteingänge")
	for _, r := range recordMaps(dsarModule(t, foreign[0], "Posteingang")) {
		assert.Equal(t, "Fremder Tenant", r["Betreff"], "only the other tenant's own entry")
	}
}
