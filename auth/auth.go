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

// LoadUsersFile 从指定的文件中读取用户信息
func LoadUsersFile(path string, secret []byte, sessionTimeout int) (*Users, error) {
	// 1. 初始化一个存储用户信息的结构体
	users := &Users{
		Lookup:      map[string]string{},
		store:       sessions.NewCookieStore(secret),
		sessionTime: sessionTimeout,
	}

	// 2. 读取用户信息，保存到Users结构体中
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
	for _, userInfo := range userInfos {
		users.Lookup[userInfo.name] = userInfo.passwd
	}
	log.Debug().Msg(fmt.Sprintf("Loaded %d users", len(users.Lookup)))
	return users, nil
}

// CurrentUser will try to get the user info from the session of
// the current request. If the session does not exist, it will
// create a new session and return "guest" as the user info.
func (u *Users) CurrentUser(r *http.Request) (string, bool) {
	session, err := u.store.Get(r, "user")
	if err != nil {
		log.Error().Err(err).Any("request", r).Msg("Failed to get the session from request or create a new session")
		return "guest", false
	}
	if username, ok := session.Values["user"].(string); ok {
		log.Debug().Str("user", username).Msg("Got username from session")
		return username, ok
	}
	log.Debug().Str("user", "guest").Msg("Failed to get username from session")
	return "guest", false
}

// Logout log out the user included in the request by creating a new session
// and overwriting the old session.
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

// validateUser check if the user and password are correct.
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

// Authenticate will check if the user and password are correct. If they are,
// it will create a new session and store the user info in the session. And
// then save the session to the store with response 200. If the password is
// not correct, it will return 401.
func (u *Users) Authenticate(w http.ResponseWriter, r *http.Request) {
	// 1. Get username and password from the request.
	user := r.FormValue("user")
	pass := r.FormValue("pass")
	if !u.validateUser(user, pass) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(&Response{
			Message: "Could not authenticate",
		})
		return
	}

	// 2. Create a new session and store the user info in the session.
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
