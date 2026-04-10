// Hand-written proto message types for the CreateQuoteFromDeal RPC.
// These are intentionally simplified (no protoimpl reflection machinery) matching
// the established pattern in this package (see biz_time.pb.go).
// Run `make proto-biz` after updating biz.proto to regenerate properly.

package bizv1

// CreateQuoteFromDealRequest carries the parameters needed to look up a CRM deal
// and create a pre-populated draft quote from it.
type CreateQuoteFromDealRequest struct {
	TenantId  string `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	DealId    string `protobuf:"bytes,2,opt,name=deal_id,json=dealId,proto3" json:"deal_id,omitempty"`
	CreatedBy string `protobuf:"bytes,3,opt,name=created_by,json=createdBy,proto3" json:"created_by,omitempty"`
}

func (x *CreateQuoteFromDealRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *CreateQuoteFromDealRequest) GetDealId() string {
	if x != nil {
		return x.DealId
	}
	return ""
}

func (x *CreateQuoteFromDealRequest) GetCreatedBy() string {
	if x != nil {
		return x.CreatedBy
	}
	return ""
}

// CreateQuoteFromDealResponse carries the newly created draft quote.
type CreateQuoteFromDealResponse struct {
	Quote *Quote `protobuf:"bytes,1,opt,name=quote,proto3" json:"quote,omitempty"`
}

func (x *CreateQuoteFromDealResponse) GetQuote() *Quote {
	if x != nil {
		return x.Quote
	}
	return nil
}
