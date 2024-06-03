package config

import (
	"crypto/rand"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
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

	TurnAddress string `default:":3478" required:"true" split_words:"true"`
	TurnRealm   string `default:"ezshare" split_words:"true"`

	CorsAllowedOrigins       []string          `split_words:"true"`
	CheckOrigin              func(string) bool `ignored:"true" json:"-"`
	UsersFile                string            `split_words:"true"`
	CloseRoomWhenOwnerLeaves bool              `default:"true" split_words:"true"`
}

// LoadConfig 加载配置文件，解析环境变量，生成 Config 配置
func LoadConfig() *Config {
	// 加载配置文件到环境变量
	log.Info().Msg("Begin to load config")
	dir, err := workOrExecAbsDir()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get work or exec dir")
		return nil
	}
	for _, file := range configFilePath(dir) {
		_, existErr := os.Stat(file)
		if existErr == nil {
			// If file exist, try to load it
			if err := godotenv.Load(file); err != nil {
				log.Error().Err(err).Str("file", file).Msg("Failed to load config file")
			}
			log.Info().Str("file", file).Msg("Config file loaded")
		} else {
			// If file not exist, continue to check next file
			log.Info().Str("file", file).Msg("Config file not exist")
			continue
		}
	}
	log.Info().Msg("All config files loaded")

	// 解析环境变量，生成 Config 配置
	config := &Config{}
	if err := envconfig.Process(prefix, config); err != nil {
		log.Error().Err(err).Msg("Failed to process env config")
		return config
	}
	log.Info().Msg("Env config processed")

	// 如果 Secret 未配置，随机生成一个临时的 Secret，但是只能用于当前进程
	// 如果进程重启，Secret 会变化
	if len(config.Secret) == 0 {
		config.Secret = make([]byte, 32)
		if _, err := rand.Read(config.Secret); err != nil {
			log.Error().Err(err).Msg("Failed to generate random secret")
		} else {
			log.Info().Msg("Random secret generated")
		}
	}

	// 配置 CORS 允许的 Origin 和检查 Origin 的函数
	var compiledAllowedOrigins []*regexp.Regexp
	for _, origin := range config.CorsAllowedOrigins {
		compiled, err := regexp.Compile(origin)
		if err != nil {
			log.Error().Err(err).Str("origin", origin).Msg("Failed to compile origin")
		}
		compiledAllowedOrigins = append(compiledAllowedOrigins, compiled)
	}

	config.CheckOrigin = func(origin string) bool {
		// 非浏览器请求，直接通过
		if origin == "" {
			return true
		}

		// 对浏览器请求的 Origin 进行检查
		for _, compiled := range compiledAllowedOrigins {
			if compiled.Match([]byte(strings.ToLower(origin))) {
				return true
			}
		}
		return false
	}

	return config
}

// workOrExecAbsDir 获取工作目录或可执行文件的目录，根据当前运行模式判断
// 如果当前是 Dev 模式，返回工作目录。如果是 Prod 模式，返回可执行文件的目录
func workOrExecAbsDir() (string, error) {
	if CurrentMode() == Dev {
		log.Info().Msg("Use work dir")
		return filepath.Abs(".")
	}
	log.Info().Msg("Use executable dir")
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
