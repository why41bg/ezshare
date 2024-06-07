package auth

import (
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/log"
	"net/http/httptest"
	"os"
	"testing"
)

func TestReader(t *testing.T) {
	// 打开文件，拿到 fd
	fd, err := os.Open("../users")
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Error().Err(err).Msg("close fd failed")
		}
	}(fd)
	if err != nil {
		log.Error().Err(err).Msg("open fd failed")
	}

	// 读取用户信息
	users, err := read(fd)
	if err != nil {
		log.Error().Err(err).Msg("read fd failed")
	}
	for idx, user := range users {
		log.Info().Str("name", user.name).Str("pass", user.pass).Msg(fmt.Sprintf("user info #%d", idx))
	}
}

func TestLoadUsersFile(t *testing.T) {
	loadUserFile("../users")
	log.Info().Msg("load users file success")
}

func TestUsers_CurrentUser(t *testing.T) {
	// 创建一个 AuthUsers 实例
	users := &Users{
		Lookup:      map[string]string{"testuser": "testpassword"},
		store:       sessions.NewCookieStore([]byte("secret")),
		sessionTime: 10,
	}

	// 创建一个 http.Request 实例
	req := httptest.NewRequest("GET", "http://localhost:8080", nil)

	// 创建一个 session，并在其中设置用户信息
	session, _ := users.store.Get(req, "user")
	session.Values["user"] = "user"
	_ = session.Save(req, httptest.NewRecorder())

	username := users.CurrentUser(req)
	log.Info().Str("username", username).Msg("current user")
	passwd := users.Lookup[username]
	log.Info().Str("pass", passwd).Msg("current user pass")
}

func loadUserFile(path string) (*Users, error) {
	users, err := LoadUsersFile(path, []byte("secret"), 10)
	if err != nil {
		log.Error().Err(err).Msg("load users file failed")
		return nil, err
	}
	return users, nil
}

func TestUsers_validateUser(t *testing.T) {
	// 创建一个 AuthUsers 实例
	users := &Users{
		Lookup:      map[string]string{"testuser": "testpassword"},
		store:       sessions.NewCookieStore([]byte("secret")),
		sessionTime: 10,
	}

	exist := users.validateUser("testuser", "testpassword")
	if exist {
		log.Info().Msg("user exist")
	} else {
		log.Info().Msg("user not exist")
	}
}

func TestUsers_Authenticate(t *testing.T) {
	// 创建一个 AuthUsers 实例
	users := &Users{
		Lookup:      map[string]string{"testuser": "testpassword"},
		store:       sessions.NewCookieStore([]byte("secret")),
		sessionTime: 10,
	}

	// 创建一个 http.Request 实例
	req := httptest.NewRequest("POST", "http://localhost:8080", nil)
	req.Form = map[string][]string{
		"user": {"testuser"},
		"pass": {"testpassword"},
	}

	// 创建一个 session 并保存
	session, _ := users.store.Get(req, "user")
	session.Values["user"] = req.FormValue("user")
	_ = session.Save(req, httptest.NewRecorder())

	users.Authenticate(httptest.NewRecorder(), req)
	log.Info().Str("user", users.CurrentUser(req)).Str("pass", users.Lookup[users.CurrentUser(req)]).Msg("authenticate success")
}
