package utils

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	pkgError "errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/fagbenjaenoch/dorms-ng/internal/config"
	"github.com/go-playground/validator/v10"
	"github.com/sqlc-dev/pqtype"
)

type contextKey string

const (
	ValidatedPayloadKey contextKey = "validated_payload"
	JWTClaimsKey        contextKey = "jwt_claims"
	PasswordProviderKey string     = "password"
)

func IsProduction() bool {
	config := config.GetGlobalConfig()
	return config.Primary.Env == "production"
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Errors []ValidationError `json:"errors"`
}

func FormatValidationErrors(err error) ErrorResponse {
	var errors []ValidationError
	var validationErrors validator.ValidationErrors

	if pkgError.As(err, &validationErrors) {
		for _, e := range validationErrors {
			var message string

			// Create human-readable messages based on the tag
			switch e.Tag() {
			case "required":
				message = fmt.Sprintf("%s is required", strings.ToLower(e.Field()))
			case "email":
				message = fmt.Sprintf("%s must be a valid email", strings.ToLower(e.Field()))
			case "min":
				message = fmt.Sprintf("%s must be at least %s characters", strings.ToLower(e.Field()), e.Param())
			case "max":
				message = fmt.Sprintf("%s must be at most %s characters", strings.ToLower(e.Field()), e.Param())
			case "oneof":
				message = fmt.Sprintf("%s must be one of: %q", strings.ToLower(e.Field()), e.Param())
			default:
				message = fmt.Sprintf("%s is invalid", strings.ToLower(e.Field()))
			}

			errors = append(errors, ValidationError{
				Field:   strings.ToLower(e.Field()),
				Message: message,
			})
		}
	}

	return ErrorResponse{
		Errors: errors,
	}
}

func GeneratePresignedURLKey(entityName, entityType, fileName string) string {
	nameHash := GenerateHash(fileName)
	return fmt.Sprintf("%ss/%s/%s", strings.ToLower(entityType), strings.ToLower(strings.ReplaceAll(entityName, " ", "-")), nameHash)
}

func GenerateSlug(stringInput ...string) string {
	var slug string
	for _, s := range stringInput {
		slug += strings.ToLower(strings.ReplaceAll(s, " ", "-")) + "-"
	}
	return strings.TrimRight(slug, "-")
}

func GenerateHash(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	return base64.RawURLEncoding.Strict().EncodeToString(h.Sum(nil))
}

func StringsToNullRawMessage(strings []string) (pqtype.NullRawMessage, error) {
	var result pqtype.NullRawMessage

	if len(strings) == 0 {
		result.Valid = false
		return result, nil
	}

	// Marshal the array to JSON
	jsonData, err := json.Marshal(strings)
	if err != nil {
		return result, err
	}

	result.RawMessage = jsonData
	result.Valid = true

	return result, nil
}

func NullRawMessageToStrings(nrm pqtype.NullRawMessage) []string {
	if !nrm.Valid {
		return nil
	}

	var result []string
	if err := json.Unmarshal(nrm.RawMessage, &result); err != nil {
		return nil
	}

	return result
}

func SetStructDefaults(s interface{}) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}

	v = v.Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := t.Field(i).Tag.Get("default")

		if tag != "" && field.IsZero() {
			switch field.Kind() {
			case reflect.String:
				field.SetString(tag)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if i, err := strconv.ParseInt(tag, 10, 64); err == nil {
					field.SetInt(i)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if u, err := strconv.ParseUint(tag, 10, 64); err == nil {
					field.SetUint(u)
				}
			case reflect.Float32, reflect.Float64:
				if f, err := strconv.ParseFloat(tag, 64); err == nil {
					field.SetFloat(f)
				}
			}
		}
	}
}
