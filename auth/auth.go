package auth

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
)

type Users struct {
	Lookup      map[string]string
	store       sessions.Store
	sessionTime int
}

type UserInfo struct {
	name   string
	passwd string
}

type Response struct {
	Message string `json:"message"`
}

func read(r io.Reader) ([]UserInfo, error) {
	// 配置用户文件读取器
	csvReader := csv.NewReader(r)
	csvReader.Comma = ':'
	csvReader.Comment = '#'
	csvReader.TrimLeadingSpace = true

	// 读取全部用户信息
	UserInfos, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	var ret []UserInfo
	for _, info := range UserInfos {
		if len(info) != 2 {
			return nil, errors.New("malformed users file")
		}
		ret = append(ret, UserInfo{name: info[0], passwd: info[1]})
	}

	// 返回用户信息列表
	return ret, nil
}

// LoadUsersFile 从指定的文件中读取用户信息，存储到 Users 结构体中
func LoadUsersFile(path string, secret []byte, sessionTimeout int) (*Users, error) {
	// 初始化一个存储用户信息的结构体
	users := &Users{
		Lookup:      map[string]string{},
		store:       sessions.NewCookieStore(secret),
		sessionTime: sessionTimeout,
	}

	// 读取存储用户信息的文件
	fd, err := os.Open(path)
	defer func(fd *os.File) {
		err := fd.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close users file")
		}
	}(fd)
	userInfos, err := read(fd)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read users file")
		return nil, err
	}

	// 将用户信息保存到 Users 结构体中
	for _, userInfo := range userInfos {
		users.Lookup[userInfo.name] = userInfo.passwd
	}

	log.Info().Msg(fmt.Sprintf("Loaded %d users", len(users.Lookup)))
	return users, nil
}

// CurrentUser 获取当前请求 session 中包含的用户信息，如果没有则返回 guest
func (u *Users) CurrentUser(r *http.Request) (string, bool) {
	session, err := u.store.Get(r, "user")
	if err != nil {
		log.Error().Err(err).Any("request", r).Msg("Failed to get the session from request or create a new session")
		return "guest", false
	}
	if username, ok := session.Values["user"].(string); ok {
		return username, ok
	}
	log.Info().Msg("Failed to get username from session")
	return "guest", false
}

// Logout 注销用户，通过创建一个新的 session，覆盖原有的 session，达到注销的目的
func (u *Users) Logout(w http.ResponseWriter, r *http.Request) {
	session := sessions.NewSession(u.store, "user")
	session.IsNew = true
	if err := u.store.Save(r, w, session); err != nil {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(&Response{
			Message: err.Error(),
		})
		return
	}
	w.WriteHeader(200)
}

// validateUser 验证用户信息是否正确，即是否存在于 Users 结构体中
func (u *Users) validateUser(user, passwd string) bool {
	pwd, ok := u.Lookup[user]
	if !ok {
		log.Info().Str("user", user).Msg("User not found")
		return false
	}
	if pwd != passwd {
		log.Info().Str("user", user).Str("password", passwd).Msg("Password not match")
		return false
	}
	return true
}

// Authenticate 验证用户信息是否正确，如果正确则创建一个新的 session
// 并将这个新的 session 与用户请求关联起来，存储在 Users 结构体中
// 后续来自这个用户的请求都会携带这个 session，从而实现会话管理
func (u *Users) Authenticate(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("user")
	pass := r.FormValue("pass")

	if !u.validateUser(user, pass) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(&Response{
			Message: "Could not authenticate",
		})
		return
	}

	// 用户信息验证通过，创建一个新的 session，用于保存会话信息
	session := sessions.NewSession(u.store, "user")
	session.IsNew = true
	session.Options.MaxAge = u.sessionTime
	session.Values["user"] = user
	if err := u.store.Save(r, w, session); err != nil {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(&Response{
			Message: err.Error(),
		})
		return
	}
	w.WriteHeader(200)
}
