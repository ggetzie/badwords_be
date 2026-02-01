package main

import (
	"net/http"

	"github.com/ggetzie/badwords_be/internal/data"
	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	// Puzzle Routes
	router.HandlerFunc(http.MethodGet, "/v1/puzzles", app.listPuzzlesHandler)
	router.HandlerFunc(http.MethodPost, "/v1/puzzles", app.requirePermission(data.PuzzlesCreate, app.createPuzzleHandler))
	router.HandlerFunc(http.MethodGet, "/v1/puzzles/:id", app.getPuzzleByIdHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/puzzles/:id", app.requirePermission(data.PuzzlesUpdate, app.updatePuzzleHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/puzzles/:id", app.requirePermission(data.PuzzlesDelete, app.deletePuzzleHandler))

	// User Routes
	router.HandlerFunc(http.MethodGet, "/v1/user", app.requirePermission(data.UsersRead, app.getCurrentUserHandler))
	router.HandlerFunc(http.MethodPost, "/v1/users", app.requirePermission(data.UsersCreate, app.addUserHandler))
	router.HandlerFunc(http.MethodPut, "/v1/user/password", app.requireAuthenticatedUser(app.changePasswordHandler))
	router.HandlerFunc(http.MethodPut, "/v1/user", app.requireActivatedUser(app.updateUserHandler))

	// Authentication routes
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/logout", app.logoutHandler)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.logRequest(app.authenticate(router)))))

}
