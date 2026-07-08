package validate

import (
	"net/http"

	"taskflow/internal/api"
	"taskflow/internal/models"
	"taskflow/internal/scheduler"
)

// Validator uses a CronParser to validate cron expressions and compute next
// scheduled times.
type Validator struct {
	parser *scheduler.CronParser
}

// NewValidator returns a new Validator backed by the given CronParser.
func NewValidator(parser *scheduler.CronParser) *Validator {
	return &Validator{parser: parser}
}

// Validate parses expr and returns a ValidationResult.
//
//   - On success: ValidationResult{Valid: true, NextTimes: <next 5 times>}
//   - On failure: ValidationResult{Valid: false, Error: &ParseError{...}}
func (v *Validator) Validate(expr string) *models.ValidationResult {
	if _, err := v.parser.ParseCron(expr); err != nil {
		return &models.ValidationResult{
			Valid: false,
			Error: &models.ParseError{
				Field:    "cron_expr",
				Position: 0,
				Message:  err.Error(),
			},
		}
	}

	times, err := v.parser.ComputeNextTimes(expr, 5)
	if err != nil {
		// ParseCron already succeeded, so this path should not be reachable in
		// practice; treat it as a parse-level error to be safe.
		return &models.ValidationResult{
			Valid: false,
			Error: &models.ParseError{
				Field:    "cron_expr",
				Position: 0,
				Message:  err.Error(),
			},
		}
	}

	return &models.ValidationResult{
		Valid:     true,
		NextTimes: times,
	}
}

// Handler serves GET /api/v1/schedule/validate?expr=...
//
// It reads the "expr" query parameter, calls Validate, and writes:
//   - 200 with the ValidationResult JSON when the expression is valid
//   - 422 with the ValidationResult JSON when the expression is invalid
func (v *Validator) Handler(w http.ResponseWriter, r *http.Request) {
	expr := r.URL.Query().Get("expr")
	result := v.Validate(expr)

	if result.Valid {
		api.WriteJSON(w, http.StatusOK, result)
	} else {
		api.WriteJSON(w, http.StatusUnprocessableEntity, result)
	}
}
