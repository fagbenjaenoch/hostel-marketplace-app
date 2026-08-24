package repositories

import (
	"context"
	"database/sql"

	"github.com/fagbenjaenoch/dorms-ng/internal/database/models"
	"github.com/fagbenjaenoch/dorms-ng/internal/dto"
	"github.com/fagbenjaenoch/dorms-ng/internal/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type NeighborhoodRepository struct {
	BaseRepository
	db *sql.DB
}

func NewNeighborhoodRepository(db *sql.DB, logger *zerolog.Logger) *NeighborhoodRepository {
	return &NeighborhoodRepository{
		BaseRepository: BaseRepository{
			Queries: models.New(db),
			Logger:  logger,
		},
		db: db,
	}
}

func (nr *NeighborhoodRepository) CheckNeighborhoodExists(ctx context.Context, neighborhood dto.CreateNeighborhood) (bool, error) {
	combinedName := utils.NormalizeNeighborhoodName(neighborhood)

	neighborhoodExists, err := nr.BaseRepository.Queries.CheckNeighborhoodExists(ctx, models.CheckNeighborhoodExistsParams{
		Name:        combinedName,
		City:        neighborhood.City,
		Institution: neighborhood.Institution,
	})
	if err != nil {
		return false, err
	}

	return neighborhoodExists, nil
}

func (nr *NeighborhoodRepository) CreateNeighborhood(ctx context.Context, neighborhood dto.CreateNeighborhood) (*models.Neighborhood, error) {
	tx, err := nr.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	qtx := nr.BaseRepository.Queries.WithTx(tx)

	address := utils.GetNeighborhoodAddress(neighborhood)

	combinedName := utils.NormalizeNeighborhoodName(neighborhood)

	var n models.CreateNeighborhoodParams
	n.ID = uuid.NewString()
	n.Name = combinedName
	n.Institution = neighborhood.Institution
	n.InstitutionID = neighborhood.InstitutionId
	n.City = neighborhood.City
	n.State = neighborhood.State

	cn, err := qtx.CreateNeighborhood(ctx, n)
	if err != nil {
		return nil, err
	}

	searchEntry := models.CreateSearchEntryParams{
		EntityID:   cn.ID,
		Entity:     cn.Name,
		EntityType: "neighborhood",
		SearchText: sql.NullString{String: combinedName, Valid: true},
		Address:    sql.NullString{String: address, Valid: true},
	}

	nr.Logger.Debug().Str("institution", cn.Institution).Msg("creating neighborhood search entry")

	if _, err := qtx.CreateSearchEntry(ctx, searchEntry); err != nil {
		return nil, err
	}

	placeSearchEntry := models.CreatePlaceSearchEntryParams{
		PlaceID:   cn.ID,
		PlaceType: "neighborhood",
		Name:      combinedName,
	}

	if _, err := qtx.CreatePlaceSearchEntry(ctx, placeSearchEntry); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &cn, nil
}

func (nr *NeighborhoodRepository) GetAllNeighborhoods(ctx context.Context) ([]models.Neighborhood, error) {
	neighborhoods, err := nr.BaseRepository.Queries.GetAllNeighborhoods(ctx)
	if err != nil {
		return nil, err
	}

	return neighborhoods, nil
}
