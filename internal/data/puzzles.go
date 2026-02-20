package data

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/ggetzie/badwords_be/internal/validator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClueData struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Clue   string `json:"clue"`
	Answer string `json:"answer"`
}

type PuzzleData struct {
	Across map[string]ClueData `json:"across"`
	Down   map[string]ClueData `json:"down"`
}

type Puzzle struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Content     PuzzleData `json:"content"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Author      User       `json:"author"`
	Published   bool       `json:"published"`
	Version     int        `json:"-"`
}

type PuzzleModel struct {
	DB *pgxpool.Pool
}

func ValidatePuzzle(v *validator.Validator, puzzle *Puzzle) {
	v.Check(puzzle.Title != "", "title", "must be provided")
	v.Check(utf8.RuneCountInString(puzzle.Title) <= 200, "title", "must not be more than 200 characters long")

	v.Check(puzzle.Description != "", "description", "must be provided")
	v.Check(utf8.RuneCountInString(puzzle.Description) <= 1000, "description", "must not be more than 1000 characters long")

	v.Check(puzzle.Width > 0, "width", "must be a positive integer")
	v.Check(puzzle.Height > 0, "height", "must be a positive integer")
}

func (m PuzzleModel) Insert(puzzle *Puzzle) error {
	query := `
		INSERT INTO puzzles (title, description, content, width, height, author_id, published, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(
		ctx,
		query,
		puzzle.Title,
		puzzle.Description,
		puzzle.Content,
		puzzle.Width,
		puzzle.Height,
		puzzle.Author.ID,
		puzzle.Published,
	).Scan(&puzzle.ID, &puzzle.CreatedAt, &puzzle.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (m PuzzleModel) GetByID(id int) (*Puzzle, error) {
	query := `
		SELECT p.id, p.title, p.description, p.content, p.width, p.height, p.created_at, p.updated_at, p.published, p.version, u.id, u.full_name, u.display_name, u.email
		FROM puzzles p
		INNER JOIN users u ON p.author_id = u.id
		WHERE p.id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := m.DB.QueryRow(ctx, query, id)

	puzzle := &Puzzle{}
	err := row.Scan(
		&puzzle.ID,
		&puzzle.Title,
		&puzzle.Description,
		&puzzle.Content,
		&puzzle.Width,
		&puzzle.Height,
		&puzzle.CreatedAt,
		&puzzle.UpdatedAt,
		&puzzle.Published,
		&puzzle.Version,
		&puzzle.Author.ID,
		&puzzle.Author.FullName,
		&puzzle.Author.DisplayName,
		&puzzle.Author.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return puzzle, nil
}

func (m PuzzleModel) Update(puzzle *Puzzle) error {
	query := `
		UPDATE puzzles
		SET title = $1, description = $2, content = $3, width = $4, height = $5, published = $6, updated_at = NOW(), version = version + 1
		WHERE id = $7 AND version = $8
		RETURNING version, updated_at`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(
		ctx,
		query,
		puzzle.Title,
		puzzle.Description,
		puzzle.Content,
		puzzle.Width,
		puzzle.Height,
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

func (m PuzzleModel) Delete(id int) error {
	query := `
		DELETE FROM puzzles
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.Exec(ctx, query, id)
	return err
}

func GetPublished(publishedVal string) (published1, published2 bool) {
	if publishedVal == "false" {
		return false, false
	}

	if publishedVal == "all" {
		return true, false
	}
	return true, true
}

func (m PuzzleModel) List(published1, published2 bool, filters Filters) ([]*Puzzle, Metadata, error) {
	query := `
		SELECT count(*) OVER(), p.id, p.title, p.description, p.content, p.width, p.height, p.created_at, p.updated_at, p.published, p.version, u.id, u.full_name, u.display_name, u.email
		FROM puzzles p
		INNER JOIN users u ON p.author_id = u.id
		WHERE p.published = $1 OR p.published = $2
		ORDER BY p.updated_at DESC
		LIMIT $3 OFFSET $4`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.Query(ctx, query, published1, published2, filters.PageSize, filters.offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	var puzzles []*Puzzle
	totalRecords := 0

	for rows.Next() {
		puzzle := &Puzzle{}
		err := rows.Scan(
			&totalRecords,
			&puzzle.ID,
			&puzzle.Title,
			&puzzle.Description,
			&puzzle.Content,
			&puzzle.Width,
			&puzzle.Height,
			&puzzle.CreatedAt,
			&puzzle.UpdatedAt,
			&puzzle.Published,
			&puzzle.Version,
			&puzzle.Author.ID,
			&puzzle.Author.FullName,
			&puzzle.Author.DisplayName,
			&puzzle.Author.Email,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		puzzles = append(puzzles, puzzle)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return puzzles, metadata, nil
}
