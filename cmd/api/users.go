package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/Crocmagnon/greenlight/internal/data"
	"github.com/Crocmagnon/greenlight/internal/validator"
)

//nolint:funlen
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	validate := validator.New()

	if data.ValidateUser(validate, user); !validate.Valid() {
		app.failedValidationResponse(w, r, validate.Errors)
		return
	}

	err = app.models.Users.Insert(user)

	switch {
	case errors.Is(err, data.ErrDuplicateEmail):
		validate.AddError("email", "a user with this email address already exists")
		app.failedValidationResponse(w, r, validate.Errors)

		return
	case err != nil:
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		mailData := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}
		err = app.mailer.Send(user.Email, "user_welcome.tmpl", mailData)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	validate := validator.New()

	if data.ValidateTokenPlaintext(validate, input.TokenPlaintext); !validate.Valid() {
		app.failedValidationResponse(w, r, validate.Errors)
		return
	}

	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenPlaintext)

	switch {
	case errors.Is(err, data.ErrRecordNotFound):
		validate.AddError("token", "invalid or expired activation token")
		app.failedValidationResponse(w, r, validate.Errors)
		return
	case err != nil:
		app.serverErrorResponse(w, r, err)
		return
	}

	user.Activated = true

	err = app.models.Users.Update(user)

	switch {
	case errors.Is(err, data.ErrEditConflict):
		app.editConflictResponse(w, r)
		return
	case err != nil:
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
