package config

import (
	"crypto/rand"
	"github.com/ezshare/server/config/ip"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"path/filepath"
	"regexp"
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
	TurnRealm      string      `default:"ezshare" split_words:"true"`
	TurnIPProvider ip.Provider `ignored:"true"`

	AuthMode                 string            `default:"turn" split_words:"true"`
	CorsAllowedOrigins       []string          `split_words:"true"`
	CheckOrigin              func(string) bool `ignored:"true" json:"-"`
	UsersFile                string            `split_words:"true"`
	CloseRoomWhenOwnerLeaves bool              `default:"true" split_words:"true"`
	Version                  string            `default:"1.0"`
}

// LoadConfig 加载配置文件到环境变量，并解析环境变量，生成 Config
func LoadConfig() *Config {
	// 1. 读取配置文件
	log.Debug().Msg("Begin to load config")
	dir, err := workOrExecAbsDir()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get work or exec dir")
		return nil
	}
	for _, file := range configFilePath(dir) {
		_, existErr := os.Stat(file)
		if existErr == nil {
			if err := godotenv.Load(file); err != nil {
				log.Error().Err(err).Str("file", file).Msg("Failed to load config file")
			}
			log.Debug().Str("file", file).Msg("Config file loaded")
			break
		} else {
			log.Debug().Str("file", file).Msg("Config file not exist")
			continue
		}
	}

	// 2. 解析环境变量，生成Config
	config := &Config{}
	if err := envconfig.Process(prefix, config); err != nil {
		log.Error().Err(err).Msg("Failed to process env config")
		return config
	}
	log.Debug().Msg("Env config processed")

	// 3. 对Config进行补充处理
	// 3.1 密码为空，生成临时随机密码，服务重启则密码失效
	if len(config.Secret) == 0 {
		config.Secret = make([]byte, 32)
		if _, err := rand.Read(config.Secret); err != nil {
			log.Error().Err(err).Msg("Failed to generate random secret")
		}
		log.Debug().Msg("Random secret generated")
	}

	// 3.2 配置CORS允许的Origin和检查Origin的函数
	var compiledAllowedOrigins []*regexp.Regexp
	for _, origin := range config.CorsAllowedOrigins {
		compiled, err := regexp.Compile(origin)
		if err != nil {
			log.Error().Err(err).Str("origin", origin).Msg("Failed to compile origin")
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

	// 3.3 TODO: IPProvider
	config.TurnIPProvider = &ip.Static{
		V4: net.ParseIP(config.ExternalIP[0]),
		V6: nil,
	}
	log.Debug().Msg("IP provider generated")
	log.Info().Msg("Config loaded")

	return config
}

// workOrExecAbsDir 获取工作目录或可执行文件的目录，根据当前运行模式判断
// 如果当前是 Dev 模式，返回工作目录。如果是 Prod 模式，返回可执行文件的目录
func workOrExecAbsDir() (string, error) {
	if CurrentMode() == Dev {
		log.Debug().Msg("Use work dir")
		return filepath.Abs(".")
	}
	log.Debug().Msg("Use executable dir")
	return execDir()
}

// execDir 获取可执行文件的目录，如果当前使用 go run 运行，返回的是 go run 的临时目录
// 如果使用 go build 编译后运行，返回的是可执行文件所在的目录
func execDir() (string, error) {
	path, err := os.Executable()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get executable dir path")
		return "", err
	}
	return filepath.Dir(path), nil
}

// configFilePath 生成配置文件的绝对路径，配置文件名固定为 files 切片中的文件名
func configFilePath(dir string) []string {
	var configFilePaths []string
	for _, file := range files {
		configFilePaths = append(configFilePaths, filepath.Join(dir, file))
	}
	return configFilePaths
}
