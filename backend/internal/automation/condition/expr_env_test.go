package condition

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEnvFromEvent_CRM_PopulatesDealContactCompany(t *testing.T) {
	payload := map[string]any{
		"deal_id":          "deal-1",
		"deal_name":        "Big Deal",
		"deal_value":       12500.5,
		"stage_name":       "negotiation",
		"stage_old":        "qualification",
		"owner_id":         "user-1",
		"contact_id":       "contact-1",
		"contact_name":     "Jane Doe",
		"contact_email":    "jane@example.com",
		"contact_phone":    "+41791234567",
		"contact_tags":     "vip",
		"company_id":       "company-1",
		"company_name":     "Acme AG",
		"company_industry": "manufacturing",
	}

	env := BuildEnvFromEvent("crm.deal.stage_changed", payload)

	require.NotNil(t, env)
	assert.Equal(t, "crm.deal.stage_changed", env.EventType)
	assert.Equal(t, "crm", env.Module)
	assert.False(t, env.Timestamp.IsZero())

	require.NotNil(t, env.Deal)
	assert.Equal(t, "deal-1", env.Deal.ID)
	assert.Equal(t, "Big Deal", env.Deal.Name)
	assert.Equal(t, 12500.5, env.Deal.Value)
	assert.Equal(t, "negotiation", env.Deal.StageName)
	assert.Equal(t, "qualification", env.Deal.StageOld)
	assert.Equal(t, "user-1", env.Deal.OwnerID)

	require.NotNil(t, env.Contact)
	assert.Equal(t, "contact-1", env.Contact.ID)
	assert.Equal(t, "Jane Doe", env.Contact.Name)
	assert.Equal(t, "jane@example.com", env.Contact.Email)
	assert.Equal(t, "+41791234567", env.Contact.Phone)
	assert.Equal(t, "vip", env.Contact.Tags)

	require.NotNil(t, env.Company)
	assert.Equal(t, "company-1", env.Company.ID)
	assert.Equal(t, "Acme AG", env.Company.Name)
	assert.Equal(t, "manufacturing", env.Company.Industry)

	assert.Nil(t, env.Invoice)
	assert.Nil(t, env.Quote)
	assert.Nil(t, env.Leave)
	assert.Nil(t, env.Shift)
	assert.Nil(t, env.Task)
	assert.Nil(t, env.Project)
	assert.Nil(t, env.Email)
}

func TestBuildEnvFromEvent_Biz_PopulatesInvoiceAndQuote(t *testing.T) {
	t.Run("days_overdue as int", func(t *testing.T) {
		payload := map[string]any{
			"invoice_id":     "inv-1",
			"invoice_number": "INV-001",
			"total":          999.99,
			"status":         "overdue",
			"days_overdue":   14,
			"quote_id":       "quote-1",
			"quote_number":   "QUO-001",
		}

		env := BuildEnvFromEvent("biz.invoice.overdue", payload)

		require.NotNil(t, env.Invoice)
		assert.Equal(t, "inv-1", env.Invoice.ID)
		assert.Equal(t, "INV-001", env.Invoice.Number)
		assert.Equal(t, 999.99, env.Invoice.Total)
		assert.Equal(t, "overdue", env.Invoice.Status)
		assert.Equal(t, 14, env.Invoice.DaysOverdue)

		require.NotNil(t, env.Quote)
		assert.Equal(t, "quote-1", env.Quote.ID)
		assert.Equal(t, "QUO-001", env.Quote.Number)

		assert.Nil(t, env.Deal)
		assert.Nil(t, env.Contact)
		assert.Nil(t, env.Company)
		assert.Nil(t, env.Leave)
		assert.Nil(t, env.Shift)
		assert.Nil(t, env.Task)
		assert.Nil(t, env.Project)
		assert.Nil(t, env.Email)
	})

	t.Run("days_overdue as float64 (typical JSON-decoded payload)", func(t *testing.T) {
		payload := map[string]any{
			"days_overdue": float64(30),
		}

		env := BuildEnvFromEvent("biz.invoice.overdue", payload)

		require.NotNil(t, env.Invoice)
		assert.Equal(t, 30, env.Invoice.DaysOverdue)
	})
}

func TestBuildEnvFromEvent_HR_PopulatesLeaveAndShift(t *testing.T) {
	payload := map[string]any{
		"employee_name": "John Smith",
		"leave_type":    "vacation",
		"days":          5.0,
		"employee_id":   "emp-1",
		"duration_h":    8.5,
	}

	env := BuildEnvFromEvent("hr.leave.approved", payload)

	require.NotNil(t, env.Leave)
	assert.Equal(t, "John Smith", env.Leave.EmployeeName)
	assert.Equal(t, "vacation", env.Leave.Type)
	assert.Equal(t, 5.0, env.Leave.Days)

	require.NotNil(t, env.Shift)
	assert.Equal(t, "emp-1", env.Shift.EmployeeID)
	assert.Equal(t, 8.5, env.Shift.DurationH)

	assert.Nil(t, env.Deal)
	assert.Nil(t, env.Invoice)
	assert.Nil(t, env.Task)
	assert.Nil(t, env.Email)
}

