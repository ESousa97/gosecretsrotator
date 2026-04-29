package rotation

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RotationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gosecrets_rotations_total",
		Help: "The total number of secret rotations",
	}, []string{"status"})

	SecretsExpiringSoon = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gosecrets_secrets_expiring_soon",
		Help: "Number of secrets expiring in the next 3 days",
	})

	LastRotationSuccess = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gosecrets_last_rotation_success",
		Help: "Timestamp of the last successful rotation per secret",
	}, []string{"secret_name"})
)
