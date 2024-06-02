package config

import (
	"github.com/rs/zerolog/log"
	"testing"
)

func TestExecutablePath(t *testing.T) {
	path, err := execDir()
	if err != nil {
		return
	}
	log.Info().Str("path", path).Msg("Executable path")
}

func TestWorkAbsDir(t *testing.T) {
	path, _ := workOrExecAbsDir()
	log.Info().Str("path", path).Msg("Work dir")

	SetMode(Prod)
	path, _ = workOrExecAbsDir()
	log.Info().Str("path", path).Msg("Executable dir")
}

func TestConfigFilePath(t *testing.T) {
	path, _ := workOrExecAbsDir()
	configFilePaths := configFilePath(path)
	log.Info().Strs("files", configFilePaths).Msg("Config files")
}
