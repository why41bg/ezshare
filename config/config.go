package config

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/ezshare/server/config/ip"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	prefix = "ezshare"
	files  = []string{"ezshare.config.development", "ezshare.config"}
)

type Config struct {
	ExternalIP []string `split_words:"true"`

	ServerAddress         string `default:":5050" split_words:"true"`
	Secret                []byte `split_words:"true"`
	SessionTimeoutSeconds int    `default:"0" split_words:"true"`

	TurnAddress    string      `default:":3478" required:"true" split_words:"true"`
	TurnPort       string      `ignored:"true"`
	TurnPortRange  string      `split_words:"true"`
	TurnRealm      string      `default:"ezshare" split_words:"true"`
	TurnIPProvider ip.Provider `ignored:"true"`

	AuthMode                 string            `default:"turn" split_words:"true"`
	CorsAllowedOrigins       []string          `split_words:"true"`
	CheckOrigin              func(string) bool `ignored:"true" json:"-"`
	UsersFile                string            `split_words:"true"`
	CloseRoomWhenOwnerLeaves bool              `default:"true" split_words:"true"`
	Version                  string            `default:"1.0"`
}

// LoadConfig firstly load config file to environment variables, then parse environment variables to generate Config.
func LoadConfig() (*Config, error) {
	log.Debug().Msg("Begin to load config file")
	dir, err := workOrExecAbsDir()
	if err != nil {
		return nil, err
	}
	for _, file := range configFilePath(dir) {
		_, existErr := os.Stat(file)
		if existErr == nil {
			if err := godotenv.Load(file); err != nil {
				return nil, err
			}
			log.Debug().Str("file", file).Msg("Config file loaded")
			break
		} else {
			log.Debug().Str("file", file).Msg("Config file not exist")
			continue
		}
	}
	log.Debug().Msg("Config file loaded")

	log.Debug().Msg("Begin to process env config")
	config := &Config{}
	if err := envconfig.Process(prefix, config); err != nil {
		return nil, err
	}
	log.Debug().Msg("Env config processed")

	if len(config.Secret) == 0 {
		log.Debug().Msg("Secret is empty, begin to generate random secret")
		config.Secret = make([]byte, 32)
		if _, err := rand.Read(config.Secret); err != nil {
			return nil, err
		}
		log.Debug().Msg("Random secret generated")
	}

	log.Debug().Msg("Begin to generate CORS check function")
	var compiledAllowedOrigins []*regexp.Regexp
	for _, origin := range config.CorsAllowedOrigins {
		compiled, err := regexp.Compile(origin)
		if err != nil {
			return nil, err
		}
		compiledAllowedOrigins = append(compiledAllowedOrigins, compiled)
	}
	config.CheckOrigin = func(origin string) bool {
		if origin == "" {
			return true
		}
		for _, compiled := range compiledAllowedOrigins {
			if compiled.Match([]byte(strings.ToLower(origin))) {
				return true
			}
		}
		return false
	}
	log.Debug().Msg("CORS check function generated")

	log.Debug().Msg("Begin to generate IP provider")
	turnIPProvider, err := parseIPProvider(config.ExternalIP)
	if err != nil {
		return nil, err
	}
	config.TurnIPProvider = turnIPProvider
	config.TurnPort = strings.Split(config.TurnAddress, ":")[1]
	log.Debug().Msg("IP provider generated")

	log.Debug().Msg("Begin to parse port range")
	minport, maxport, err := config.parsePortRange()
	if err != nil {
		return nil, err
	} else if minport == 0 || maxport == 0 || minport > maxport {
		return nil, errors.New("invalid port range")
	} else if (maxport - minport) < 40 {
		return nil, errors.New("port range too small")
	}
	log.Debug().Msg("Port range parsed")

	log.Info().Msg("All config loaded")
	return config, nil
}

// workOrExecAbsDir returns the working directory or the directory of the executable file, depending on the
// current running mode. if Dev, return working directory. if Prod, return the directory of the executable file.
func workOrExecAbsDir() (string, error) {
	if CurrentMode() == Dev {
		log.Debug().Msg("Use work dir")
		return filepath.Abs(".")
	}
	log.Debug().Msg("Use executable dir")
	return execDir()
}

// execDir returns the directory of the executable file. If the program is running with go run, it returns the
// temporary directory of go run. If the program is running with go build, it returns the directory of the
// executable file.
func execDir() (string, error) {
	path, err := os.Executable()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get executable dir path")
		return "", err
	}
	return filepath.Dir(path), nil
}

// configFilePath generates the absolute path of the config file, the file name is fixed to the file name
// in the files slice.
func configFilePath(dir string) []string {
	var configFilePaths []string
	for _, file := range files {
		configFilePaths = append(configFilePaths, filepath.Join(dir, file))
	}
	return configFilePaths
}

// parsePortRange parses the port range from the environment variable TurnPortRange, and returns the min port,
// max port, and an error. If the environment variable is not set, it returns 0, 0, nil.
func (c *Config) parsePortRange() (uint16, uint16, error) {
	if c.TurnPortRange == "" {
		return 0, 0, errors.New("port range not set")
	}

	parts := strings.Split(c.TurnPortRange, ":")
	if len(parts) != 2 {
		return 0, 0, errors.New("must include one colon")
	}
	stringMin := parts[0]
	stringMax := parts[1]

	// 尝试将 stringMin 和 stringMax 转换为无符号16位整数
	min64, err := strconv.ParseUint(stringMin, 10, 16)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid min: %s", err)
	}
	max64, err := strconv.ParseUint(stringMax, 10, 16)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid max: %s", err)
	}

	return uint16(min64), uint16(max64), nil
}

// PortRange returns the min port, max port, and a bool value indicating whether the port range is valid.
func (c *Config) PortRange() (uint16, uint16, bool) {
	minport, maxport, _ := c.parsePortRange()
	return minport, maxport, minport != 0 && maxport != 0
}
