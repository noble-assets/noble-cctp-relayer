package relayer

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PromMetrics struct {
	WalletBalance   *prometheus.GaugeVec
	LatestHeight    *prometheus.GaugeVec
	BroadcastErrors *prometheus.CounterVec
}

func InitPromMetrics(port int16) *PromMetrics {
	reg := prometheus.NewRegistry()

	// labels
	var (
		walletLabels         = []string{"chain", "address", "denom"}
		heightLabels         = []string{"chain", "domain"}
		broadcastErrorLabels = []string{"chain", "domain"}
	)

	m := &PromMetrics{
		WalletBalance: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cctp_relayer_wallet_balance",
			Help: "The current balance for a wallet",
		}, walletLabels),
		LatestHeight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cctp_relayer_chain_latest_height",
			Help: "The current height of the chain",
		}, heightLabels),
		BroadcastErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cctp_relayer_broadcast_errors_total",
			Help: "The total number of failed broadcasts. Note: this is AFTER is retires `broadcast-retries` number of times (config setting).",
		}, broadcastErrorLabels),
	}

	reg.MustRegister(m.WalletBalance)
	reg.MustRegister(m.LatestHeight)
	reg.MustRegister(m.BroadcastErrors)

	// Expose /metrics HTTP endpoint
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		server := &http.Server{
			Addr:        fmt.Sprintf(":%d", port),
			ReadTimeout: 3 * time.Second,
		}
		log.Fatal(server.ListenAndServe())
	}()

	return m
}

func (m *PromMetrics) SetWalletBalance(chain, address, denom string, balance float64) {
	m.WalletBalance.WithLabelValues(chain, address, denom).Set(balance)
}

func (m *PromMetrics) SetLatestHeight(chain, domain string, height int64) {
	m.LatestHeight.WithLabelValues(chain, domain).Set(float64(height))
}

func (m *PromMetrics) IncBroadcastErrors(chain, domain string) {
	m.BroadcastErrors.WithLabelValues(chain, domain).Inc()
}
