package middleware

import (
	"context"
	"net/http"
	"strconv"
)

type HostelFilterParams struct {
	SortBy     string
	MinPrice   int
	MaxPrice   int
	IsVerified bool
}

type HostelFilterKeyType string

const HostelFilterKey HostelFilterKeyType = "hostelFilters"

func HostelFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := &HostelFilterParams{
			SortBy:   "price-asc",
			MinPrice: 0,
			MaxPrice: maxPriceLimit,
		}

		if sortByStr := r.URL.Query().Get("sortBy"); sortByStr != "" {
			params.SortBy = sortByStr
		}
		if minPriceStr := r.URL.Query().Get("minPrice"); minPriceStr != "" {
			if minPrice, err := strconv.Atoi(minPriceStr); err == nil && minPrice >= 0 {
				params.MinPrice = minPrice
			}
		}
		if maxPriceStr := r.URL.Query().Get("maxPrice"); maxPriceStr != "" {
			if maxPrice, err := strconv.Atoi(maxPriceStr); err == nil && maxPrice > 0 && maxPrice <= maxPriceLimit {
				params.MaxPrice = maxPrice
			}
		}
		if isVerifiedStr := r.URL.Query().Get("isVerified"); isVerifiedStr != "" {
			params.IsVerified = isVerifiedStr == "true"
		}

		ctx := context.WithValue(r.Context(), HostelFilterKey, params)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetHostelFilterParams(ctx context.Context) *HostelFilterParams {
	if params, ok := ctx.Value(HostelFilterKey).(*HostelFilterParams); ok {
		return params
	}
	return &HostelFilterParams{
		SortBy:     "price-asc",
		MinPrice:   0,
		MaxPrice:   5000000,
		IsVerified: false,
	}
}
