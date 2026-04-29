package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/rotation"
	"github.com/esousa97/gosecretsrotator/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var daemonCheckInterval time.Duration

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the rotation engine in the foreground; rotates expired secrets on each tick",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		hdb, err := storage.NewHistoryDB("history.db")
		if err != nil {
			return fmt.Errorf("init history db: %w", err)
		}
		defer func() {
			if err := hdb.Close(); err != nil {
				log.Printf("failed to close history db: %v", err)
			}
		}()

		// Start Metrics Server
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.Handler())
			srv := &http.Server{
				Addr:              fmt.Sprintf(":%d", cfg.MetricsPort),
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
			}
			log.Printf("metrics server starting on :%d", cfg.MetricsPort)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("metrics server error: %v", err)
			}
		}()

		log.Printf("daemon starting; check interval=%s", daemonCheckInterval)
		tick := time.NewTicker(daemonCheckInterval)
		defer tick.Stop()

		runOnce := func() {
			store := storage.NewStore("secrets.json", cfg.MasterPassword)
			if err := store.Load(); err != nil {
				log.Printf("load vault: %v", err)
				return
			}

			due := rotation.DueSecrets(store, time.Now().UTC())
			if len(due) == 0 {
				return
			}
			for _, k := range due {
				log.Printf("rotating %q (interval=%dd)", k, store.Secrets[k].IntervalDays)
				if err := rotation.RotateSecret(store, hdb, k, cfg.WebhookURL); err != nil {
					log.Printf("rotate %q failed: %v", k, err)
					continue
				}
				log.Printf("rotated %q OK", k)
			}
		}

		runOnce()
		for {
			select {
			case <-ctx.Done():
				log.Printf("daemon shutting down")
				return nil
			case <-tick.C:
				runOnce()
			}
		}
	},
}

func init() {
	daemonCmd.Flags().DurationVar(
		&daemonCheckInterval,
		"check-interval",
		time.Hour,
		"How often the daemon checks for expired secrets",
	)
	rootCmd.AddCommand(daemonCmd)
}
