package handlers

import (
	"net/http"

	"github.com/fagbenjaenoch/dorms-ng/internal/dto"
	"github.com/fagbenjaenoch/dorms-ng/internal/server"
	"github.com/fagbenjaenoch/dorms-ng/internal/services"
	"github.com/fagbenjaenoch/dorms-ng/internal/utils"
	"github.com/go-chi/chi/v5"
)

type InstitutionHandler struct {
	BaseHandler
	service *services.InstitutionService
}

func NewInstitutionHandler(s *server.Server) InstitutionHandler {
	return InstitutionHandler{
		BaseHandler: BaseHandler{
			server: s,
		},
		service: services.NewInstitutionService(s.DB, s.Logger, s.NJS),
	}
}

func (h *InstitutionHandler) CreateInstitution(w http.ResponseWriter, r *http.Request) {
	institution, err := utils.GetValidatedPayloadFromRequest[dto.CreateInstitution](r.Context())
	if err != nil {
		msg := "failed to process request body"
		h.server.Logger.Err(err).Msg(msg)
		utils.WriteJSON(w, dto.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: msg,
			Payload: nil,
		})
		return
	}

	res, err := h.service.CreateInstitution(r.Context(), institution)
	if err != nil {
		msg := "failed to create institution"
		h.server.Logger.Err(err).Msg(msg)
		utils.WriteJSON(w, res)
		return
	}

	utils.WriteJSON(w, res)
}

func (h *InstitutionHandler) GetInstitution(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	h.server.Logger.Debug().Msgf("get institution: %s", slug)

	res, err := h.service.GetInstitution(r.Context(), slug)
	if err != nil {
		msg := "failed to get institution"
		h.server.Logger.Err(err).Msg(msg)
		utils.WriteJSON(w, res)
		return
	}

	utils.WriteJSON(w, res)
}

func (h *InstitutionHandler) GetAllInstitutions(w http.ResponseWriter, r *http.Request) {
	res, err := h.service.GetAllInstitutions(r.Context())
	if err != nil {
		msg := "failed to get all institutions"
		h.server.Logger.Err(err).Msg(msg)
		utils.WriteJSON(w, res)
		return
	}

	utils.WriteJSON(w, res)
}
