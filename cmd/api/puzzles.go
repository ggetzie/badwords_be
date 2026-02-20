package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ggetzie/badwords_be/internal/data"
	"github.com/ggetzie/badwords_be/internal/validator"
)

func (app *application) listPuzzlesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Published string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Published = app.readString(qs, "published", "true")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-updated_at")

	published1, published2 := data.GetPublished(input.Published)

	user := app.contextGetUser(r)
	permissions, err := app.models.Permissions.GetAllForUser(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !(published1 && published2) && !permissions.Include(data.PuzzlesUpdate) {
		published1 = true
		published2 = true
	}
	puzzles, metadata, err := app.models.Puzzles.List(published1, published2, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"puzzles": puzzles, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getPuzzleByIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	puzzle, err := app.models.Puzzles.GetByID(id)
	if err != nil {
		switch {
		case err == data.ErrRecordNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user := app.contextGetUser(r)
	permissions, err := app.models.Permissions.GetAllForUser(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !puzzle.Published && !permissions.Include(data.PuzzlesUpdate) {
		app.notFoundResponse(w, r)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"puzzle": puzzle}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) createPuzzleHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string          `json:"title"`
		Description string          `json:"description"`
		Content     data.PuzzleData `json:"content"`
		Published   bool            `json:"published"`
		Width       int             `json:"width"`
		Height      int             `json:"height"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.logger.Debug(fmt.Sprintf("Error reading input: %v", r))
		app.badRequestResponse(w, r, err)
		return
	}
	user := app.contextGetUser(r)
	puzzle := &data.Puzzle{
		Title:       input.Title,
		Description: input.Description,
		Content:     input.Content,
		Published:   input.Published,
		Width:       input.Width,
		Height:      input.Height,
		Author:      *user,
	}

	v := validator.New()
	data.ValidatePuzzle(v, puzzle)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Puzzles.Insert(puzzle)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/puzzles/%d", puzzle.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"puzzle": puzzle}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updatePuzzleHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	puzzle, err := app.models.Puzzles.GetByID(id)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Title       *string          `json:"title"`
		Description *string          `json:"description"`
		Content     *data.PuzzleData `json:"content"`
		Width       *int             `json:"width"`
		Height      *int             `json:"height"`
		Published   *bool            `json:"published"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Title != nil {
		puzzle.Title = *input.Title
	}
	if input.Description != nil {
		puzzle.Description = *input.Description
	}
	if input.Content != nil {
		puzzle.Content = *input.Content
	}
	if input.Published != nil {
		puzzle.Published = *input.Published
	}
	if input.Width != nil {
		puzzle.Width = *input.Width
	}
	if input.Height != nil {
		puzzle.Height = *input.Height
	}
	v := validator.New()
	data.ValidatePuzzle(v, puzzle)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Puzzles.Update(puzzle)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"puzzle": puzzle}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) deletePuzzleHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Puzzles.Delete(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "puzzle successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
