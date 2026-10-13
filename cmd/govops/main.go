package main

import (
	"log/slog"
	"os"

	"github.com/khwajalabs/govops/internal/catalog"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// A real baseline would be loaded from a signed, versioned file; this proves
	// the control plane can name the posture it holds the platform to.
	b := catalog.GovProdModerate()
	log.Info("loaded compliance baseline", "name", b.Name, "controls", b.IDs())
}
