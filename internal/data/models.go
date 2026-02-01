package data

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Users       UserModel
	Permissions PermissionModel
	Tokens      TokenModel
	Puzzles     PuzzleModel
}

func NewModels(db *pgxpool.Pool) Models {
	return Models{
		Users:       UserModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Puzzles:     PuzzleModel{DB: db},
	}
}
