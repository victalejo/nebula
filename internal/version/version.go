package version

// Variables inyectadas via ldflags durante build por GoReleaser
// Ejemplo: -X github.com/victalejo/nebula/internal/version.Version={{.Version}}
var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

// Info retorna la información de versión completa
type Info struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	Commit    string `json:"commit"`
}

// GetInfo retorna la información de versión actual
func GetInfo() Info {
	return Info{
		Version:   Version,
		BuildTime: BuildTime,
		Commit:    Commit,
	}
}
