package model

type Dockerfile struct {
	ExtensionID string          `toml:"extension_id"`
	Path        string          `toml:"path"`
	Build       bool            `toml:"build"`
	Run         bool            `toml:"run"`
	Args        DockerfileArg   `toml:"args"`
}

type DockerfileArg struct {
	BuildArg []BuildArg   `toml:"build"` //map[string]string --> won't work: https://github.com/BurntSushi/toml/issues/195
	RunArg   []RunArg     `toml:"run"`  //map[string]string
}

type BuildArg struct {
	Key   string `toml:"name"`
	Value string `toml:"value"`
}

type RunArg struct {
	Key   string `toml:"name"`
	Value string `toml:"value"`
}
