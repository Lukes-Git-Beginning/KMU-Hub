package server

// Open items (Offene Posten) — gRPC surface for the receivables list.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/biz/dunning"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ListOpenItems returns the tenant's unpaid receivables with their aging.
func (s *BizGRPCServer) ListOpenItems(
	ctx context.Context,
	req *bizv1.ListOpenItemsRequest,
) (*bizv1.ListOpenItemsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	limit, offset := pageToLimitOffset(req.GetPage(), req.GetPerPage())

	page, err := s.dunningService.ListOpenItems(ctx, tenantID, dunning.ListOpenItemsInput{
		Bucket:      req.GetBucket(),
		OverdueOnly: req.GetOverdueOnly(),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, mapBizError(err)
	}

	items := make([]*bizv1.OpenItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toProtoOpenItem(item))
	}

	return &bizv1.ListOpenItemsResponse{
		Items:   items,
		Total:   int32(page.Total),
		Summary: toProtoOpenItemsSummary(page.Summary),
	}, nil
}

func toProtoOpenItem(item *models.OpenItem) *bizv1.OpenItem {
	if item == nil {
		return nil
	}
	out := &bizv1.OpenItem{
		InvoiceId:     item.InvoiceID.String(),
		InvoiceNumber: item.InvoiceNumber,
		Status:        item.Status,
		CustomerName:  item.CustomerName,
		CustomerEmail: item.CustomerEmail,
		Currency:      item.Currency,
		GrossTotal:    item.GrossTotal.String(),
		PaidAmount:    item.PaidAmount.String(),
		OpenAmount:    item.OpenAmount.String(),
		InvoiceDate:   item.InvoiceDate.Format("2006-01-02"),
		DueDate:       item.DueDate.Format("2006-01-02"),
		DaysOverdue:   int32(item.DaysOverdue),
		AgingBucket:   item.AgingBucket,
		DunningLevel:  int32(item.DunningLevel),
		DunningStatus: item.DunningStatus,
	}
	if item.LastDunnedAt != nil {
		sentAt := item.LastDunnedAt.UTC().Format(time.RFC3339)
		out.LastDunnedAt = &sentAt
	}
	if item.ContactID != nil {
		contactID := item.ContactID.String()
		out.ContactId = &contactID
	}
	return out
}

func toProtoOpenItemsSummary(summary *models.OpenItemsSummary) *bizv1.OpenItemsSummary {
	if summary == nil {
		return &bizv1.OpenItemsSummary{}
	}

	totals := make([]*bizv1.OpenItemCurrencyTotal, 0, len(summary.Totals))
	for _, total := range summary.Totals {
		if total == nil {
			continue
		}
		totals = append(totals, &bizv1.OpenItemCurrencyTotal{
			Currency:       total.Currency,
			OpenAmount:     total.OpenAmount.String(),
			OpenCount:      int32(total.OpenCount),
			OverdueAmount:  total.OverdueAmount.String(),
			OverdueCount:   int32(total.OverdueCount),
			AvgDaysOverdue: int32(total.AvgDaysOverdue),
		})
	}

	buckets := make([]*bizv1.OpenItemBucketTotal, 0, len(summary.Buckets))
	for _, bucket := range summary.Buckets {
		if bucket == nil {
			continue
		}
		buckets = append(buckets, &bizv1.OpenItemBucketTotal{
			Currency: bucket.Currency,
			Bucket:   bucket.Bucket,
			Count:    int32(bucket.Count),
			Amount:   bucket.Amount.String(),
		})
	}

	return &bizv1.OpenItemsSummary{Totals: totals, Buckets: buckets}
}
