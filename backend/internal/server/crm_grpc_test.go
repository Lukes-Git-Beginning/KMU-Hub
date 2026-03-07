package server

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/crm/activity"
	"github.com/kmuhub/kmuhub/internal/crm/company"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/crm/customfield"
	"github.com/kmuhub/kmuhub/internal/crm/deal"
	emailcontact "github.com/kmuhub/kmuhub/internal/email/contact"
	"github.com/kmuhub/kmuhub/internal/crm/pipelinestage"
	"github.com/kmuhub/kmuhub/internal/crm/report"
	"github.com/kmuhub/kmuhub/internal/crm/savedfilter"
	"github.com/kmuhub/kmuhub/internal/crm/search"
	"github.com/kmuhub/kmuhub/internal/crm/tag"
)

func TestMapCRMError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		// --- Custom field errors ---
		{"field not found", customfield.ErrFieldNotFound, codes.NotFound},
		{"field exists", customfield.ErrFieldExists, codes.AlreadyExists},
		{"invalid field type", customfield.ErrInvalidFieldType, codes.InvalidArgument},
		{"invalid entity type (cf)", customfield.ErrInvalidEntityType, codes.InvalidArgument},
		{"options required", customfield.ErrOptionsRequired, codes.InvalidArgument},
		{"field name required", customfield.ErrFieldNameRequired, codes.InvalidArgument},
		{"field label required", customfield.ErrFieldLabelRequired, codes.InvalidArgument},
		{"field in use", customfield.ErrFieldInUse, codes.FailedPrecondition},
		// --- Tag errors ---
		{"tag not found", tag.ErrTagNotFound, codes.NotFound},
		{"tag exists", tag.ErrTagExists, codes.AlreadyExists},
		{"tag name required", tag.ErrTagNameRequired, codes.InvalidArgument},
		{"invalid color", tag.ErrInvalidColor, codes.InvalidArgument},
		{"invalid entity type (tag)", tag.ErrInvalidEntityType, codes.InvalidArgument},
		{"tag in use", tag.ErrTagInUse, codes.FailedPrecondition},
		// --- Contact errors ---
		{"contact not found", contact.ErrContactNotFound, codes.NotFound},
		{"email exists", contact.ErrEmailExists, codes.AlreadyExists},
		{"first name required", contact.ErrFirstNameRequired, codes.InvalidArgument},
		{"last name required", contact.ErrLastNameRequired, codes.InvalidArgument},
		{"invalid email", contact.ErrInvalidEmail, codes.InvalidArgument},
		{"company not found (contact)", contact.ErrCompanyNotFound, codes.NotFound},
		{"contact in use", contact.ErrContactInUse, codes.FailedPrecondition},
		{"tag not found (contact)", contact.ErrTagNotFound, codes.NotFound},
		// --- Company errors ---
		{"company not found", company.ErrCompanyNotFound, codes.NotFound},
		{"name required (company)", company.ErrNameRequired, codes.InvalidArgument},
		{"company in use", company.ErrCompanyInUse, codes.FailedPrecondition},
		{"tag not found (company)", company.ErrTagNotFound, codes.NotFound},
		// --- Pipeline stage errors ---
		{"stage not found", pipelinestage.ErrStageNotFound, codes.NotFound},
		{"name required (stage)", pipelinestage.ErrNameRequired, codes.InvalidArgument},
		{"invalid color (stage)", pipelinestage.ErrInvalidColor, codes.InvalidArgument},
		{"invalid probability", pipelinestage.ErrInvalidProbability, codes.InvalidArgument},
		{"stage has deals", pipelinestage.ErrStageHasDeals, codes.FailedPrecondition},
		{"won stage exists", pipelinestage.ErrWonStageExists, codes.AlreadyExists},
		{"lost stage exists", pipelinestage.ErrLostStageExists, codes.AlreadyExists},
		{"invalid reorder", pipelinestage.ErrInvalidReorder, codes.InvalidArgument},
		// --- Deal errors ---
		{"deal not found", deal.ErrDealNotFound, codes.NotFound},
		{"name required (deal)", deal.ErrNameRequired, codes.InvalidArgument},
		{"invalid currency", deal.ErrInvalidCurrency, codes.InvalidArgument},
		{"stage not found (deal)", deal.ErrStageNotFound, codes.NotFound},
		{"contact not found (deal)", deal.ErrContactNotFound, codes.NotFound},
		{"company not found (deal)", deal.ErrCompanyNotFound, codes.NotFound},
		{"owner not found", deal.ErrOwnerNotFound, codes.NotFound},
		{"tag not found (deal)", deal.ErrTagNotFound, codes.NotFound},
		// --- Activity errors ---
		{"activity not found", activity.ErrActivityNotFound, codes.NotFound},
		{"subject required", activity.ErrSubjectRequired, codes.InvalidArgument},
		{"invalid activity type", activity.ErrInvalidActivityType, codes.InvalidArgument},
		{"contact not found (activity)", activity.ErrContactNotFound, codes.NotFound},
		{"company not found (activity)", activity.ErrCompanyNotFound, codes.NotFound},
		{"deal not found (activity)", activity.ErrDealNotFound, codes.NotFound},
		{"user not found (activity)", activity.ErrUserNotFound, codes.NotFound},
		{"tag not found (activity)", activity.ErrTagNotFound, codes.NotFound},
		// --- Search errors ---
		{"query too short", search.ErrQueryTooShort, codes.InvalidArgument},
		{"query too long", search.ErrQueryTooLong, codes.InvalidArgument},
		{"invalid entity", search.ErrInvalidEntity, codes.InvalidArgument},
		// --- Saved filter errors ---
		{"filter not found", savedfilter.ErrFilterNotFound, codes.NotFound},
		{"name required (filter)", savedfilter.ErrNameRequired, codes.InvalidArgument},
		{"invalid entity type (filter)", savedfilter.ErrInvalidEntityType, codes.InvalidArgument},
		{"invalid filter json", savedfilter.ErrInvalidFilterJSON, codes.InvalidArgument},
		{"filter not owned", savedfilter.ErrFilterNotOwned, codes.PermissionDenied},
		// --- Report errors ---
		{"invalid date range", report.ErrInvalidDateRange, codes.InvalidArgument},
		{"start date required", report.ErrStartDateRequired, codes.InvalidArgument},
		{"end date required", report.ErrEndDateRequired, codes.InvalidArgument},
		// --- Visibility errors ---
		{"invalid visibility", emailcontact.ErrInvalidVisibility, codes.InvalidArgument},
		{"not authorized (visibility)", emailcontact.ErrNotAuthorized, codes.PermissionDenied},
		// --- Fallback ---
		{"unknown error", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireGRPCCode(t, mapCRMError(tt.err), tt.wantCode)
		})
	}
}
