package main

import (
	"errors"
	"net/http"

	"github.com/Crocmagnon/greenlight/internal/data"
	"github.com/Crocmagnon/greenlight/internal/validator"
)

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

	err = app.mailer.Send(user.Email, "user_welcome.tmpl", user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}