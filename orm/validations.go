package orm

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/shaurya/gails/framework/i18n"
)

var validate = validator.New()

func Validate(model any) map[string][]string {
	err := validate.Struct(model)
	if err == nil {
		return nil
	}

	errors := make(map[string][]string)
	for _, err := range err.(validator.ValidationErrors) {
		field := err.Field()
		tag := err.Tag()
		param := err.Param()

		// Key for i18n lookup: errors.validations.required
		i18nKey := fmt.Sprintf("errors.validations.%s", tag)

		msg := i18n.T(i18nKey, i18n.Vars{
			"field": i18n.T("models.fields."+field, nil),
			"param": param,
		})

		// Fallback if not translated
		if msg == i18nKey {
			msg = fmt.Sprintf("%s is invalid (%s)", field, tag)
		}

		errors[field] = append(errors[field], msg)
	}

	return errors
}

// HandleDBError catches common DB errors like unique constraints
func HandleDBError(err error) map[string][]string {
	if err == nil {
		return nil
	}

	msg := err.Error()
	errors := make(map[string][]string)

	// Check for unique constraint violation (Postgres code 23505)
	if strings.Contains(msg, "duplicate key value violates unique constraint") {
		// Try to parse the constraint name or field
		// Example: "duplicate key value violates unique constraint \"users_email_key\""
		parts := strings.Split(msg, "\"")
		if len(parts) >= 2 {
			constraint := parts[1]
			// convention: table_field_key
			fieldParts := strings.Split(constraint, "_")
			if len(fieldParts) >= 2 {
				field := fieldParts[1]
				field = strings.Title(field)

				i18nKey := "errors.validations.unique"
				errMsg := i18n.T(i18nKey, i18n.Vars{
					"field": i18n.T("models.fields."+field, nil),
				})
				if errMsg == i18nKey {
					errMsg = "has already been taken"
				}
				errors[field] = append(errors[field], errMsg)
			}
		}
	}

	if len(errors) == 0 {
		errors["base"] = []string{msg}
	}

	return errors
}
