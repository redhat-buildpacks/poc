package model

type Dockerfile struct {
	ExtensionID string          `toml:"extension_id"`
	Path        string          `toml:"path"`
	Build       bool            `toml:"build"`
	Run         bool            `toml:"run"`
	Args        []DockerfileArg `toml:"args"`
}

type DockerfileArg struct {
	Key   string `toml:"name"`
	Value string `toml:"value"`
}