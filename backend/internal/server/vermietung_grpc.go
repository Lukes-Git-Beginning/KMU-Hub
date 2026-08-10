package server

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/vermietung"
	vermietungv1 "github.com/kmuhub/kmuhub/proto/vermietung/v1"
)

// VermietungGRPCServer implements the VermietungService gRPC server.
type VermietungGRPCServer struct {
	vermietungv1.UnimplementedVermietungServiceServer
	svc *vermietung.Service
}

// NewVermietungGRPCServer creates a new Vermietung gRPC server.
func NewVermietungGRPCServer(svc *vermietung.Service) *VermietungGRPCServer {
	return &VermietungGRPCServer{svc: svc}
}

// ============================================================================
// Object RPCs
// ============================================================================

func (s *VermietungGRPCServer) CreateObject(ctx context.Context, req *vermietungv1.CreateObjectRequest) (*vermietungv1.ObjectResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	obj, err := s.svc.CreateObject(ctx, vermietung.CreateObjectInput{
		TenantID:    tenantID,
		Name:        req.GetName(),
		Description: req.Description,
		Category:    req.GetCategory(),
		DailyRate:   req.GetDailyRate(),
		Deposit:     req.GetDeposit(),
		Location:    req.Location,
		Notes:       req.Notes,
	})
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.ObjectResponse{Object: objectToProto(obj)}, nil
}

func (s *VermietungGRPCServer) UpdateObject(ctx context.Context, req *vermietungv1.UpdateObjectRequest) (*vermietungv1.ObjectResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	objectID, err := uuid.Parse(req.GetObjectId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid object_id: %v", err)
	}

	input := vermietung.UpdateObjectInput{
		TenantID:    tenantID,
		ObjectID:    objectID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Location:    req.Location,
		Notes:       req.Notes,
		Active:      req.Active,
	}
	if req.DailyRate != nil {
		v := req.GetDailyRate()
		input.DailyRate = &v
	}
	if req.Deposit != nil {
		v := req.GetDeposit()
		input.Deposit = &v
	}

	obj, err := s.svc.UpdateObject(ctx, input)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.ObjectResponse{Object: objectToProto(obj)}, nil
}

func (s *VermietungGRPCServer) DeleteObject(ctx context.Context, req *vermietungv1.DeleteObjectRequest) (*vermietungv1.DeleteObjectResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	objectID, err := uuid.Parse(req.GetObjectId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid object_id: %v", err)
	}

	if err := s.svc.DeleteObject(ctx, tenantID, objectID); err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.DeleteObjectResponse{}, nil
}

func (s *VermietungGRPCServer) GetObject(ctx context.Context, req *vermietungv1.GetObjectRequest) (*vermietungv1.ObjectResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	objectID, err := uuid.Parse(req.GetObjectId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid object_id: %v", err)
	}

	obj, err := s.svc.GetObject(ctx, tenantID, objectID)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.ObjectResponse{Object: objectToProto(obj)}, nil
}

