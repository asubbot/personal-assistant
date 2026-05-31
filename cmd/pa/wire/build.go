package wire

import (
	"log/slog"
	"pa/internal/config"
)

// Build constructs the application composition root from configuration (EP-042).
func Build(cfg *config.Config, configPath string, logger *slog.Logger) (*Application, error) {
	infra, err := BuildInfrastructure(cfg, configPath, logger)
	if err != nil {
		return nil, err
	}
	return &Application{
		Cfg:        cfg,
		ConfigPath: configPath,
		Logger:     logger,
		Infra:      infra,
	}, nil
}
