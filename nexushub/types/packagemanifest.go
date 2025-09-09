package types

type PackageManifest struct {
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	Subscriptions []string `json:"subscriptions"`
}
