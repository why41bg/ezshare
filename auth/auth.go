package auth

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/log"
	"gopkg.in/boj/redistore.v1"
	"net/http"
	"os"
)

type Users struct {
	Lookup      map[string]string
	store       sessions.Store
	sessionTime int
}

type UserInfo struct {
	name string
	pass string
}

type Response struct {
	Message string `json:"message"`
}

// LoadUsersFile loads the user information from the file specified by the path.
func LoadUsersFile(path string, secret []byte, sessionTimeout int) (*Users, error) {
	store, err := redistore.NewRediStore(10, "tcp", ":6379", "123456", secret)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to redis.")
		return nil, err
	}
	users := &Users{
		Lookup:      map[string]string{},
		store:       store,
		sessionTime: sessionTimeout,
	}

	fd, err := os.Open(path)
	defer func(fd *os.File) {
		err := fd.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close file descriptor")
		}
	}(fd)
	csvReader := csv.NewReader(fd)
	csvReader.Comma = ':'
	csvReader.Comment = '#'
	csvReader.TrimLeadingSpace = true
	UserInfos, err := csvReader.ReadAll()
	if err != nil {
		log.Error().Err(err).Msg("Failed to read users file")
		return nil, err
	}
	for _, info := range UserInfos {
		if len(info) != 2 {
			return nil, errors.New("malformed users file")
		}
		users.Lookup[info[0]] = info[1]
	}

	log.Debug().Msg(fmt.Sprintf("Loaded %d users", len(users.Lookup)))
	return users, nil
}

// CurrentUser according to the cookie in the request to get the session and then
// to get the username. If the user not authenticated, "guest" will return.
func (u *Users) CurrentUser(r *http.Request) (string, bool) {
	session, err := u.store.Get(r, "user")
	session.Options.MaxAge = u.sessionTime
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
			Message: "Login system error, please try again",
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

// Authenticate will check if the user and password are correct. If so,
// it will create a new session which stored the user information. And
// then save the session to the store with response 200. If the password is
// not correct, it will return 401.
func (u *Users) Authenticate(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("user")
	pass := r.FormValue("pass")
	if !u.validateUser(user, pass) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(&Response{
			Message: fmt.Sprintf("User %s not found or password not match", user),
		})
		return
	}

	session := sessions.NewSession(u.store, "user")
	session.IsNew = true
	session.Options.MaxAge = u.sessionTime
	session.Values["user"] = user
	if err := u.store.Save(r, w, session); err != nil {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(&Response{
			Message: "Login system error, please try again",
		})
		return
	}
	w.WriteHeader(200)
}
