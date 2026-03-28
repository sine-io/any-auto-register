package main

import (
	"fmt"
	"os"

	viperconfig "go-control-plane/internal/adapters/config/viper"
	gingateway "go-control-plane/internal/adapters/http/gin"
	zerologadapter "go-control-plane/internal/adapters/log/zerolog"

	"github.com/spf13/cobra"
)

func newServerCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the Go control plane HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "path to config file")
	return cmd
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use: "go-control-plane",
	}
	root.AddCommand(newServerCommand())
	return root
}

func runServer(configPath string) error {
	cfg, err := viperconfig.Load(configPath)
	if err != nil {
		return err
	}

	logger := zerologadapter.New(cfg.Log.Level)
	router := gingateway.NewRouter(cfg, logger)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	logger.Info().Str("addr", addr).Msg("starting go control plane")
	return router.Run(addr)
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