func TestBuildEnvFromEvent_Work_PopulatesTaskAndProject(t *testing.T) {
	payload := map[string]any{
		"task_id":      "task-1",
		"task_title":   "Fix bug",
		"status":       "done",
		"status_old":   "in_progress",
		"priority":     "high",
		"project_id":   "proj-1",
		"project_name": "Website Relaunch",
		"project_key":  "WEB",
	}

	env := BuildEnvFromEvent("work.task.status_changed", payload)

	require.NotNil(t, env.Task)
	assert.Equal(t, "task-1", env.Task.ID)
	assert.Equal(t, "Fix bug", env.Task.Title)
	assert.Equal(t, "done", env.Task.Status)
	assert.Equal(t, "in_progress", env.Task.StatusOld)
	assert.Equal(t, "high", env.Task.Priority)
	assert.Equal(t, "proj-1", env.Task.ProjectID)

	require.NotNil(t, env.Project)
	assert.Equal(t, "proj-1", env.Project.ID)
	assert.Equal(t, "Website Relaunch", env.Project.Name)
	assert.Equal(t, "WEB", env.Project.Key)
	assert.Equal(t, "done", env.Project.Status)

	assert.Nil(t, env.Deal)
	assert.Nil(t, env.Invoice)
	assert.Nil(t, env.Leave)
	assert.Nil(t, env.Email)
}

func TestBuildEnvFromEvent_Email_PopulatesEmail(t *testing.T) {
	payload := map[string]any{
		"email_id": "email-1",
		"from":     "sender@example.com",
		"to":       "receiver@example.com",
		"subject":  "Hello",
		"has_crm":  true,
	}

	env := BuildEnvFromEvent("email.message.received", payload)

	require.NotNil(t, env.Email)
	assert.Equal(t, "email-1", env.Email.ID)
	assert.Equal(t, "sender@example.com", env.Email.From)
	assert.Equal(t, "receiver@example.com", env.Email.To)
	assert.Equal(t, "Hello", env.Email.Subject)
	assert.True(t, env.Email.HasCRM)

	assert.Nil(t, env.Deal)
	assert.Nil(t, env.Invoice)
	assert.Nil(t, env.Task)
}

func TestBuildEnvFromEvent_UnknownModule_LeavesAllEntitiesNil(t *testing.T) {
	payload := map[string]any{"foo": "bar"}

	env := BuildEnvFromEvent("foo.bar.baz", payload)

	require.NotNil(t, env)
	assert.Equal(t, "foo.bar.baz", env.EventType)
	assert.Equal(t, "foo", env.Module)
	assert.False(t, env.Timestamp.IsZero())

	assert.Nil(t, env.Deal)
	assert.Nil(t, env.Contact)
	assert.Nil(t, env.Company)
	assert.Nil(t, env.Invoice)
	assert.Nil(t, env.Quote)
	assert.Nil(t, env.Leave)
	assert.Nil(t, env.Shift)
	assert.Nil(t, env.Task)
	assert.Nil(t, env.Project)
	assert.Nil(t, env.Email)
}

func TestBuildEnvFromEvent_TriggerTypeWithoutDot_ModuleIsWholeString(t *testing.T) {
	env := BuildEnvFromEvent("standalone", map[string]any{})

	assert.Equal(t, "standalone", env.Module)
	assert.Equal(t, "standalone", env.EventType)
}

func TestBuildEnvFromEvent_NilPayload_ReturnsBareEnv(t *testing.T) {
	env := BuildEnvFromEvent("crm.deal.stage_changed", nil)

	require.NotNil(t, env)
	assert.Equal(t, "crm.deal.stage_changed", env.EventType)
	assert.Equal(t, "crm", env.Module)
	assert.False(t, env.Timestamp.IsZero())

	assert.Nil(t, env.Deal)
	assert.Nil(t, env.Contact)
	assert.Nil(t, env.Company)
	assert.Nil(t, env.Invoice)
	assert.Nil(t, env.Quote)
	assert.Nil(t, env.Leave)
	assert.Nil(t, env.Shift)
	assert.Nil(t, env.Task)
	assert.Nil(t, env.Project)
	assert.Nil(t, env.Email)

	require.NotNil(t, env.Prev)
	assert.Empty(t, env.Prev)
}

func TestBuildEnvFromEvent_MissingKeys_UseZeroValueDefaults(t *testing.T) {
	// Empty payload map (not nil) exercises the "key not found" branch of every
	// get* helper: getString -> "", getFloat64 -> 0, getInt -> 0, getBool -> false.
	env := BuildEnvFromEvent("crm.deal.stage_changed", map[string]any{})

	require.NotNil(t, env.Deal)
	assert.Equal(t, "", env.Deal.ID)
	assert.Equal(t, 0.0, env.Deal.Value)

	env2 := BuildEnvFromEvent("biz.invoice.overdue", map[string]any{})
	require.NotNil(t, env2.Invoice)
	assert.Equal(t, 0, env2.Invoice.DaysOverdue)

	env3 := BuildEnvFromEvent("email.message.received", map[string]any{})
	require.NotNil(t, env3.Email)
	assert.False(t, env3.Email.HasCRM)
}
