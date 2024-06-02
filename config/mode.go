package config

const (
	Dev  = "dev"
	Prod = "prod"
)

var mode = Dev

func SetMode(newMode string) {
	mode = newMode
}

func CurrentMode() string {
	return mode
}
