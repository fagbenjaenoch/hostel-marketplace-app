package middleware

import (
	"context"
	"net/http"
	"strconv"
)

type PaginationParams struct {
	Page   int
	Limit  int
	Offset int
}

const maxLimit = 20
const maxPriceLimit = 5000000

type PaginationKeyType string

const PaginationKey PaginationKeyType = "pagination"

func Pagination(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := &PaginationParams{
			Page:  1,
			Limit: 10,
		}

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
				params.Page = page
			}
		}

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit < maxLimit {
				params.Limit = limit
			}
		}

		params.Offset = (params.Page - 1) * params.Limit

		ctx := context.WithValue(r.Context(), PaginationKey, params)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetPaginationParams(ctx context.Context) *PaginationParams {
	if params, ok := ctx.Value(PaginationKey).(*PaginationParams); ok {
		return params
	}
	return &PaginationParams{
		Page:   1,
		Limit:  10,
		Offset: 0,
	}
}
