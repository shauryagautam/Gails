package helpers

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"
	"time"
)

type FormBuilder struct {
	Model  any
	Errors map[string][]string
}

func FormFor(model any, action, method string, errors map[string][]string) template.HTML {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return template.HTML(fmt.Sprintf(`<form action="%s" method="%s"></form>`, action, method))
	}

	html := fmt.Sprintf(`<form action="%s" method="%s">`, action, method)
	html += `<input type="hidden" name="csrf_token" value="token">` // Placeholder

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // Skip unexported fields
			continue
		}
		if f.Name == "Model" || f.Name == "ID" || f.Name == "CreatedAt" || f.Name == "UpdatedAt" || f.Name == "DeletedAt" {
			continue // Skip framework fields
		}

		html += `<div class="form-group mb-3">`
		html += string(LabelFor(f.Name))

		inputType := InferInputType(v.Field(i).Interface())
		html += string(InputFor(model, f.Name, inputType, errors))
		html += `</div>`
	}

	html += string(SubmitButton("Submit", ""))
	html += `</form>`

	return template.HTML(html)
}

func InputFor(model any, fieldName string, inputType string, errors map[string][]string) template.HTML {
	val := getFieldValue(model, fieldName)
	errs := errors[fieldName]

	errorClass := ""
	if len(errs) > 0 {
		errorClass = " is-invalid"
	}

	html := fmt.Sprintf(`<input type="%s" name="%s" value="%v" class="form-control%s">`,
		inputType, fieldName, val, errorClass)

	if len(errs) > 0 {
		html += fmt.Sprintf(`<div class="invalid-feedback">%s</div>`, strings.Join(errs, ", "))
	}

	return template.HTML(html)
}

func LabelFor(fieldName string) template.HTML {
	label := humanize(fieldName)
	return template.HTML(fmt.Sprintf(`<label for="%s">%s</label>`, fieldName, label))
}

func TextareaFor(model any, fieldName string, errors map[string][]string) template.HTML {
	val := getFieldValue(model, fieldName)
	errs := errors[fieldName]

	errorClass := ""
	if len(errs) > 0 {
		errorClass = " is-invalid"
	}

	html := fmt.Sprintf(`<textarea name="%s" class="form-control%s">%v</textarea>`,
		fieldName, errorClass, val)

	if len(errs) > 0 {
		html += fmt.Sprintf(`<div class="invalid-feedback">%s</div>`, strings.Join(errs, ", "))
	}

	return template.HTML(html)
}

func CheckboxFor(model any, fieldName string) template.HTML {
	val := getFieldValue(model, fieldName)
	checked := ""
	if b, ok := val.(bool); ok && b {
		checked = " checked"
	}

	return template.HTML(fmt.Sprintf(`<input type="checkbox" name="%s" value="true"%s>`, fieldName, checked))
}

func SelectFor(model any, fieldName string, options map[string]string) template.HTML {
	selectedVal := fmt.Sprint(getFieldValue(model, fieldName))

	html := fmt.Sprintf(`<select name="%s" class="form-control">`, fieldName)
	for val, label := range options {
		selected := ""
		if val == selectedVal {
			selected = " selected"
		}
		html += fmt.Sprintf(`<option value="%s"%s>%s</option>`, val, selected, label)
	}
	html += `</select>`

	return template.HTML(html)
}

func SubmitButton(text, class string) template.HTML {
	if class == "" {
		class = "btn btn-primary"
	}
	return template.HTML(fmt.Sprintf(`<button type="submit" class="%s">%s</button>`, class, text))
}

// HiddenField generates a hidden input field.
func HiddenField(name, value string) template.HTML {
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, name, value))
}

func ErrorMessages(errors map[string][]string) template.HTML {
	if len(errors) == 0 {
		return ""
	}

	html := `<div class="alert alert-danger"><ul>`
	for field, errs := range errors {
		for _, err := range errs {
			html += fmt.Sprintf("<li>%s: %s</li>", humanize(field), err)
		}
	}
	html += `</ul></div>`

	return template.HTML(html)
}

// Internal helpers

func getFieldValue(model any, fieldName string) any {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ""
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}

	return field.Interface()
}

func humanize(s string) string {
	// Simple humanization: FirstName -> First Name
	var result []string
	start := 0
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, s[start:i])
			start = i
		}
	}
	result = append(result, s[start:])
	return strings.Join(result, " ")
}

func InferInputType(val any) string {
	switch val.(type) {
	case string:
		return "text"
	case bool:
		return "checkbox"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	case float32, float64:
		return "number"
	case time.Time:
		return "datetime-local"
	default:
		return "text"
	}
}
