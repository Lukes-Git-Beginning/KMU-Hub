package server

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/fuhrpark"
	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
)

// FuhrparkGRPCServer implements the FuhrparkService gRPC server.
type FuhrparkGRPCServer struct {
	fuhrparkv1.UnimplementedFuhrparkServiceServer
	svc *fuhrpark.Service
}

// NewFuhrparkGRPCServer creates a new Fuhrpark gRPC server.
func NewFuhrparkGRPCServer(svc *fuhrpark.Service) *FuhrparkGRPCServer {
	return &FuhrparkGRPCServer{svc: svc}
}

// ============================================================================
// Vehicle RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) CreateVehicle(ctx context.Context, req *fuhrparkv1.CreateVehicleRequest) (*fuhrparkv1.VehicleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := fuhrpark.CreateVehicleInput{
		TenantID:     tenantID,
		LicensePlate: req.GetLicensePlate(),
		Make:         req.GetMake(),
		Model:        req.GetModel(),
		Year:         int(req.GetYear()),
		VIN:          req.Vin,
		Color:        req.Color,
		FuelType:     req.GetFuelType(),
		MileageKm:    req.GetMileageKm(),
		Notes:        req.Notes,
	}

	if req.TuevDueDate != nil {
		t, parseErr := time.Parse("2006-01-02", *req.TuevDueDate)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid tuev_due_date: %v", parseErr)
		}
		input.TuevDueDate = &t
	}
	if req.AssignedDriverId != nil {
		id, parseErr := uuid.Parse(*req.AssignedDriverId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid assigned_driver_id: %v", parseErr)
		}
		input.AssignedDriverID = &id
	}

	v, err := s.svc.CreateVehicle(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.VehicleResponse{Vehicle: vehicleToProto(v)}, nil
}

func (s *FuhrparkGRPCServer) UpdateVehicle(ctx context.Context, req *fuhrparkv1.UpdateVehicleRequest) (*fuhrparkv1.VehicleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	vehicleID, err := uuid.Parse(req.GetVehicleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}

	input := fuhrpark.UpdateVehicleInput{
		TenantID:     tenantID,
		VehicleID:    vehicleID,
		LicensePlate: req.LicensePlate,
		Make:         req.Make,
		Model:        req.Model,
		VIN:          req.Vin,
		Color:        req.Color,
		Notes:        req.Notes,
	}
	if req.Year != nil {
		y := int(*req.Year)
		input.Year = &y
	}
	if req.FuelType != nil {
		ft := fuhrpark.VehicleStatus(*req.FuelType)
		_ = ft
		input.FuelType = req.FuelType
	}
	if req.Status != nil {
		st := fuhrpark.VehicleStatus(*req.Status)
		input.Status = &st
	}
	if req.MileageKm != nil {
		input.MileageKm = req.MileageKm
	}
	if req.TuevDueDate != nil {
		t, parseErr := time.Parse("2006-01-02", *req.TuevDueDate)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid tuev_due_date: %v", parseErr)
		}
		input.TuevDueDate = &t
	}
	if req.AssignedDriverId != nil {
		id, parseErr := uuid.Parse(*req.AssignedDriverId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid assigned_driver_id: %v", parseErr)
		}
		input.AssignedDriverID = &id
	}

	v, err := s.svc.UpdateVehicle(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.VehicleResponse{Vehicle: vehicleToProto(v)}, nil
}

func (s *FuhrparkGRPCServer) DeleteVehicle(ctx context.Context, req *fuhrparkv1.DeleteVehicleRequest) (*fuhrparkv1.DeleteVehicleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	vehicleID, err := uuid.Parse(req.GetVehicleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}
	if err := s.svc.DeleteVehicle(ctx, tenantID, vehicleID); err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DeleteVehicleResponse{}, nil
}

func (s *FuhrparkGRPCServer) GetVehicle(ctx context.Context, req *fuhrparkv1.GetVehicleRequest) (*fuhrparkv1.VehicleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	vehicleID, err := uuid.Parse(req.GetVehicleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}
	v, err := s.svc.GetVehicle(ctx, tenantID, vehicleID)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.VehicleResponse{Vehicle: vehicleToProto(v)}, nil
}

