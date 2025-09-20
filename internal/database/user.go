package database

import (
	"database/sql"
	"fmt"

	"com.duole/datax-web-go/internal/entities"
)

// UserDB 用户数据库操作（空结构体）
type UserDB struct{}

// List 获取用户列表
func (d *UserDB) List() ([]entities.User, error) {
	query := `
		SELECT 
		    u.id, 
		    u.username, 
		    u.role, 
		    u.disabled,
		    COALESCE(uc.username, '系统') as created_by_name,
		    COALESCE(uu.username, '系统') as updated_by_name,
		    u.created_at
		FROM users u
		LEFT JOIN users uc ON u.created_by = uc.id
		LEFT JOIN users uu ON u.updated_by = uu.id
		ORDER BY u.id DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}
	defer rows.Close()

	var users []entities.User
	for rows.Next() {
		var user entities.User
		err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.Disabled,
			&user.CreatedByName, &user.UpdatedByName, &user.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描用户数据失败: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetByID 根据ID获取用户
func (d *UserDB) GetByID(id int) (*entities.User, error) {
	query := `SELECT id,username,role,disabled FROM users WHERE id=?`

	var user entities.User
	err := db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Role, &user.Disabled)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}

// Create 创建用户
func (d *UserDB) Create(user *entities.User) error {
	query := `INSERT INTO users(username,password,role,created_by,updated_by) VALUES(?,?,?,?,?)`

	_, err := db.Exec(query, user.Username, user.Password, user.Role, user.CreatedBy, user.UpdatedBy)
	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	return nil
}

// Update 更新用户
func (d *UserDB) Toggle(id, updateBy int) error {
	query := `UPDATE users SET disabled=1-disabled, updated_by=? WHERE id=?`

	result, err := db.Exec(query, updateBy, id)
	if err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查更新结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("用户不存在")
	}

	return nil
}

// GetByUsername 根据用户名获取用户信息（用于认证）
func (d *UserDB) GetByUsername(username string) (*entities.User, error) {
	query := `SELECT id, username, password, role, disabled FROM users WHERE username=?`

	var user entities.User
	err := db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.Role, &user.Disabled)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}