func (s *VermietungGRPCServer) ListObjects(ctx context.Context, req *vermietungv1.ListObjectsRequest) (*vermietungv1.ListObjectsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	activeOnly := false
	if req.ActiveOnly != nil {
		activeOnly = req.GetActiveOnly()
	}

	objects, total, err := s.svc.ListObjects(ctx, vermietung.ListObjectsInput{
		TenantID:   tenantID,
		Search:     req.GetSearch(),
		Category:   req.Category,
		ActiveOnly: activeOnly,
		Page:       int(req.GetPage()),
		PageSize:   int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapVermietungError(err)
	}

	protoObjs := make([]*vermietungv1.RentalObject, len(objects))
	for i, obj := range objects {
		protoObjs[i] = objectToProto(obj)
	}
	return &vermietungv1.ListObjectsResponse{Objects: protoObjs, Total: int32(total)}, nil
}

// ============================================================================
// Availability RPC
// ============================================================================

func (s *VermietungGRPCServer) CheckAvailability(ctx context.Context, req *vermietungv1.CheckAvailabilityRequest) (*vermietungv1.CheckAvailabilityResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	objectID, err := uuid.Parse(req.GetObjectId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid object_id: %v", err)
	}

	input := vermietung.CheckAvailabilityInput{
		TenantID:  tenantID,
		ObjectID:  objectID,
		StartDate: req.GetStartDate().AsTime(),
		EndDate:   req.GetEndDate().AsTime(),
	}
	if req.ExcludeRentalId != nil {
		id, parseErr := uuid.Parse(*req.ExcludeRentalId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid exclude_rental_id: %v", parseErr)
		}
		input.ExcludeRentalID = &id
	}

	result, err := s.svc.CheckAvailability(ctx, input)
	if err != nil {
		return nil, mapVermietungError(err)
	}

	conflicts := make([]*vermietungv1.Rental, len(result.ConflictingRentals))
	for i, r := range result.ConflictingRentals {
		conflicts[i] = rentalToProto(r)
	}
	return &vermietungv1.CheckAvailabilityResponse{
		Available:          result.Available,
		ConflictingRentals: conflicts,
	}, nil
}

// ============================================================================
// Rental RPCs
// ============================================================================

func (s *VermietungGRPCServer) CreateRental(ctx context.Context, req *vermietungv1.CreateRentalRequest) (*vermietungv1.RentalResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	objectID, err := uuid.Parse(req.GetObjectId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid object_id: %v", err)
	}

	input := vermietung.CreateRentalInput{
		TenantID:    tenantID,
		ObjectID:    objectID,
		RenterName:  req.GetRenterName(),
		StartDate:   req.GetStartDate().AsTime(),
		EndDate:     req.GetEndDate().AsTime(),
		TotalPrice:  req.GetTotalPrice(),
		DepositPaid: req.GetDepositPaid(),
		Notes:       req.Notes,
	}
	if req.ContactId != nil {
		id, parseErr := uuid.Parse(*req.ContactId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", parseErr)
		}
		input.ContactID = &id
	}

	rental, err := s.svc.CreateRental(ctx, input)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.RentalResponse{Rental: rentalToProto(rental)}, nil
}

func (s *VermietungGRPCServer) UpdateRental(ctx context.Context, req *vermietungv1.UpdateRentalRequest) (*vermietungv1.RentalResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	rentalID, err := uuid.Parse(req.GetRentalId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rental_id: %v", err)
	}

	input := vermietung.UpdateRentalInput{
		TenantID:   tenantID,
		RentalID:   rentalID,
		RenterName: req.RenterName,
		Notes:      req.Notes,
	}
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		input.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		input.EndDate = &t
	}
	if req.TotalPrice != nil {
		v := req.GetTotalPrice()
		input.TotalPrice = &v
	}
	if req.DepositPaid != nil {
		v := req.GetDepositPaid()
		input.DepositPaid = &v
	}

	rental, err := s.svc.UpdateRental(ctx, input)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.RentalResponse{Rental: rentalToProto(rental)}, nil
}

func (s *VermietungGRPCServer) DeleteRental(ctx context.Context, req *vermietungv1.DeleteRentalRequest) (*vermietungv1.DeleteRentalResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	rentalID, err := uuid.Parse(req.GetRentalId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rental_id: %v", err)
	}

	if err := s.svc.DeleteRental(ctx, tenantID, rentalID); err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.DeleteRentalResponse{}, nil
}

func (s *VermietungGRPCServer) GetRental(ctx context.Context, req *vermietungv1.GetRentalRequest) (*vermietungv1.RentalResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	rentalID, err := uuid.Parse(req.GetRentalId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rental_id: %v", err)
	}

	rental, err := s.svc.GetRental(ctx, tenantID, rentalID)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.RentalResponse{Rental: rentalToProto(rental)}, nil
}

