// Package data — workspace.go definit le format workspace generique.
package data

// Workspace est le manifeste d'un corpus chambre.
type Workspace struct {
	Name    string          `json:"name"`
	Version string          `json:"version,omitempty"`
	Created string          `json:"created,omitempty"`
	Files   WorkspaceFiles  `json:"files"`
	Config  WorkspaceConfig `json:"config,omitempty"`
}

// WorkspaceFiles pointe vers chaque fichier de donnees.
// Les chemins sont relatifs au dossier contenant workspace.json.
type WorkspaceFiles struct {
	Glossaire   string `json:"glossaire"`
	Kernel      string `json:"kernel,omitempty"`
	Index       string `json:"index"`
	SearchCache string `json:"search_cache"`
	Embeddings  string `json:"embeddings,omitempty"`
	Memory      string `json:"memory,omitempty"`
}

// WorkspaceConfig contient la configuration optionnelle.
type WorkspaceConfig struct {
	Modes   map[string]ModeConfig   `json:"modes,omitempty"`
	Piliers map[string]PilierConfig `json:"piliers,omitempty"`
}

// ModeConfig definit un mode de recherche.
type ModeConfig struct {
	G     float64 `json:"g"`
	NSpin float64 `json:"n_spin"`
}

// PilierConfig definit l'apparence d'un pilier.
type PilierConfig struct {
	Color string `json:"color,omitempty"`
	Label string `json:"label,omitempty"`
}
