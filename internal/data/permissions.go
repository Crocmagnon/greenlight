package data

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/lib/pq"
)

// Permissions holds permission codes as strings.
type Permissions []string

// Include checks whether the permissions table contains
// the given permission code. Typically used to check whether
// a user has a permission.
func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

// PermissionModel implements methods to query the database.
type PermissionModel struct {
	DB *sql.DB
}

// GetAllForUser returns all permission codes the user passed
// in parameter has.
func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
	SELECT p.code
	FROM permissions as p
	    JOIN users_permissions as up
	        ON up.permission_id = p.id
	WHERE up.user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying permissions: %w", err)
	}

	defer rows.Close() //nolint:errcheck

	var permissions Permissions

	for rows.Next() {
		var perm string

		err := rows.Scan(&perm) //nolint:govet // intentionally shadowing the var
		if err != nil {
			return nil, fmt.Errorf("scanning permission: %w", err)
		}

		permissions = append(permissions, perm)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating over permissions: %w", err)
	}

	return permissions, nil
}

// AddForUser gives permissions to the user.
func (m PermissionModel) AddForUser(userID int64, codes ...string) error {
	query := `
	INSERT INTO users_permissions
	SELECT $1, p.id FROM permissions as p WHERE p.code = ANY($2)`
	args := []any{userID, pq.Array(codes)}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("inserting permissions: %w", err)
	}

	return nil
}
