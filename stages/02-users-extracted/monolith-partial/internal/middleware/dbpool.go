package middleware

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// poolCollector expõe as estatísticas do pool de conexões do pgx como métricas
// Prometheus. Serve para evidenciar contenção na camada de banco de dados: sob
// concorrência, AcquireDuration e EmptyAcquireCount crescem quando as goroutines
// passam a esperar por uma conexão livre.
type poolCollector struct {
	pool    *pgxpool.Pool
	service string

	acquiredConns   *prometheus.Desc
	idleConns       *prometheus.Desc
	totalConns      *prometheus.Desc
	maxConns        *prometheus.Desc
	acquireCount    *prometheus.Desc
	emptyAcquire    *prometheus.Desc
	canceledAcquire *prometheus.Desc
	acquireDuration *prometheus.Desc
}

func newPoolCollector(service string, pool *pgxpool.Pool) *poolCollector {
	labels := prometheus.Labels{"service": service}
	d := func(name, help string) *prometheus.Desc {
		return prometheus.NewDesc("db_pool_"+name, help, nil, labels)
	}
	return &poolCollector{
		pool:            pool,
		service:         service,
		acquiredConns:   d("acquired_conns", "Conexões atualmente em uso pelo pool"),
		idleConns:       d("idle_conns", "Conexões ociosas no pool"),
		totalConns:      d("total_conns", "Total de conexões abertas pelo pool"),
		maxConns:        d("max_conns", "Limite máximo de conexões do pool"),
		acquireCount:    d("acquire_total", "Total de aquisições de conexão bem-sucedidas"),
		emptyAcquire:    d("empty_acquire_total", "Aquisições que precisaram esperar por uma conexão livre"),
		canceledAcquire: d("canceled_acquire_total", "Aquisições canceladas antes de obter conexão"),
		acquireDuration: d("acquire_duration_seconds_total", "Tempo acumulado de espera por conexão"),
	}
}

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquiredConns
	ch <- c.idleConns
	ch <- c.totalConns
	ch <- c.maxConns
	ch <- c.acquireCount
	ch <- c.emptyAcquire
	ch <- c.canceledAcquire
	ch <- c.acquireDuration
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	g := func(desc *prometheus.Desc, v float64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v)
	}
	counter := func(desc *prometheus.Desc, v float64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v)
	}

	g(c.acquiredConns, float64(s.AcquiredConns()))
	g(c.idleConns, float64(s.IdleConns()))
	g(c.totalConns, float64(s.TotalConns()))
	g(c.maxConns, float64(s.MaxConns()))
	counter(c.acquireCount, float64(s.AcquireCount()))
	counter(c.emptyAcquire, float64(s.EmptyAcquireCount()))
	counter(c.canceledAcquire, float64(s.CanceledAcquireCount()))
	counter(c.acquireDuration, s.AcquireDuration().Seconds())
}

// RegisterPoolCollector registra as métricas do pool de conexões no coletor
// padrão do Prometheus, sob o rótulo do serviço informado.
func RegisterPoolCollector(service string, pool *pgxpool.Pool) {
	prometheus.MustRegister(newPoolCollector(service, pool))
}
