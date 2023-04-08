package main

import "github.com/prometheus/client_golang/prometheus"

type metricas struct {
	emailsRecebidos            prometheus.Counter
	emailsRecebidosBytes       prometheus.Counter
	emailsEnviados             prometheus.Counter
	emailsEnviadosBytes        prometheus.Counter
	emailsAnexosEnviados       prometheus.Counter
	emailsAnexosEnviadosBytes  prometheus.Counter
	emailsEnviadosComAnexo     prometheus.Counter
	emailsReenviados           prometheus.Counter
	emailsTempoDeEnvioSegundos prometheus.Histogram
	emailsCacheAnexos          prometheus.Gauge
	emailsCacheAnexosBytes     prometheus.Gauge
}

func criarMetricas() *metricas {
	return &metricas{
		emailsRecebidos: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_recebidos",
			Help: "A quantidade de emails recebidos pela fila do rabbit",
		}),
		emailsRecebidosBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_recebidos_bytes",
			Help: "A quantidade em bytes de emails recebidos pela fila do rebbit",
		}),
		emailsEnviados: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados",
			Help: "A quantidade de emails enviados com sucesso",
		}),
		emailsEnviadosBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados_bytes",
			Help: "A quantidade em bytes de emails enviados com sucesso",
		}),
		emailsAnexosEnviados: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_anexos_enviados",
			Help: "A quantidade de anexos enviados",
		}),
		emailsAnexosEnviadosBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_anexos_enviados_bytes",
			Help: "A quantidade em bytes de anexos enviados com sucesso",
		}),
		emailsEnviadosComAnexo: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados_com_anexo",
			Help: "A quantidade de emails enviados com sucesso e com anexo",
		}),
		emailsReenviados: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_reenviados",
			Help: "A quantidade de emails reeenviados para a fila do rabbit",
		}),
		emailsTempoDeEnvioSegundos: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "emails_tempo_de_envio_segundos",
			Help: "O tempo de envio de lotes de emails em segundos",
		}),
		emailsCacheAnexos: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "emails_cache_anexo",
			Help: "A quantidade de anexos no cache",
		}),
		emailsCacheAnexosBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "emails_cache_anexo_bytes",
			Help: "A quantidade em bytes de anexos no cache",
		}),
	}
}
