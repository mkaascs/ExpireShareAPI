package response

import (
	"errors"
	domainErrors "expire-share/internal/domain/entities/errors"
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Response struct {
	Errors []string `json:"errors,omitempty"`
}

func Error(errorMessages ...string) Response {
	return Response{Errors: errorMessages}
}

func ValidationError(errs validator.ValidationErrors) Response {
	var validationErrorsMessages []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "min":
			validationErrorsMessages = append(validationErrorsMessages, fmt.Sprintf("%s must be greater or equal than %s", err.Field(), err.Param()))
		case "required":
			validationErrorsMessages = append(validationErrorsMessages, fmt.Sprintf("%s is required", err.Field()))
		case "url":
			validationErrorsMessages = append(validationErrorsMessages, fmt.Sprintf("%s is not a URL", err.Field()))
		default:
			validationErrorsMessages = append(validationErrorsMessages, fmt.Sprintf("%s is not a valid value", err.Field()))
		}
	}

	return Response{Errors: validationErrorsMessages}
}

func RenderError(w http.ResponseWriter, r *http.Request, statusCode int, errorMessage string) {
	render.Status(r, statusCode)
	render.JSON(w, r, Error(errorMessage))
}

func RenderValidationError(w http.ResponseWriter, r *http.Request, errors validator.ValidationErrors) {
	render.Status(r, http.StatusUnprocessableEntity)
	render.JSON(w, r, ValidationError(errors))
}

func RenderFileServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if errors.Is(err, domainErrors.ErrFileNotFound) {
		RenderError(w, r,
			http.StatusNotFound,
			"file with current alias not found")
		return true
	}

	if errors.Is(err, domainErrors.ErrFilePasswordRequired) {
		RenderError(w, r,
			http.StatusUnauthorized,
			"password for file is required")
		return true
	}

	if errors.Is(err, domainErrors.ErrFilePasswordInvalid) {
		RenderError(w, r,
			http.StatusForbidden,
			"invalid password")
		return true
	}

	if errors.Is(err, domainErrors.ErrFileSizeTooBig) {
		RenderError(w, r,
			http.StatusRequestEntityTooLarge,
			"file size is very big")
		return true
	}

	if errors.Is(err, domainErrors.ErrForbidden) {
		RenderError(w, r,
			http.StatusForbidden,
			"forbidden")
		return true
	}

	if errors.Is(err, domainErrors.ErrUploadLimitExceeded) {
		RenderError(w, r,
			http.StatusForbidden,
			"your upload limit exceeded. delete unnecessary files to upload new")
		return true
	}

	return false
}

func RenderAuthServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if errors.Is(err, domainErrors.ErrAccessTokenExpired) {
		RenderError(w, r,
			http.StatusUnauthorized,
			"access token is expired, refresh or login again")
		return true
	}

	if errors.Is(err, domainErrors.ErrAccessTokenRevoked) {
		RenderError(w, r,
			http.StatusUnauthorized,
			"access token is revoked, refresh or login again")
		return true
	}

	if errors.Is(err, domainErrors.ErrInvalidArgument) {
		RenderError(w, r,
			http.StatusUnprocessableEntity,
			err.Error())
		return true
	}

	if errors.Is(err, domainErrors.ErrInvalidAccessToken) {
		RenderError(w, r,
			http.StatusUnauthorized,
			"invalid access token")
		return true
	}

	if errors.Is(err, domainErrors.ErrUserAlreadyExists) {
		RenderError(w, r,
			http.StatusConflict,
			"user with this login or email already exists")
		return true
	}

	if errors.Is(err, domainErrors.ErrRoleNotExist) {
		RenderError(w, r,
			http.StatusNotFound,
			"user has not this role")
		return true
	}

	if errors.Is(err, domainErrors.ErrTooManyRequests) {
		RenderError(w, r,
			http.StatusTooManyRequests,
			"too many requests. try again later")
		return true
	}

	if errors.Is(err, domainErrors.ErrUserNotFound) {
		RenderError(w, r,
			http.StatusNotFound,
			"user not found")
		return true
	}

	if errors.Is(err, domainErrors.ErrInvalidCredentials) {
		RenderError(w, r,
			http.StatusUnauthorized,
			"login or password is invalid")
		return true
	}

	if errors.Is(err, domainErrors.ErrInvalidRefreshToken) {
		RenderError(w, r,
			http.StatusUnauthorized,
			"refresh token is invalid")
		return true
	}

	return false
}