func (s *VermietungGRPCServer) ListRentals(ctx context.Context, req *vermietungv1.ListRentalsRequest) (*vermietungv1.ListRentalsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := vermietung.ListRentalsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.ObjectId != nil {
		id, parseErr := uuid.Parse(*req.ObjectId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid object_id: %v", parseErr)
		}
		input.ObjectID = &id
	}
	if req.Status != nil {
		st := vermietung.RentalStatus(*req.Status)
		input.Status = &st
	}
	if req.From != nil {
		t := req.From.AsTime()
		input.From = &t
	}
	if req.To != nil {
		t := req.To.AsTime()
		input.To = &t
	}

	rentals, total, err := s.svc.ListRentals(ctx, input)
	if err != nil {
		return nil, mapVermietungError(err)
	}

	protoRentals := make([]*vermietungv1.Rental, len(rentals))
	for i, r := range rentals {
		protoRentals[i] = rentalToProto(r)
	}
	return &vermietungv1.ListRentalsResponse{Rentals: protoRentals, Total: int32(total)}, nil
}

func (s *VermietungGRPCServer) StartRental(ctx context.Context, req *vermietungv1.StartRentalRequest) (*vermietungv1.RentalResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	rentalID, err := uuid.Parse(req.GetRentalId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rental_id: %v", err)
	}

	rental, err := s.svc.StartRental(ctx, tenantID, rentalID)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.RentalResponse{Rental: rentalToProto(rental)}, nil
}

func (s *VermietungGRPCServer) EndRental(ctx context.Context, req *vermietungv1.EndRentalRequest) (*vermietungv1.RentalResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	rentalID, err := uuid.Parse(req.GetRentalId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rental_id: %v", err)
	}

	rental, err := s.svc.EndRental(ctx, tenantID, rentalID)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.RentalResponse{Rental: rentalToProto(rental)}, nil
}

// ============================================================================
// Inspection RPCs
// ============================================================================

func (s *VermietungGRPCServer) CreateInspection(ctx context.Context, req *vermietungv1.CreateInspectionRequest) (*vermietungv1.InspectionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	rentalID, err := uuid.Parse(req.GetRentalId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rental_id: %v", err)
	}

	input := vermietung.CreateInspectionInput{
		TenantID:  tenantID,
		RentalID:  rentalID,
		Kind:      vermietung.InspectionKind(req.GetKind()),
		Notes:     req.GetNotes(),
		PhotoURLs: req.GetPhotoUrls(),
		Checklist: checklistFromProto(req.GetChecklist()),
	}
	if req.PerformedBy != nil {
		id, parseErr := uuid.Parse(*req.PerformedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid performed_by: %v", parseErr)
		}
		input.PerformedBy = &id
	}

	ins, err := s.svc.CreateInspection(ctx, input)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.InspectionResponse{Inspection: inspectionToProto(ins)}, nil
}

func (s *VermietungGRPCServer) UpdateInspection(ctx context.Context, req *vermietungv1.UpdateInspectionRequest) (*vermietungv1.InspectionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	inspectionID, err := uuid.Parse(req.GetInspectionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid inspection_id: %v", err)
	}

	input := vermietung.UpdateInspectionInput{
		TenantID:     tenantID,
		InspectionID: inspectionID,
		Notes:        req.Notes,
		// Bug #17: use explicit replace_photos flag so callers can clear the photo list
		// by sending replace_photos=true with an empty photo_urls array.
		ReplacePhotos: req.GetReplacePhotos(),
		PhotoURLs:     req.GetPhotoUrls(),
		// Same convention for checklist: replace_checklist distinguishes "not
		// provided" from "provided empty" the way a bare repeated field can't.
		ReplaceChecklist: req.GetReplaceChecklist(),
		Checklist:        checklistFromProto(req.GetChecklist()),
		SignatureData:    req.SignatureData,
	}

	ins, err := s.svc.UpdateInspection(ctx, input)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.InspectionResponse{Inspection: inspectionToProto(ins)}, nil
}

func (s *VermietungGRPCServer) GetInspection(ctx context.Context, req *vermietungv1.GetInspectionRequest) (*vermietungv1.InspectionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	inspectionID, err := uuid.Parse(req.GetInspectionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid inspection_id: %v", err)
	}

	ins, err := s.svc.GetInspection(ctx, tenantID, inspectionID)
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.InspectionResponse{Inspection: inspectionToProto(ins)}, nil
}

