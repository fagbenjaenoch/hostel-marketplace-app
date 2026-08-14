package services

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/fagbenjaenoch/dorms-ng/internal/database/models"
	"github.com/fagbenjaenoch/dorms-ng/internal/dto"
	"github.com/fagbenjaenoch/dorms-ng/internal/middleware"
	"github.com/fagbenjaenoch/dorms-ng/internal/repositories"
	"github.com/fagbenjaenoch/dorms-ng/internal/utils"
	workerpool "github.com/fagbenjaenoch/dorms-ng/internal/workers"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var hostelTracer = otel.Tracer("hostel_service")

type HostelService struct {
	repo   *repositories.HostelRepository
	Logger *zerolog.Logger
	njs    *workerpool.NATSJetStream
}

func NewHostelService(db *sql.DB, logger *zerolog.Logger) *HostelService {
	return &HostelService{
		repo:   repositories.NewHostelRepository(db, logger),
		Logger: logger,
	}
}

func (s HostelService) CreateHostel(ctx context.Context, hostel dto.CreateHostel) (dto.StructuredResponse, error) {
	ctx, span := hostelTracer.Start(ctx, "CreateHostel")
	defer span.End()

	hostelExists, err := s.repo.CheckHostelExists(ctx, hostel.Name)
	if err != nil {
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to check hostel exists",
			Payload: nil,
		}, err
	}

	if hostelExists {
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusConflict,
			Message: "hostel already exists",
			Payload: nil,
		}, nil
	}

	h, err := s.repo.CreateHostel(ctx, hostel)
	if err != nil {
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to create hostel",
			Payload: nil,
		}, err
	}

	err = s.njs.PublishMessage(s.Logger, dto.SearchEvent{
		EventType: "search.created",
		EventPayload: dto.SearchEventPayload{
			Name: h.Name,
		},
	})
	if err != nil {
		s.Logger.Err(err).Msg("failed to publish search.created message")
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to create hostel",
			Payload: nil,
		}, err
	}

	return dto.StructuredResponse{
		Success: true,
		Status:  http.StatusCreated,
		Message: "Hostel created successfully",
		Payload: dto.Hostel{
			Name:                h.Name,
			Description:         h.Description.String,
			Address:             h.Address.String,
			Latitude:            h.Latitude,
			Longitude:           h.Longitude,
			PhotoURLs:           h.PhotoUrls.String,
			Slug:                h.Slug,
			EstimatedPriceRange: h.EstimatedPriceRange.Float64,
			IsVerified:          h.IsVerifiedByAdmin.Bool,
			Amenities:           utils.NullRawMessageToStrings(h.Amenities),
		},
	}, nil
}

func (s HostelService) GetHostel(ctx context.Context, slug string) (dto.StructuredResponse, error) {
	ctx, span := hostelTracer.Start(ctx, "GetHostel")
	defer span.End()

	h, err := s.repo.GetHostel(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.StructuredResponse{
				Success: false,
				Status:  http.StatusNotFound,
				Message: "could not find hostel",
				Payload: nil,
			}, err
		}

		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to get hostel",
			Payload: nil,
		}, err
	}

	return dto.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "Hostel retrieved successfully",
		Payload: dto.Hostel{
			Name:                h.Name,
			Description:         h.Description.String,
			Address:             h.Address.String,
			Latitude:            h.Latitude,
			Longitude:           h.Longitude,
			PhotoURLs:           h.PhotoUrls.String,
			EstimatedPriceRange: h.EstimatedPriceRange.Float64,
			Amenities:           utils.NullRawMessageToStrings(h.Amenities),
		},
	}, nil
}

func (s HostelService) SearchHostels(ctx context.Context, searchType, id string, filters *middleware.HostelFilterParams, paginationParams *middleware.PaginationParams) (dto.StructuredResponse, error) {
	var res []models.Hostel
	var err error

	switch searchType {
	case "institution":
		res, err = s.repo.Queries.GetHostelsByInstitution(ctx, models.GetHostelsByInstitutionParams{
			Limit:         int32(paginationParams.Limit),
			Offset:        int32(paginationParams.Offset),
			MinPrice:      sql.NullFloat64{Float64: float64(filters.MinPrice), Valid: true},
			MaxPrice:      sql.NullFloat64{Float64: float64(filters.MaxPrice), Valid: true},
			InstitutionID: id,
		})
		if err != nil {
			return dto.StructuredResponse{
				Message: "could not find hostels",
				Status:  http.StatusNotFound,
			}, err
		}

	case "neighborhood":
		res, err = s.repo.Queries.GetHostelsByNeighborhood(ctx, models.GetHostelsByNeighborhoodParams{
			Limit:          int32(paginationParams.Limit),
			Offset:         int32(paginationParams.Offset),
			MinPrice:       sql.NullFloat64{Float64: float64(filters.MinPrice), Valid: true},
			MaxPrice:       sql.NullFloat64{Float64: float64(filters.MaxPrice), Valid: true},
			NeighborhoodID: sql.NullString{String: id, Valid: true},
		})
		if err != nil {
			return dto.StructuredResponse{
				Message: "could not find hostels",
				Status:  http.StatusNotFound,
			}, err
		}

	default:
		return dto.StructuredResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid search type",
		}, nil
	}

	var hostels []dto.Hostel

	for _, v := range res {
		hostels = append(hostels, dto.Hostel{
			Name:                v.Name,
			Address:             v.Address.String,
			Description:         v.Description.String,
			Neighborhood:        v.Neighborhood.String,
			EstimatedPriceRange: v.EstimatedPriceRange.Float64,
			Longitude:           v.Longitude,
			Latitude:            v.Latitude,
			Slug:                v.Slug,
			PhotoURLs:           v.PhotoUrls.String,
			IsVerified:          v.IsVerifiedByAdmin.Bool,
			Amenities:           utils.NullRawMessageToStrings(v.Amenities),
		})
	}

	return dto.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "successfully searched hostels",
		Payload: hostels,
	}, nil
}
