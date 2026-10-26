package consent

// feat-scrub-email-sender-names-on-contact-anonymize: email_messages.from_name/
// .from_email of the mails linked to an anonymized contact carried the
// contact's display name and address. Body, HTML and headers stay (Handelsbriefe,
// §257 HGB) and are listed under "Deliberately NOT scrubbed" in scrub.go.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAnonymizeContact_ScrubsEmailSenderIdentity(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Consent Email Scrub Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Consent Email Scrub Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenantID) })

	newUser := func(tenant uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenant,
			"email":         fmt.Sprintf("email-scrub-%s@tenant.local", uuid.NewString()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	userID := newUser(tenantID)
	otherUserID := newUser(otherTenantID)

	// One mailbox per tenant; the owner's address is what an outbound mail
	// carries in from_email.
	type mailbox struct{ account, folder uuid.UUID }
	newMailbox := func(tenant, user uuid.UUID, address string) mailbox {
		acc := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
			"tenant_id": tenant, "user_id": user, "email_address": address,
			"imap_host": "imap.invalid", "smtp_host": "smtp.invalid",
			"username": address, "password_encrypted": "x",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", acc) })
		fol := testutil.SeedRow(t, pool, "email_folders", map[string]any{
			"tenant_id": tenant, "account_id": acc, "name": "INBOX", "imap_name": "INBOX",
		})
		return mailbox{account: acc, folder: fol}
	}
	ownAddress := fmt.Sprintf("owner-%s@tenant.local", uuid.NewString()[:8])
	box := newMailbox(tenantID, userID, ownAddress)
	otherBox := newMailbox(otherTenantID, otherUserID, fmt.Sprintf("owner-%s@other.local", uuid.NewString()[:8]))

	uid := int64(0)
	newMail := func(tenant uuid.UUID, mb mailbox, fromName, fromEmail string) uuid.UUID {
		uid++
		id := testutil.SeedRow(t, pool, "email_messages", map[string]any{
			"tenant_id": tenant, "account_id": mb.account, "folder_id": mb.folder, "uid": uid,
			"from_name": fromName, "from_email": fromEmail, "subject": "Angebot 4711",
			"body_text": "Mit freundlichen Grüßen", "date": time.Now().UTC(),
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_messages", id) })
		return id
	}
	link := func(tenant, mail, contact uuid.UUID) {
		testutil.SeedRow(t, pool, "email_contact_links", map[string]any{
			"tenant_id": tenant, "message_id": mail, "contact_id": contact,
		})
	}
	newContact := func(tenant, user uuid.UUID, first string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "contacts", map[string]any{
			"tenant_id": tenant, "first_name": first, "last_name": "MailScrub", "created_by": user,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", id) })
		return id
	}

	contactID := newContact(tenantID, userID, "Mail")
	siblingContactID := newContact(tenantID, userID, "Sibling")
	otherContactID := newContact(otherTenantID, otherUserID, "Foreign")

	inbound := newMail(tenantID, box, "Mail MailScrub", "mail.mailscrub@kunde.example")
	outbound := newMail(tenantID, box, "Owner", ownAddress)
	unlinked := newMail(tenantID, box, "Unlinked Sender", "unlinked@kunde.example")
	sibling := newMail(tenantID, box, "Sibling MailScrub", "sibling.mailscrub@kunde.example")
	foreign := newMail(otherTenantID, otherBox, "Foreign MailScrub", "foreign.mailscrub@kunde.example")
	link(tenantID, inbound, contactID)
	link(tenantID, outbound, contactID) // contact is the recipient of the owner's mail
	link(tenantID, sibling, siblingContactID)
	link(otherTenantID, foreign, otherContactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)
	if err := repo.AnonymizeContact(ctxOwn, contactID, tenantID); err != nil {
		t.Fatalf("AnonymizeContact: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	read := func(id uuid.UUID) (name *string, addr, subject, body string) {
		t.Helper()
		if err := pool.QueryRow(sysCtx,
			"SELECT from_name, from_email, subject, body_text FROM email_messages WHERE id = $1", id,
		).Scan(&name, &addr, &subject, &body); err != nil {
			t.Fatalf("read back mail %s: %v", id, err)
		}
		return name, addr, subject, body
	}

	name, addr, subject, body := read(inbound)
	if name != nil || addr != AnonymizedRequesterEmail {
		t.Fatalf("expected linked inbound mail scrubbed, got from_name=%v from_email=%q", name, addr)
	}
	if subject != "Angebot 4711" || body != "Mit freundlichen Grüßen" {
		t.Fatalf("subject and body are deliberately kept, got %q / %q", subject, body)
	}
	if name, addr, _, _ := read(outbound); name == nil || *name != "Owner" || addr != ownAddress {
		t.Fatalf("the owner's own outbound mail must keep its sender, got from_name=%v from_email=%q", name, addr)
	}
	if name, addr, _, _ := read(unlinked); name == nil || *name != "Unlinked Sender" || addr != "unlinked@kunde.example" {
		t.Fatalf("unlinked mail must survive, got from_name=%v from_email=%q", name, addr)
	}
	if name, addr, _, _ := read(sibling); name == nil || *name != "Sibling MailScrub" || addr != "sibling.mailscrub@kunde.example" {
		t.Fatalf("another contact's mail must survive, got from_name=%v from_email=%q", name, addr)
	}
	if name, addr, _, _ := read(foreign); name == nil || *name != "Foreign MailScrub" || addr != "foreign.mailscrub@kunde.example" {
		t.Fatalf("foreign tenant's mail must survive, got from_name=%v from_email=%q", name, addr)
	}
}
