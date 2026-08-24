package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fagbenjaenoch/dorms-ng/internal/database/models"
	"github.com/fagbenjaenoch/dorms-ng/internal/dto"
	"github.com/fagbenjaenoch/dorms-ng/internal/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type InstitutionRepository struct {
	BaseRepository
	db *sql.DB
}

func NewInstitutionRepository(db *sql.DB, logger *zerolog.Logger) *InstitutionRepository {
	return &InstitutionRepository{
		BaseRepository: BaseRepository{
			Queries: models.New(db),
			Logger:  logger,
		},
		db: db,
	}
}

func (ir *InstitutionRepository) CheckInstitutionExists(ctx context.Context, institution dto.CreateInstitution) (bool, error) {

	institutionExists, err := ir.BaseRepository.Queries.CheckInstitutionExists(ctx, models.CheckInstitutionExistsParams{
		Name: institution.Name,
		City: institution.City,
	})
	if err != nil {
		return false, err
	}

	return institutionExists, nil
}

func (ir *InstitutionRepository) CreateInstitution(ctx context.Context, institution dto.CreateInstitution) (*models.Institution, error) {
	tx, err := ir.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	qtx := ir.BaseRepository.Queries.WithTx(tx)

	var i models.CreateInstitutionParams
	i.ID = uuid.New().String()
	i.Name = institution.Name
	i.Acronym = sql.NullString{String: strings.ToUpper(institution.Acronym), Valid: true}
	i.Latitude = institution.Latitude
	i.Longitude = institution.Longitude
	i.State = institution.State
	i.City = institution.City
	i.Slug = utils.GenerateSlug(institution.Acronym)
	i.Description = sql.NullString{String: institution.Description, Valid: true}

	ci, err := qtx.CreateInstitution(ctx, i)
	if err != nil {
		return nil, err
	}

	searchEntry := models.CreateSearchEntryParams{
		EntityID:   ci.ID,
		EntityType: "institution",
		Entity:     ci.Name,
		SearchText: sql.NullString{String: fmt.Sprintf("%s, %s, %s", ci.Name, ci.City, ci.Acronym.String), Valid: true},
		Slug:       ci.Slug,
		Address:    sql.NullString{String: fmt.Sprintf("%s, %s", ci.State, ci.City), Valid: true},
	}

	_, err = qtx.CreateSearchEntry(ctx, searchEntry)
	if err != nil {
		return nil, err
	}

	placeSearchEntry := models.CreatePlaceSearchEntryParams{
		PlaceID:   ci.ID,
		PlaceType: "institution",
		Name:      fmt.Sprintf("%s, %s", ci.Name, ci.City),
	}

	if _, err := qtx.CreatePlaceSearchEntry(ctx, placeSearchEntry); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &ci, nil
}

func (ir *InstitutionRepository) GetInstitution(ctx context.Context, slug string) (*models.Institution, error) {
	i, err := ir.BaseRepository.Queries.GetInstitutionBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func (ir *InstitutionRepository) GetAllInstitutions(ctx context.Context) ([]models.Institution, error) {
	institutions, err := ir.BaseRepository.Queries.GetAllInstitutions(ctx)
	if err != nil {
		return nil, err
	}

	return institutions, nil
}
