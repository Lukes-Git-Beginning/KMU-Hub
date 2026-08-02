package server

// Document chains (Belegkette) — gRPC surface for the read-only lifecycle
// overview (quote -> invoice -> payment/dunning/credit note).

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ListDocumentChains returns every document chain of the tenant.
func (s *BizGRPCServer) ListDocumentChains(ctx context.Context, req *bizv1.ListDocumentChainsRequest) (*bizv1.ListDocumentChainsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	chains, err := s.invoiceService.ListDocumentChains(ctx, tenantID)
	if err != nil {
		return nil, mapBizError(err)
	}

	out := make([]*bizv1.DocumentChain, 0, len(chains))
	for _, c := range chains {
		out = append(out, toProtoDocumentChain(c))
	}
	return &bizv1.ListDocumentChainsResponse{Chains: out}, nil
}

func toProtoDocumentChain(c *models.DocumentChain) *bizv1.DocumentChain {
	nodes := make([]*bizv1.ChainNode, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		nodes = append(nodes, toProtoChainNode(n))
	}
	return &bizv1.DocumentChain{
		Id:         c.ID.String(),
		Customer:   c.Customer,
		Currency:   c.Currency,
		TotalValue: c.TotalValue.String(),
		IsComplete: c.IsComplete,
		Nodes:      nodes,
	}
}

func toProtoChainNode(n models.ChainNode) *bizv1.ChainNode {
	out := &bizv1.ChainNode{
		Type:   n.Type,
		Amount: n.Amount.String(),
		Status: n.Status,
	}
	if n.Number != "" {
		number := n.Number
		out.Number = &number
	}
	if n.Date != nil {
		date := n.Date.Format("2006-01-02")
		out.Date = &date
	}
	return out
}
