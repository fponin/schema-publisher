package state

// State holds persistent user history across runs.
type State struct {
	LastAuthor           string   `json:"lastAuthor"`
	RecentCommitMessages []string `json:"recentCommitMessages"`
	RecentServices       []string `json:"recentServices"`
	RecentOutputFiles    []string `json:"recentOutputFiles"`
	LastUsed             LastUsed `json:"lastUsed"`
}

// LastUsed captures the most recently used parameters.
type LastUsed struct {
	Env        string `json:"env"`
	Service    string `json:"service"`
	Namespace  string `json:"namespace"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
	OutputFile string `json:"outputFile"`
}

const (
	maxCommitMessages = 10
	maxServices       = 20
	maxOutputFiles    = 5
)
