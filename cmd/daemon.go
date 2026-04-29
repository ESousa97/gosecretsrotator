package cmd

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/rotation"
	"github.com/esousa97/gosecretsrotator/internal/storage"
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
		defer hdb.Close()

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
				if err := rotation.RotateSecret(store, hdb, k); err != nil {
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
	daemonCmd.Flags().DurationVar(&daemonCheckInterval, "check-interval", time.Hour, "How often the daemon checks for expired secrets")
	rootCmd.AddCommand(daemonCmd)
}
