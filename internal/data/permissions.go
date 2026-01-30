package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Permissions []string

const (
	Superuser     = "000superuser"
	PuzzlesCreate = "puzzles:create"
	PuzzlesRead   = "puzzles:read"
	PuzzlesUpdate = "puzzles:update"
	PuzzlesDelete = "puzzles:delete"
	UsersRead     = "users:read"
	UsersUpdate   = "users:update"
	UsersDelete   = "users:delete"
)

var StandardPermissions = Permissions{
	PuzzlesCreate,
	PuzzlesRead,
	PuzzlesUpdate,
	PuzzlesDelete,
	UsersRead,
	UsersUpdate,
	UsersDelete,
}

func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}

type PermissionModel struct {
	DB *pgxpool.Pool
}

func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON permissions.id = users_permissions.permission_id
		WHERE users_permissions.user_id = $1
		ORDER BY permissions.code`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var permissions Permissions

	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return permissions, nil
}

func (m PermissionModel) AddForUser(userID int64, codes ...string) error {
	query := `
		INSERT INTO users_permissions
		SELECT $1, permissions.id
		FROM permissions
		WHERE permissions.code = ANY($2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.Exec(ctx, query, userID, codes)
	return err
}
