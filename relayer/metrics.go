package relayer

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PromMetrics struct {
	WalletBalance *prometheus.GaugeVec
}

func InitPromMetrics() *PromMetrics {
	reg := prometheus.NewRegistry()

	// labels
	var (
		walletLabels = []string{"chain", "address"}
	)

	m := &PromMetrics{
		WalletBalance: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cctp_relayer_wallet_balance",
			Help: "The current balance for a wallet",
		}, walletLabels),
	}

	reg.MustRegister(m.WalletBalance)

	// Expose /metrics HTTP endpoint
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		log.Fatal(http.ListenAndServe(":2112", nil))
	}()

	return m
}

func (m *PromMetrics) SetWalletBalance(chain, address string, balance float64) {
	m.WalletBalance.WithLabelValues(chain, address).Set(balance)
}
