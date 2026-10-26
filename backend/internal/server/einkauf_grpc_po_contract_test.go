package server

// framework_contract_id over the einkauf gRPC hop: parsed on create and update
// (empty string clears), echoed on the PurchaseOrder, and a currency mismatch
// maps to InvalidArgument (400), a foreign contract to NotFound.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/einkauf"
	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

func TestEinkauf_PO_FrameworkContractID(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)
	eur := addStubContract(repo, tenantID, supplier.ID, "RV-EUR", einkauf.ContractStatusActive)
	chf := addStubContract(repo, tenantID, supplier.ID, "RV-CHF", einkauf.ContractStatusActive)
	chf.Currency = "CHF"
	foreign := addStubContract(repo, uuid.New(), supplier.ID, "RV-FREMD", einkauf.ContractStatusActive)

	create := func(number string, contractID *string) (*einkaufv1.POResponse, error) {
		return srv.CreatePO(ctx, &einkaufv1.CreatePORequest{
			TenantId: tenantID.String(), SupplierId: supplier.ID.String(), PoNumber: number,
			Currency: "EUR", FrameworkContractId: contractID,
		})
	}

	created, err := create("PO-FC-1", strPtr(eur.ID.String()))
	require.NoError(t, err)
	assert.Equal(t, eur.ID.String(), created.Po.GetFrameworkContractId(), "the binding must be echoed")

	unbound, err := create("PO-FC-2", strPtr(""))
	require.NoError(t, err)
	assert.Nil(t, unbound.Po.FrameworkContractId, "an empty id creates an unbound PO")

	_, err = create("PO-FC-3", strPtr("kein-uuid"))
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = create("PO-FC-4", strPtr(chf.ID.String()))
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = create("PO-FC-5", strPtr(foreign.ID.String()))
	requireGRPCCode(t, err, codes.NotFound)

	poID := created.Po.Id
	update := func(contractID string) (*einkaufv1.POResponse, error) {
		return srv.UpdatePO(ctx, &einkaufv1.UpdatePORequest{TenantId: tenantID.String(), PoId: poID, FrameworkContractId: &contractID})
	}
	_, err = update("kein-uuid")
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = update(chf.ID.String())
	requireGRPCCode(t, err, codes.InvalidArgument)

	cleared, err := update("")
	require.NoError(t, err)
	assert.Nil(t, cleared.Po.FrameworkContractId, "an empty string clears the binding")

	notes := "nur Notiz"
	untouched, err := srv.UpdatePO(ctx, &einkaufv1.UpdatePORequest{TenantId: tenantID.String(), PoId: unbound.Po.Id, Notes: &notes})
	require.NoError(t, err)
	assert.Nil(t, untouched.Po.FrameworkContractId, "an omitted id leaves the binding alone")
}
