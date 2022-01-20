package model

type Metadata struct {
	Dockerfiles []Dockerfile `toml:"dockerfiles,omitempty" json:"dockerfiles,omitempty"`
}
