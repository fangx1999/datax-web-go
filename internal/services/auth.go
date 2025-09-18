package services

import (
	"database/sql"
	"errors"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

// AuthService 封装用户认证、会话处理和授权逻辑。
// 它使用 gorilla/sessions cookie 存储来在请求之间持久化
// 已认证用户信息。数据库中的所有密码必须存储为 bcrypt 哈希。
// 在成功认证期间，会话会使用已登录用户和角色进行更新。
type AuthService struct {
	db    *sql.DB
	store *sessions.CookieStore
}

// NewAuthService 使用给定的数据库句柄和 cookie 存储创建新的 AuthService。
// 在将其传递给此构造函数之前，应该使用安全选项配置存储。
func NewAuthService(db *sql.DB, store *sessions.CookieStore) *AuthService {
	// 为内部使用配置会话选项
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days for internal use
		HttpOnly: true,
	}
	return &AuthService{db: db, store: store}
}

// Login 尝试认证给定的用户名和密码。如果成功，
// 用户和角色将存储在 HTTP 会话中。它返回
// 已认证的角色或失败时的错误。调用者负责
// 通过 sess.Save() 持久化会话。
func (a *AuthService) Login(w http.ResponseWriter, r *http.Request, username, password string) (string, error) {
	var hash, role string
	var disabled bool
	err := a.db.QueryRow("SELECT password, role, disabled FROM users WHERE username=?", username).Scan(&hash, &role, &disabled)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// 检查账户是否被禁用
	if disabled {
		return "", errors.New("account is disabled")
	}

	// 比较 bcrypt 哈希密码
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", errors.New("invalid credentials")
	}

	// 创建会话
	sess, err := a.store.Get(r, "sess")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		return "", errors.New("session error")
	}

	sess.Values["user"] = username
	sess.Values["role"] = role

	if err := sess.Save(r, w); err != nil {
		log.Printf("Error saving session: %v", err)
		return "", errors.New("session error")
	}

	return role, nil
}

// Logout 清除当前会话，有效登出用户。
func (a *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	sess, err := a.store.Get(r, "sess")
	if err != nil {
		log.Printf("Error getting session during logout: %v", err)
		return
	}

	// Clear session values
	delete(sess.Values, "user")
	delete(sess.Values, "role")

	// Invalidate session
	sess.Options.MaxAge = -1
	if err := sess.Save(r, w); err != nil {
		log.Printf("Error saving session during logout: %v", err)
	}
}

// CurrentUser 从会话中检索用户名和角色。如果没有用户
// 登录，两个返回值都将是空字符串。
func (a *AuthService) CurrentUser(r *http.Request) (username, role string) {
	sess, err := a.store.Get(r, "sess")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		return "", ""
	}

	u, ok := sess.Values["user"].(string)
	if !ok {
		return "", ""
	}

	roleVal, ok := sess.Values["role"].(string)
	if !ok {
		return "", ""
	}

	return u, roleVal
}

// HashPassword 生成给定纯文本密码的 bcrypt 哈希。使用的
// 成本是 bcrypt.DefaultCost。生成的字符串可以存储
// 直接存储在用户表中。
func HashPassword(pw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(bytes), err
}
