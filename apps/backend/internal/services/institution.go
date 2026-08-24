package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/fagbenjaenoch/dorms-ng/internal/dto"
	"github.com/fagbenjaenoch/dorms-ng/internal/repositories"
	workerpool "github.com/fagbenjaenoch/dorms-ng/internal/workers"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var institutionTracer = otel.Tracer("institution_service")

type InstitutionService struct {
	repo   *repositories.InstitutionRepository
	Logger *zerolog.Logger
	njs    *workerpool.NATSJetStream
}

func NewInstitutionService(db *sql.DB, logger *zerolog.Logger, njs *workerpool.NATSJetStream) *InstitutionService {
	return &InstitutionService{
		repo:   repositories.NewInstitutionRepository(db, logger),
		Logger: logger,
		njs:    njs,
	}
}

func (s InstitutionService) CreateInstitution(ctx context.Context, institution dto.CreateInstitution) (dto.StructuredResponse, error) {
	ctx, span := institutionTracer.Start(ctx, "CreateInstitution")
	defer span.End()

	institutionExists, err := s.repo.CheckInstitutionExists(ctx, institution)
	if err != nil {
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to check if institution exists",
			Payload: nil,
		}, fmt.Errorf("failed to check if institution exists: %s", err.Error())
	}

	if institutionExists {
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusConflict,
			Message: "institution already exists",
			Payload: nil,
		}, nil
	}

	i, err := s.repo.CreateInstitution(ctx, institution)
	if err != nil {
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to create institution",
			Payload: nil,
		}, fmt.Errorf("repo.CreateInstitution error: %s", err.Error())
	}

	err = s.njs.PublishMessage(ctx, s.Logger, dto.SearchEvent{
		EventType: "institution.create",
		EventPayload: dto.SearchEventPayload{
			Name: i.Name,
		},
	})

	if err != nil {
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to publish event",
			Payload: nil,
		}, fmt.Errorf("failed to publish event: %s", err.Error())
	}

	return dto.StructuredResponse{
		Success: true,
		Status:  http.StatusCreated,
		Message: "Institution created successfully",
		Payload: dto.Institution{
			ID:          i.ID,
			Name:        i.Name,
			Acronym:     i.Acronym.String,
			Latitude:    i.Latitude,
			Longitude:   i.Longitude,
			State:       i.State,
			City:        i.City,
			Description: i.Description.String,
		},
	}, nil
}

func (s InstitutionService) GetInstitution(ctx context.Context, slug string) (dto.StructuredResponse, error) {
	ctx, span := institutionTracer.Start(ctx, "GetInstitution")
	defer span.End()

	i, err := s.repo.GetInstitution(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.StructuredResponse{
				Success: false,
				Status:  http.StatusNotFound,
				Message: "could not find institution",
				Payload: nil,
			}, err
		}

		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to get institution",
			Payload: nil,
		}, err
	}

	return dto.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "Institution retrieved successfully",
		Payload: dto.Institution{
			ID:          i.ID,
			Name:        i.Name,
			Acronym:     i.Acronym.String,
			State:       i.State,
			City:        i.City,
			Latitude:    i.Latitude,
			Longitude:   i.Longitude,
			Description: i.Description.String,
		},
	}, nil
}

func (s InstitutionService) GetAllInstitutions(ctx context.Context) (dto.StructuredResponse, error) {
	ctx, span := institutionTracer.Start(ctx, "GetAllInstitutions")
	defer span.End()

	institutions, err := s.repo.GetAllInstitutions(ctx)
	if err != nil {
		return dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to get all institutions",
			Payload: nil,
		}, err
	}

	return dto.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "Institutions retrieved successfully",
		Payload: institutions,
	}, nil
}
