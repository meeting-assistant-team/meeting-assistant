package validator

import (
	"github.com/go-playground/validator/v10"
)

// CustomValidator implements echo.Validator using go-playground/validator
type CustomValidator struct {
	v *validator.Validate
}

// New creates a new CustomValidator instance
func New() *CustomValidator {
	v := validator.New()
	return &CustomValidator{v: v}
}

// Validate performs struct validation
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.v.Struct(i)
}
