package utils

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/fagbenjaenoch/dorms-ng/internal/dto"
)

type QueryParam string

func (q QueryParam) String() string {
	return string(q)
}

const (
	SearchParam   QueryParam = "q"
	AreaParam     QueryParam = "areaId"
	AreaTypeParam QueryParam = "areaType"
)

func WriteJSON(w http.ResponseWriter, response dto.StructuredResponse) {
	responseJSON, err := json.Marshal(response)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Status)
	_, _ = w.Write(responseJSON)
}

func DecodeJSONBody(w http.ResponseWriter, r *http.Request, body any) error {
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		WriteJSON(w, dto.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "could not process message body",
		})
		return err
	}
	defer r.Body.Close()

	return nil
}

func GetValidatedPayloadFromRequest[T any](ctx context.Context) (T, error) {
	payload, ok := ctx.Value(ValidatedPayloadKey).(T)
	if !ok {
		return payload, errors.New("invalid payload")
	}
	return payload, nil
}

func SkipTelemetry(r *http.Request) bool {
	if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
		return false
	}

	if strings.HasPrefix(r.URL.Path, "/docs/") {
		return false
	}

	return true
}