func (s *FuhrparkGRPCServer) ListVehicles(ctx context.Context, req *fuhrparkv1.ListVehiclesRequest) (*fuhrparkv1.ListVehiclesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := fuhrpark.ListVehiclesInput{
		TenantID: tenantID,
		Search:   req.GetSearch(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.Status != nil {
		st := fuhrpark.VehicleStatus(*req.Status)
		input.Status = &st
	}
	if req.FuelType != nil {
		input.FuelType = req.FuelType
	}

	vehicles, total, err := s.svc.ListVehicles(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}

	protoVehicles := make([]*fuhrparkv1.Vehicle, len(vehicles))
	for i, v := range vehicles {
		protoVehicles[i] = vehicleToProto(v)
	}
	return &fuhrparkv1.ListVehiclesResponse{
		Vehicles: protoVehicles,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Service RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) ScheduleService(ctx context.Context, req *fuhrparkv1.ScheduleServiceRequest) (*fuhrparkv1.VehicleServiceResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	vehicleID, err := uuid.Parse(req.GetVehicleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}

	scheduledAt, parseErr := time.Parse("2006-01-02", req.GetScheduledAt())
	if parseErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid scheduled_at: %v", parseErr)
	}

	input := fuhrpark.ScheduleServiceInput{
		TenantID:    tenantID,
		VehicleID:   vehicleID,
		ServiceType: req.GetServiceType(),
		Description: req.Description,
		ScheduledAt: scheduledAt,
		CostCents:   req.CostCents,
		Workshop:    req.Workshop,
		MileageKm:   req.MileageKm,
		Notes:       req.Notes,
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	svc, err := s.svc.ScheduleService(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.VehicleServiceResponse{Service: serviceToProto(svc)}, nil
}

func (s *FuhrparkGRPCServer) UpdateService(ctx context.Context, req *fuhrparkv1.UpdateServiceRequest) (*fuhrparkv1.VehicleServiceResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	serviceID, err := uuid.Parse(req.GetServiceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid service_id: %v", err)
	}

	input := fuhrpark.UpdateServiceInput{
		TenantID:    tenantID,
		ServiceID:   serviceID,
		ServiceType: req.ServiceType,
		Description: req.Description,
		CostCents:   req.CostCents,
		Workshop:    req.Workshop,
		MileageKm:   req.MileageKm,
		Notes:       req.Notes,
	}
	if req.ScheduledAt != nil {
		t, parseErr := time.Parse("2006-01-02", *req.ScheduledAt)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid scheduled_at: %v", parseErr)
		}
		input.ScheduledAt = &t
	}
	if req.Status != nil {
		st := fuhrpark.ServiceStatus(*req.Status)
		input.Status = &st
	}

	svc, err := s.svc.UpdateService(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.VehicleServiceResponse{Service: serviceToProto(svc)}, nil
}

func (s *FuhrparkGRPCServer) DeleteService(ctx context.Context, req *fuhrparkv1.DeleteServiceRequest) (*fuhrparkv1.DeleteServiceResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	serviceID, err := uuid.Parse(req.GetServiceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid service_id: %v", err)
	}
	if err := s.svc.DeleteService(ctx, tenantID, serviceID); err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DeleteServiceResponse{}, nil
}

func (s *FuhrparkGRPCServer) CompleteService(ctx context.Context, req *fuhrparkv1.CompleteServiceRequest) (*fuhrparkv1.VehicleServiceResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	serviceID, err := uuid.Parse(req.GetServiceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid service_id: %v", err)
	}

	svc, err := s.svc.CompleteService(ctx, fuhrpark.CompleteServiceInput{
		TenantID:  tenantID,
		ServiceID: serviceID,
		CostCents: req.CostCents,
		MileageKm: req.MileageKm,
		Notes:     req.Notes,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.VehicleServiceResponse{Service: serviceToProto(svc)}, nil
}

func (s *FuhrparkGRPCServer) ListServices(ctx context.Context, req *fuhrparkv1.ListServicesRequest) (*fuhrparkv1.ListServicesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := fuhrpark.ListServicesInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.VehicleId != nil {
		id, parseErr := uuid.Parse(*req.VehicleId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", parseErr)
		}
		input.VehicleID = &id
	}
	if req.Status != nil {
		st := fuhrpark.ServiceStatus(*req.Status)
		input.Status = &st
	}

	services, total, err := s.svc.ListServices(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}

	protoServices := make([]*fuhrparkv1.VehicleService, len(services))
	for i, svc := range services {
		protoServices[i] = serviceToProto(svc)
	}
	return &fuhrparkv1.ListServicesResponse{
		Services: protoServices,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Damage RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) ReportDamage(ctx context.Context, req *fuhrparkv1.ReportDamageRequest) (*fuhrparkv1.DamageResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	vehicleID, err := uuid.Parse(req.GetVehicleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}

	input := fuhrpark.ReportDamageInput{
		TenantID:    tenantID,
		VehicleID:   vehicleID,
		Description: req.GetDescription(),
		Severity:    req.GetSeverity(),
		PhotoKeys:   req.GetPhotoKeys(),
		CostCents:   req.CostCents,
		Notes:       req.Notes,
	}
	if req.ReportedBy != nil {
		id, parseErr := uuid.Parse(*req.ReportedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid reported_by: %v", parseErr)
		}
		input.ReportedBy = &id
	}

	d, err := s.svc.ReportDamage(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DamageResponse{Damage: damageToProto(d)}, nil
}

func (s *FuhrparkGRPCServer) UpdateDamage(ctx context.Context, req *fuhrparkv1.UpdateDamageRequest) (*fuhrparkv1.DamageResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	damageID, err := uuid.Parse(req.GetDamageId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid damage_id: %v", err)
	}

	input := fuhrpark.UpdateDamageInput{
		TenantID:    tenantID,
		DamageID:    damageID,
		Description: req.Description,
		Severity:    req.Severity,
		CostCents:   req.CostCents,
		Notes:       req.Notes,
	}
	if len(req.GetPhotoKeys()) > 0 {
		input.PhotoKeys = req.GetPhotoKeys()
	}
	if req.Status != nil {
		st := fuhrpark.DamageStatus(*req.Status)
		input.Status = &st
	}

	d, err := s.svc.UpdateDamage(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DamageResponse{Damage: damageToProto(d)}, nil
}

func (s *FuhrparkGRPCServer) ResolveDamage(ctx context.Context, req *fuhrparkv1.ResolveDamageRequest) (*fuhrparkv1.DamageResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	damageID, err := uuid.Parse(req.GetDamageId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid damage_id: %v", err)
	}

	input := fuhrpark.ResolveDamageInput{
		TenantID:  tenantID,
		DamageID:  damageID,
		CostCents: req.CostCents,
		Notes:     req.Notes,
	}
	if req.ResolvedBy != nil {
		id, parseErr := uuid.Parse(*req.ResolvedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid resolved_by: %v", parseErr)
		}
		input.ResolvedBy = &id
	}

	d, err := s.svc.ResolveDamage(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DamageResponse{Damage: damageToProto(d)}, nil
}

func (s *FuhrparkGRPCServer) ListDamages(ctx context.Context, req *fuhrparkv1.ListDamagesRequest) (*fuhrparkv1.ListDamagesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := fuhrpark.ListDamagesInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.VehicleId != nil {
		id, parseErr := uuid.Parse(*req.VehicleId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", parseErr)
		}
		input.VehicleID = &id
	}
	if req.Status != nil {
		st := fuhrpark.DamageStatus(*req.Status)
		input.Status = &st
	}

	damages, total, err := s.svc.ListDamages(ctx, input)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}

	protoDamages := make([]*fuhrparkv1.Damage, len(damages))
	for i, d := range damages {
		protoDamages[i] = damageToProto(d)
	}
	return &fuhrparkv1.ListDamagesResponse{
		Damages: protoDamages,
		Total:   int32(total),
	}, nil
}

// ============================================================================
// History & Report RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) GetVehicleHistory(ctx context.Context, req *fuhrparkv1.GetVehicleHistoryRequest) (*fuhrparkv1.VehicleHistoryResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	vehicleID, err := uuid.Parse(req.GetVehicleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}

	entries, total, err := s.svc.GetVehicleHistory(ctx, fuhrpark.GetVehicleHistoryInput{
		TenantID:  tenantID,
		VehicleID: vehicleID,
		Page:      int(req.GetPage()),
		PageSize:  int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}

	protoEntries := make([]*fuhrparkv1.VehicleHistoryEntry, len(entries))
	for i, e := range entries {
		protoEntries[i] = &fuhrparkv1.VehicleHistoryEntry{
			Kind:       e.Kind,
			Id:         e.ID.String(),
			Summary:    e.Summary,
			OccurredAt: timestamppb.New(e.OccurredAt),
		}
	}
	return &fuhrparkv1.VehicleHistoryResponse{
		Entries: protoEntries,
		Total:   int32(total),
	}, nil
}

func (s *FuhrparkGRPCServer) CheckTuevDue(ctx context.Context, req *fuhrparkv1.CheckTuevDueRequest) (*fuhrparkv1.CheckTuevDueResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	days := int(req.GetDaysAhead())
	if days <= 0 {
		days = 30
	}

	vehicles, err := s.svc.CheckTuevDue(ctx, tenantID, days)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}

	protoVehicles := make([]*fuhrparkv1.Vehicle, len(vehicles))
	for i, v := range vehicles {
		protoVehicles[i] = vehicleToProto(v)
	}
	return &fuhrparkv1.CheckTuevDueResponse{Vehicles: protoVehicles}, nil
}

func (s *FuhrparkGRPCServer) ListUpcomingServices(ctx context.Context, req *fuhrparkv1.ListUpcomingServicesRequest) (*fuhrparkv1.ListServicesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	services, total, err := s.svc.ListUpcomingServices(
		ctx, tenantID, int(req.GetDaysAhead()), int(req.GetPage()), int(req.GetPageSize()),
	)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}

	protoServices := make([]*fuhrparkv1.VehicleService, len(services))
	for i, svc := range services {
		protoServices[i] = serviceToProto(svc)
	}
	return &fuhrparkv1.ListServicesResponse{
		Services: protoServices,
		Total:    int32(total),
	}, nil
}

func (s *FuhrparkGRPCServer) ExportVehicleReport(ctx context.Context, req *fuhrparkv1.ExportVehicleReportRequest) (*fuhrparkv1.ExportVehicleReportResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	vehicles, _, err := s.svc.ListVehicles(ctx, fuhrpark.ListVehiclesInput{
		TenantID: tenantID, Page: 1, PageSize: 10000,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"ID", "License Plate", "Make", "Model", "Year", "Fuel Type", "Status", "Mileage (km)", "TUEV Due"})
	for _, v := range vehicles {
		tuev := ""
		if v.TuevDueDate != nil {
			tuev = v.TuevDueDate.Format("2006-01-02")
		}
		_ = w.Write([]string{
			v.ID.String(), v.LicensePlate, v.Make, v.Model,
			fmt.Sprintf("%d", v.Year), v.FuelType, string(v.Status),
			fmt.Sprintf("%d", v.MileageKm), tuev,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, status.Errorf(codes.Internal, "csv flush: %v", err)
	}

	return &fuhrparkv1.ExportVehicleReportResponse{
		Payload:     buf.Bytes(),
		ContentType: "text/csv; charset=utf-8",
		Filename:    fmt.Sprintf("fuhrpark-%s.csv", time.Now().Format("2006-01-02")),
	}, nil
}

// ============================================================================
// Mapping helpers
// ============================================================================

func vehicleToProto(v *fuhrpark.Vehicle) *fuhrparkv1.Vehicle {
	p := &fuhrparkv1.Vehicle{
		Id:           v.ID.String(),
		TenantId:     v.TenantID.String(),
		LicensePlate: v.LicensePlate,
		Make:         v.Make,
		Model:        v.Model,
		Year:         int32(v.Year),
		Vin:          v.VIN,
		Color:        v.Color,
		FuelType:     v.FuelType,
		Status:       string(v.Status),
		MileageKm:    v.MileageKm,
		Notes:        v.Notes,
		CreatedAt:    timestamppb.New(v.CreatedAt),
		UpdatedAt:    timestamppb.New(v.UpdatedAt),
	}
	if v.TuevDueDate != nil {
		s := v.TuevDueDate.Format("2006-01-02")
		p.TuevDueDate = &s
	}
	if v.AssignedDriverID != nil {
		s := v.AssignedDriverID.String()
		p.AssignedDriverId = &s
	}
	return p
}

func serviceToProto(s *fuhrpark.VehicleService) *fuhrparkv1.VehicleService {
	p := &fuhrparkv1.VehicleService{
		Id:          s.ID.String(),
		TenantId:    s.TenantID.String(),
		VehicleId:   s.VehicleID.String(),
		ServiceType: s.ServiceType,
		Description: s.Description,
		ScheduledAt: s.ScheduledAt.Format("2006-01-02"),
		CostCents:   s.CostCents,
		Workshop:    s.Workshop,
		MileageKm:   s.MileageKm,
		Status:      string(s.Status),
		Notes:       s.Notes,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
	if s.CompletedAt != nil {
		p.CompletedAt = timestamppb.New(*s.CompletedAt)
	}
	if s.CreatedBy != nil {
		str := s.CreatedBy.String()
		p.CreatedBy = &str
	}
	return p
}

func damageToProto(d *fuhrpark.VehicleDamage) *fuhrparkv1.Damage {
	p := &fuhrparkv1.Damage{
		Id:          d.ID.String(),
		TenantId:    d.TenantID.String(),
		VehicleId:   d.VehicleID.String(),
		Description: d.Description,
		Severity:    d.Severity,
		Status:      string(d.Status),
		PhotoKeys:   d.PhotoKeys,
		CostCents:   d.CostCents,
		Notes:       d.Notes,
		CreatedAt:   timestamppb.New(d.CreatedAt),
		UpdatedAt:   timestamppb.New(d.UpdatedAt),
	}
	if d.ReportedBy != nil {
		s := d.ReportedBy.String()
		p.ReportedBy = &s
	}
	if d.ResolvedBy != nil {
		s := d.ResolvedBy.String()
		p.ResolvedBy = &s
	}
	if d.ResolvedAt != nil {
		p.ResolvedAt = timestamppb.New(*d.ResolvedAt)
	}
	return p
}

func mapFuhrparkError(err error) error {
	switch {
	case errors.Is(err, fuhrpark.ErrVehicleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrServiceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrDamageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrPlateTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, fuhrpark.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, fuhrpark.ErrInvalidTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
