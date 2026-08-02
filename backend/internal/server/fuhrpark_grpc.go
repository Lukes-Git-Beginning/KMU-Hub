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

	"github.com/kmuhub/kmuhub/internal/fuhrpark"
	"github.com/kmuhub/kmuhub/internal/middleware"
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

// ============================================================================
// Fuel Log RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) ListFuelLogs(ctx context.Context, req *fuhrparkv1.ListFuelLogsRequest) (*fuhrparkv1.ListFuelLogsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	params := fuhrpark.ListFuelLogsParams{
		TenantID: tenantID,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if req.VehicleId != "" {
		id, parseErr := uuid.Parse(req.VehicleId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", parseErr)
		}
		params.VehicleID = id
	}
	logs, total, err := s.svc.ListFuelLogs(ctx, params)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	pb := make([]*fuhrparkv1.FuelLog, len(logs))
	for i, l := range logs {
		pb[i] = fuelLogToProto(l)
	}
	return &fuhrparkv1.ListFuelLogsResponse{FuelLogs: pb, Total: int32(total)}, nil
}

func (s *FuhrparkGRPCServer) CreateFuelLog(ctx context.Context, req *fuhrparkv1.CreateFuelLogRequest) (*fuhrparkv1.CreateFuelLogResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	vehicleID, err := uuid.Parse(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}
	date, _ := time.Parse("2006-01-02", req.Date)
	l, err := s.svc.CreateFuelLog(ctx, fuhrpark.FuelLog{
		TenantID:  tenantID,
		VehicleID: vehicleID,
		Date:      date,
		Liters:    req.Liters,
		CostCents: req.CostCents,
		MileageKm: req.MileageKm,
		FuelType:  req.FuelType,
		Notes:     req.Notes,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.CreateFuelLogResponse{FuelLog: fuelLogToProto(l)}, nil
}

func (s *FuhrparkGRPCServer) UpdateFuelLog(ctx context.Context, req *fuhrparkv1.UpdateFuelLogRequest) (*fuhrparkv1.UpdateFuelLogResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	date, _ := time.Parse("2006-01-02", req.Date)
	l, err := s.svc.UpdateFuelLog(ctx, fuhrpark.FuelLog{
		ID:        id,
		TenantID:  tenantID,
		Date:      date,
		Liters:    req.Liters,
		CostCents: req.CostCents,
		MileageKm: req.MileageKm,
		FuelType:  req.FuelType,
		Notes:     req.Notes,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.UpdateFuelLogResponse{FuelLog: fuelLogToProto(l)}, nil
}

func (s *FuhrparkGRPCServer) DeleteFuelLog(ctx context.Context, req *fuhrparkv1.DeleteFuelLogRequest) (*fuhrparkv1.DeleteFuelLogResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	if err := s.svc.DeleteFuelLog(ctx, tenantID, id); err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DeleteFuelLogResponse{}, nil
}

// ============================================================================
// Trip Log RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) ListTripLogs(ctx context.Context, req *fuhrparkv1.ListTripLogsRequest) (*fuhrparkv1.ListTripLogsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	params := fuhrpark.ListTripLogsParams{
		TenantID: tenantID,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if req.VehicleId != "" {
		id, parseErr := uuid.Parse(req.VehicleId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", parseErr)
		}
		params.VehicleID = id
	}
	logs, total, err := s.svc.ListTripLogs(ctx, params)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	pb := make([]*fuhrparkv1.TripLog, len(logs))
	for i, l := range logs {
		pb[i] = tripLogToProto(l)
	}
	return &fuhrparkv1.ListTripLogsResponse{TripLogs: pb, Total: int32(total)}, nil
}

func (s *FuhrparkGRPCServer) CreateTripLog(ctx context.Context, req *fuhrparkv1.CreateTripLogRequest) (*fuhrparkv1.CreateTripLogResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	vehicleID, err := uuid.Parse(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}
	date, _ := time.Parse("2006-01-02", req.Date)
	l, err := s.svc.CreateTripLog(ctx, fuhrpark.TripLog{
		TenantID:      tenantID,
		VehicleID:     vehicleID,
		Date:          date,
		StartLocation: req.StartLocation,
		EndLocation:   req.EndLocation,
		Purpose:       req.Purpose,
		StartKm:       req.StartKm,
		EndKm:         req.EndKm,
		IsPrivate:     req.IsPrivate,
		DriverName:    req.DriverName,
		Notes:         req.Notes,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.CreateTripLogResponse{TripLog: tripLogToProto(l)}, nil
}

func (s *FuhrparkGRPCServer) UpdateTripLog(ctx context.Context, req *fuhrparkv1.UpdateTripLogRequest) (*fuhrparkv1.UpdateTripLogResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	date, _ := time.Parse("2006-01-02", req.Date)
	l, err := s.svc.UpdateTripLog(ctx, fuhrpark.TripLog{
		ID:            id,
		TenantID:      tenantID,
		Date:          date,
		StartLocation: req.StartLocation,
		EndLocation:   req.EndLocation,
		Purpose:       req.Purpose,
		StartKm:       req.StartKm,
		EndKm:         req.EndKm,
		IsPrivate:     req.IsPrivate,
		DriverName:    req.DriverName,
		Notes:         req.Notes,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.UpdateTripLogResponse{TripLog: tripLogToProto(l)}, nil
}

func (s *FuhrparkGRPCServer) DeleteTripLog(ctx context.Context, req *fuhrparkv1.DeleteTripLogRequest) (*fuhrparkv1.DeleteTripLogResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	if err := s.svc.DeleteTripLog(ctx, tenantID, id); err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DeleteTripLogResponse{}, nil
}

// ============================================================================
// Vehicle Document RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) ListVehicleDocuments(ctx context.Context, req *fuhrparkv1.ListVehicleDocumentsRequest) (*fuhrparkv1.ListVehicleDocumentsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	vehicleID, err := uuid.Parse(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}
	docs, total, err := s.svc.ListVehicleDocuments(ctx, fuhrpark.ListVehicleDocumentsParams{
		TenantID:  tenantID,
		VehicleID: vehicleID,
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	pb := make([]*fuhrparkv1.VehicleDocument, len(docs))
	for i, d := range docs {
		pb[i] = vehicleDocumentToProto(d)
	}
	return &fuhrparkv1.ListVehicleDocumentsResponse{Documents: pb, Total: int32(total)}, nil
}

func (s *FuhrparkGRPCServer) CreateVehicleDocument(ctx context.Context, req *fuhrparkv1.CreateVehicleDocumentRequest) (*fuhrparkv1.CreateVehicleDocumentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	vehicleID, err := uuid.Parse(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}
	var expiryDate *time.Time
	if req.ExpiryDate != "" {
		t, parseErr := time.Parse("2006-01-02", req.ExpiryDate)
		if parseErr == nil {
			expiryDate = &t
		}
	}
	doc, err := s.svc.CreateVehicleDocument(ctx, fuhrpark.VehicleDocument{
		TenantID:   tenantID,
		VehicleID:  vehicleID,
		DocType:    req.DocType,
		Name:       req.Name,
		ObjectKey:  req.ObjectKey,
		UploadDate: time.Now(),
		ExpiryDate: expiryDate,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.CreateVehicleDocumentResponse{Document: vehicleDocumentToProto(doc)}, nil
}

func (s *FuhrparkGRPCServer) DeleteVehicleDocument(ctx context.Context, req *fuhrparkv1.DeleteVehicleDocumentRequest) (*fuhrparkv1.DeleteVehicleDocumentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	if err := s.svc.DeleteVehicleDocument(ctx, tenantID, id); err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DeleteVehicleDocumentResponse{}, nil
}

// ============================================================================
// Driver License RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) ListDriverLicenses(ctx context.Context, req *fuhrparkv1.ListDriverLicensesRequest) (*fuhrparkv1.ListDriverLicensesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	var driverID uuid.UUID
	if req.DriverId != "" {
		driverID, err = uuid.Parse(req.DriverId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid driver_id: %v", err)
		}
	}
	licenses, total, err := s.svc.ListDriverLicenses(ctx, fuhrpark.ListDriverLicensesParams{
		TenantID: tenantID,
		DriverID: driverID,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	pb := make([]*fuhrparkv1.DriverLicense, len(licenses))
	for i, l := range licenses {
		pb[i] = driverLicenseToProto(l)
	}
	return &fuhrparkv1.ListDriverLicensesResponse{Licenses: pb, Total: int32(total)}, nil
}

func (s *FuhrparkGRPCServer) CreateDriverLicense(ctx context.Context, req *fuhrparkv1.CreateDriverLicenseRequest) (*fuhrparkv1.CreateDriverLicenseResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	driverID, err := uuid.Parse(req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid driver_id: %v", err)
	}
	checkedAt := time.Now()
	if req.CheckedAt != "" {
		t, parseErr := time.Parse("2006-01-02", req.CheckedAt)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid checked_at: %v", parseErr)
		}
		checkedAt = t
	}
	nextDue, err := time.Parse("2006-01-02", req.NextCheckDueDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid next_check_due_date: %v", err)
	}
	var expiryDate *time.Time
	if req.ExpiryDate != "" {
		t, parseErr := time.Parse("2006-01-02", req.ExpiryDate)
		if parseErr == nil {
			expiryDate = &t
		}
	}
	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}
	lic, err := s.svc.CreateDriverLicense(ctx, fuhrpark.DriverLicense{
		TenantID:         tenantID,
		DriverID:         driverID,
		LicenseClasses:   req.LicenseClasses,
		ExpiryDate:       expiryDate,
		CheckedAt:        checkedAt,
		NextCheckDueDate: nextDue,
		Notes:            notes,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.CreateDriverLicenseResponse{License: driverLicenseToProto(lic)}, nil
}

func (s *FuhrparkGRPCServer) UpdateDriverLicense(ctx context.Context, req *fuhrparkv1.UpdateDriverLicenseRequest) (*fuhrparkv1.UpdateDriverLicenseResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	nextDue, err := time.Parse("2006-01-02", req.NextCheckDueDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid next_check_due_date: %v", err)
	}
	checkedAt := time.Now()
	if req.CheckedAt != "" {
		t, parseErr := time.Parse("2006-01-02", req.CheckedAt)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid checked_at: %v", parseErr)
		}
		checkedAt = t
	}
	var expiryDate *time.Time
	if req.ExpiryDate != "" {
		t, parseErr := time.Parse("2006-01-02", req.ExpiryDate)
		if parseErr == nil {
			expiryDate = &t
		}
	}
	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}
	lic, err := s.svc.UpdateDriverLicense(ctx, fuhrpark.DriverLicense{
		ID:               id,
		TenantID:         tenantID,
		LicenseClasses:   req.LicenseClasses,
		ExpiryDate:       expiryDate,
		CheckedAt:        checkedAt,
		NextCheckDueDate: nextDue,
		Notes:            notes,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.UpdateDriverLicenseResponse{License: driverLicenseToProto(lic)}, nil
}

func (s *FuhrparkGRPCServer) DeleteDriverLicense(ctx context.Context, req *fuhrparkv1.DeleteDriverLicenseRequest) (*fuhrparkv1.DeleteDriverLicenseResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	if err := s.svc.DeleteDriverLicense(ctx, tenantID, id); err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.DeleteDriverLicenseResponse{}, nil
}

// ============================================================================
// GPS RPCs
// ============================================================================

func (s *FuhrparkGRPCServer) IngestGpsPositions(ctx context.Context, req *fuhrparkv1.IngestGpsPositionsRequest) (*fuhrparkv1.IngestGpsPositionsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	vehicleID, err := uuid.Parse(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}
	positions := make([]fuhrpark.GpsPosition, 0, len(req.Positions))
	for _, p := range req.Positions {
		t, _ := time.Parse(time.RFC3339, p.RecordedAt)
		pos := fuhrpark.GpsPosition{
			Lat:        p.Lat,
			Lng:        p.Lng,
			RecordedAt: t,
		}
		if p.SpeedKmh != 0 {
			v := p.SpeedKmh
			pos.SpeedKmh = &v
		}
		positions = append(positions, pos)
	}
	n, err := s.svc.IngestGpsPositions(ctx, tenantID, vehicleID, positions)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	return &fuhrparkv1.IngestGpsPositionsResponse{Ingested: int32(n)}, nil
}

func (s *FuhrparkGRPCServer) GetVehicleRoutes(ctx context.Context, req *fuhrparkv1.GetVehicleRoutesRequest) (*fuhrparkv1.GetVehicleRoutesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	dateTo, _ := time.Parse("2006-01-02", req.DateTo)
	dateFrom, _ := time.Parse("2006-01-02", req.DateFrom)
	if dateTo.IsZero() {
		dateTo = time.Now()
	}
	if dateFrom.IsZero() {
		dateFrom = dateTo.AddDate(0, 0, -7)
	}
	params := fuhrpark.GetVehicleRoutesParams{
		TenantID: tenantID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
	}
	if req.VehicleId != "" {
		id, parseErr := uuid.Parse(req.VehicleId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", parseErr)
		}
		params.VehicleID = id
	}
	routes, err := s.svc.GetVehicleRoutes(ctx, params)
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	pb := make([]*fuhrparkv1.VehicleRoute, len(routes))
	for i, r := range routes {
		pb[i] = vehicleRouteToProto(r)
	}
	return &fuhrparkv1.GetVehicleRoutesResponse{Routes: pb}, nil
}

func (s *FuhrparkGRPCServer) GetGpsPositions(ctx context.Context, req *fuhrparkv1.GetGpsPositionsRequest) (*fuhrparkv1.GetGpsPositionsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}
	vehicleID, err := uuid.Parse(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle_id: %v", err)
	}
	to, _ := time.Parse(time.RFC3339, req.To)
	from, _ := time.Parse(time.RFC3339, req.From)
	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() {
		from = to.Add(-24 * time.Hour)
	}
	positions, err := s.svc.GetGpsPositions(ctx, fuhrpark.GetGpsPositionsParams{
		TenantID:  tenantID,
		VehicleID: vehicleID,
		From:      from,
		To:        to,
		Limit:     req.Limit,
	})
	if err != nil {
		return nil, mapFuhrparkError(err)
	}
	pb := make([]*fuhrparkv1.GpsPosition, len(positions))
	for i, p := range positions {
		pb[i] = gpsPositionToProto(p)
	}
	return &fuhrparkv1.GetGpsPositionsResponse{Positions: pb}, nil
}

// ============================================================================
// Proto-Mapping Helpers (extended)
// ============================================================================

func fuelLogToProto(l fuhrpark.FuelLog) *fuhrparkv1.FuelLog {
	return &fuhrparkv1.FuelLog{
		Id:        l.ID.String(),
		TenantId:  l.TenantID.String(),
		VehicleId: l.VehicleID.String(),
		Date:      l.Date.Format("2006-01-02"),
		Liters:    l.Liters,
		CostCents: l.CostCents,
		MileageKm: l.MileageKm,
		FuelType:  l.FuelType,
		Notes:     l.Notes,
		CreatedAt: l.CreatedAt.Format(time.RFC3339),
		UpdatedAt: l.UpdatedAt.Format(time.RFC3339),
	}
}

func tripLogToProto(l fuhrpark.TripLog) *fuhrparkv1.TripLog {
	return &fuhrparkv1.TripLog{
		Id:            l.ID.String(),
		TenantId:      l.TenantID.String(),
		VehicleId:     l.VehicleID.String(),
		Date:          l.Date.Format("2006-01-02"),
		StartLocation: l.StartLocation,
		EndLocation:   l.EndLocation,
		Purpose:       l.Purpose,
		StartKm:       l.StartKm,
		EndKm:         l.EndKm,
		Km:            l.Km,
		IsPrivate:     l.IsPrivate,
		DriverName:    l.DriverName,
		Notes:         l.Notes,
		CreatedAt:     l.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     l.UpdatedAt.Format(time.RFC3339),
	}
}

func vehicleDocumentToProto(d fuhrpark.VehicleDocument) *fuhrparkv1.VehicleDocument {
	expiry := ""
	if d.ExpiryDate != nil {
		expiry = d.ExpiryDate.Format("2006-01-02")
	}
	return &fuhrparkv1.VehicleDocument{
		Id:         d.ID.String(),
		TenantId:   d.TenantID.String(),
		VehicleId:  d.VehicleID.String(),
		DocType:    d.DocType,
		Name:       d.Name,
		ObjectKey:  d.ObjectKey,
		UploadDate: d.UploadDate.Format("2006-01-02"),
		ExpiryDate: expiry,
		CreatedAt:  d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  d.UpdatedAt.Format(time.RFC3339),
	}
}

func driverLicenseToProto(l fuhrpark.DriverLicense) *fuhrparkv1.DriverLicense {
	expiry := ""
	if l.ExpiryDate != nil {
		expiry = l.ExpiryDate.Format("2006-01-02")
	}
	notes := ""
	if l.Notes != nil {
		notes = *l.Notes
	}
	return &fuhrparkv1.DriverLicense{
		Id:               l.ID.String(),
		TenantId:         l.TenantID.String(),
		DriverId:         l.DriverID.String(),
		LicenseClasses:   l.LicenseClasses,
		ExpiryDate:       expiry,
		CheckedAt:        l.CheckedAt.Format("2006-01-02"),
		NextCheckDueDate: l.NextCheckDueDate.Format("2006-01-02"),
		Notes:            notes,
		CreatedAt:        l.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        l.UpdatedAt.Format(time.RFC3339),
	}
}

func gpsPositionToProto(p fuhrpark.GpsPosition) *fuhrparkv1.GpsPosition {
	speed := 0.0
	if p.SpeedKmh != nil {
		speed = *p.SpeedKmh
	}
	return &fuhrparkv1.GpsPosition{
		Id:         p.ID.String(),
		VehicleId:  p.VehicleID.String(),
		Lat:        p.Lat,
		Lng:        p.Lng,
		SpeedKmh:   speed,
		RecordedAt: p.RecordedAt.Format(time.RFC3339),
	}
}

func vehicleRouteToProto(r fuhrpark.VehicleRouteAggregation) *fuhrparkv1.VehicleRoute {
	positions := make([]*fuhrparkv1.GpsPosition, len(r.Positions))
	for i, p := range r.Positions {
		positions[i] = gpsPositionToProto(p)
	}
	return &fuhrparkv1.VehicleRoute{
		VehicleId:   r.VehicleID.String(),
		VehicleName: r.VehicleName,
		Date:        r.Date.Format("2006-01-02"),
		Positions:   positions,
		DailyKm:     r.DailyKm,
		Status:      r.Status,
	}
}

func mapFuhrparkError(err error) error {
	switch {
	case errors.Is(err, fuhrpark.ErrVehicleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrServiceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrDamageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrFuelLogNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrTripLogNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrDocumentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrDriverLicenseNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrDriverNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, fuhrpark.ErrPlateTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, fuhrpark.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, fuhrpark.ErrInvalidTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		slog.Error("unhandled fuhrpark service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
