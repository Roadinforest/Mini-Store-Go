package validation

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"

	"mini-store-go/backend/internal/apperror"
)

type FieldError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value,omitempty"`
}

type Validator struct {
	engine *validator.Validate
}

func New() *Validator {
	return &Validator{
		engine: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (v *Validator) Validate(input interface{}) error {
	if err := v.engine.Struct(input); err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return apperror.Wrap(apperror.CodeValidation, "validation failed", err)
		}

		fields := make([]FieldError, 0, len(validationErrors))
		for _, item := range validationErrors {
			fields = append(fields, FieldError{
				Field: strings.ToLower(item.Field()),
				Tag:   item.Tag(),
				Value: fmt.Sprint(item.Value()),
			})
		}

		return apperror.WithDetails(
			apperror.Wrap(apperror.CodeValidation, "validation failed", err),
			fields,
		)
	}
	return nil
}
