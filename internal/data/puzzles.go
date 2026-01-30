package data

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Puzzle struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   int64     `json:"created_by"`
	Published   bool      `json:"published"`
	Version     int       `json:"-"`
}

type PuzzleModel struct {
	db *pgxpool.Pool
}

func (m PuzzleModel) Insert(puzzle *Puzzle) error {
	query := `
		INSERT INTO puzzles (title, description, content, created_by, published, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.db.QueryRow(
		ctx,
		query,
		puzzle.Title,
		puzzle.Description,
		puzzle.Content,
		puzzle.CreatedBy,
		puzzle.Published,
	).Scan(&puzzle.ID, &puzzle.CreatedAt, &puzzle.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (m PuzzleModel) GetByID(id int64) (*Puzzle, error) {
	query := `
		SELECT id, title, description, content, created_at, updated_at, created_by, published
		FROM puzzles
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := m.db.QueryRow(ctx, query, id)
	puzzle := &Puzzle{}
	err := row.Scan(
		&puzzle.ID,
		&puzzle.Title,
		&puzzle.Description,
		&puzzle.Content,
		&puzzle.CreatedAt,
		&puzzle.UpdatedAt,
		&puzzle.CreatedBy,
		&puzzle.Published,
	)
	if err != nil {
		return nil, err
	}
	return puzzle, nil
}

func (m PuzzleModel) Update(puzzle *Puzzle) error {
	query := `
		UPDATE puzzles
		SET title = $1, description = $2, content = $3, published = $4, updated_at = NOW(), version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version, updated_at`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.db.QueryRow(
		ctx,
		query,
		puzzle.Title,
		puzzle.Description,
		puzzle.Content,
		puzzle.Published,
		puzzle.ID,
		puzzle.Version,
	).Scan(&puzzle.Version, &puzzle.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEditConflict
		}
		return err
	}
	return nil
}

func (m PuzzleModel) Delete(id int64) error {
	query := `
		DELETE FROM puzzles
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.db.Exec(ctx, query, id)
	return err
}

func (m PuzzleModel) List() ([]*Puzzle, error) {
	// Implementation for listing all puzzles from the database
	return nil, nil
}