func (s *VermietungGRPCServer) ListInspections(ctx context.Context, req *vermietungv1.ListInspectionsRequest) (*vermietungv1.ListInspectionsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	rentalID, err := uuid.Parse(req.GetRentalId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rental_id: %v", err)
	}

	inspections, total, err := s.svc.ListInspections(ctx, vermietung.ListInspectionsInput{
		TenantID: tenantID,
		RentalID: rentalID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapVermietungError(err)
	}

	protoInspections := make([]*vermietungv1.RentalInspection, len(inspections))
	for i, ins := range inspections {
		protoInspections[i] = inspectionToProto(ins)
	}
	return &vermietungv1.ListInspectionsResponse{Inspections: protoInspections, Total: int32(total)}, nil
}

// ============================================================================
// Signature RPC
// ============================================================================

func (s *VermietungGRPCServer) SaveSignature(ctx context.Context, req *vermietungv1.SaveRentalSignatureRequest) (*vermietungv1.RentalResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	rentalID, err := uuid.Parse(req.GetRentalId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rental_id: %v", err)
	}

	rental, err := s.svc.SaveSignature(ctx, tenantID.String(), rentalID.String(), req.GetSignatureData(), req.GetSignedBy())
	if err != nil {
		return nil, mapVermietungError(err)
	}
	return &vermietungv1.RentalResponse{Rental: rentalToProto(rental)}, nil
}

// ============================================================================
// Calendar & Report RPCs
// ============================================================================

func (s *VermietungGRPCServer) GetRentalCalendar(ctx context.Context, req *vermietungv1.GetRentalCalendarRequest) (*vermietungv1.GetRentalCalendarResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	year := int(req.GetYear())
	month := int(req.GetMonth())
	if year == 0 {
		year = time.Now().Year()
	}
	if month < 1 || month > 12 {
		month = int(time.Now().Month())
	}

	entries, err := s.svc.GetRentalCalendar(ctx, tenantID, year, month)
	if err != nil {
		return nil, mapVermietungError(err)
	}

	protoEntries := make([]*vermietungv1.CalendarEntry, len(entries))
	for i, e := range entries {
		protoRentals := make([]*vermietungv1.Rental, len(e.Rentals))
		for j, r := range e.Rentals {
			protoRentals[j] = rentalToProto(r)
		}
		protoEntries[i] = &vermietungv1.CalendarEntry{
			ObjectId:   e.Object.ID.String(),
			ObjectName: e.Object.Name,
			Rentals:    protoRentals,
		}
	}
	return &vermietungv1.GetRentalCalendarResponse{Entries: protoEntries}, nil
}

func (s *VermietungGRPCServer) ExportRentalReport(ctx context.Context, req *vermietungv1.ExportRentalReportRequest) (*vermietungv1.ExportRentalReportResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := vermietung.ListRentalsInput{
		TenantID: tenantID,
		Page:     1,
		PageSize: 10000,
	}
	if req.From != nil {
		t := req.From.AsTime()
		input.From = &t
	}
	if req.To != nil {
		t := req.To.AsTime()
		input.To = &t
	}

	rentals, _, err := s.svc.ListRentals(ctx, input)
	if err != nil {
		return nil, mapVermietungError(err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if writeErr := w.Write([]string{"id", "object_id", "renter_name", "start_date", "end_date", "status", "total_price", "deposit_paid"}); writeErr != nil {
		return nil, status.Errorf(codes.Internal, "write csv header: %v", writeErr)
	}

	for _, r := range rentals {
		depositStr := "false"
		if r.DepositPaid {
			depositStr = "true"
		}
		row := []string{
			r.ID.String(),
			r.ObjectID.String(),
			r.RenterName,
			r.StartDate.Format(time.RFC3339),
			r.EndDate.Format(time.RFC3339),
			string(r.Status),
			fmt.Sprintf("%.2f", r.TotalPrice),
			depositStr,
		}
		if writeErr := w.Write(row); writeErr != nil {
			return nil, status.Errorf(codes.Internal, "write csv row: %v", writeErr)
		}
	}
	w.Flush()
	if flushErr := w.Error(); flushErr != nil {
		return nil, status.Errorf(codes.Internal, "flush csv: %v", flushErr)
	}

	filename := fmt.Sprintf("vermietung_%s.csv", tenantID.String()[:8])
	return &vermietungv1.ExportRentalReportResponse{
		Payload:     buf.Bytes(),
		ContentType: "text/csv",
		Filename:    filename,
	}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func objectToProto(obj *vermietung.RentalObject) *vermietungv1.RentalObject {
	if obj == nil {
		return nil
	}
	return &vermietungv1.RentalObject{
		Id:          obj.ID.String(),
		TenantId:    obj.TenantID.String(),
		Name:        obj.Name,
		Description: obj.Description,
		Category:    obj.Category,
		DailyRate:   obj.DailyRate,
		Deposit:     obj.Deposit,
		Location:    obj.Location,
		Notes:       obj.Notes,
		Active:      obj.Active,
		CreatedAt:   timestamppb.New(obj.CreatedAt),
		UpdatedAt:   timestamppb.New(obj.UpdatedAt),
	}
}

func rentalToProto(r *vermietung.Rental) *vermietungv1.Rental {
	if r == nil {
		return nil
	}
	proto := &vermietungv1.Rental{
		Id:          r.ID.String(),
		TenantId:    r.TenantID.String(),
		ObjectId:    r.ObjectID.String(),
		RenterName:  r.RenterName,
		StartDate:   timestamppb.New(r.StartDate),
		EndDate:     timestamppb.New(r.EndDate),
		Status:      string(r.Status),
		TotalPrice:  r.TotalPrice,
		DepositPaid: r.DepositPaid,
		Notes:       r.Notes,
		CreatedAt:   timestamppb.New(r.CreatedAt),
		UpdatedAt:   timestamppb.New(r.UpdatedAt),
	}
	if r.ContactID != nil {
		s := r.ContactID.String()
		proto.ContactId = &s
	}
	if r.SignatureData != nil {
		proto.SignatureData = *r.SignatureData
	}
	if r.SignedBy != nil {
		proto.SignedBy = *r.SignedBy
	}
	if r.SignedAt != nil {
		proto.SignedAt = timestamppb.New(*r.SignedAt)
	}
	return proto
}

func inspectionToProto(ins *vermietung.RentalInspection) *vermietungv1.RentalInspection {
	if ins == nil {
		return nil
	}
	proto := &vermietungv1.RentalInspection{
		Id:            ins.ID.String(),
		TenantId:      ins.TenantID.String(),
		RentalId:      ins.RentalID.String(),
		Kind:          string(ins.Kind),
		Notes:         ins.Notes,
		PhotoUrls:     ins.PhotoURLs,
		Checklist:     checklistToProto(ins.Checklist),
		SignatureData: ins.SignatureData,
		CreatedAt:     timestamppb.New(ins.CreatedAt),
		UpdatedAt:     timestamppb.New(ins.UpdatedAt),
	}
	if ins.PerformedBy != nil {
		s := ins.PerformedBy.String()
		proto.PerformedBy = &s
	}
	return proto
}

// checklistToProto converts inspection checklist items to their wire form.
// A nil/empty input becomes an empty (non-nil) slice so the field never
// serializes as JSON null.
func checklistToProto(items []vermietung.ChecklistItem) []*vermietungv1.ChecklistItem {
	out := make([]*vermietungv1.ChecklistItem, 0, len(items))
	for _, item := range items {
		out = append(out, &vermietungv1.ChecklistItem{
			Label:     item.Label,
			Condition: item.Condition,
			Remark:    item.Remark,
		})
	}
	return out
}

// checklistFromProto converts wire checklist items back to the domain type.
// Returns nil for an empty input so service-layer nil-checks ("was a
// checklist provided at all") keep working.
func checklistFromProto(items []*vermietungv1.ChecklistItem) []vermietung.ChecklistItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]vermietung.ChecklistItem, 0, len(items))
	for _, item := range items {
		out = append(out, vermietung.ChecklistItem{
			Label:     item.GetLabel(),
			Condition: item.GetCondition(),
			Remark:    item.GetRemark(),
		})
	}
	return out
}

// ============================================================================
// Error mapping
// ============================================================================

func mapVermietungError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, vermietung.ErrObjectNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, vermietung.ErrRentalNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, vermietung.ErrInspectionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, vermietung.ErrRentalConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, vermietung.ErrInvalidStateTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, vermietung.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled vermietung service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
