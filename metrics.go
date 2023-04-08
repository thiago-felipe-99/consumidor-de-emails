package main

import "github.com/prometheus/client_golang/prometheus"

type metricas struct {
	emailsReceived             prometheus.Counter
	emailsReceivedBytes        prometheus.Counter
	emailsSent                 prometheus.Counter
	emailsSentBytes            prometheus.Counter
	emailsSentAttachment       prometheus.Counter
	emailsSentAttachmentBytes  prometheus.Counter
	emailsSentWithAttachment   prometheus.Counter
	emailsResent               prometheus.Counter
	emailsSentTimeSeconds      prometheus.Histogram
	emailsCacheAttachment      prometheus.Gauge
	emailsCacheAttachmentBytes prometheus.Gauge
}

func newMetrics() *metricas {
	return &metricas{
		emailsReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_recebidos",
			Help: "A quantidade de emails recebidos pela fila do rabbit",
		}),
		emailsReceivedBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_recebidos_bytes",
			Help: "A quantidade em bytes de emails recebidos pela fila do rebbit",
		}),
		emailsSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados",
			Help: "A quantidade de emails enviados com sucesso",
		}),
		emailsSentBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados_bytes",
			Help: "A quantidade em bytes de emails enviados com sucesso",
		}),
		emailsSentAttachment: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_anexos_enviados",
			Help: "A quantidade de anexos enviados",
		}),
		emailsSentAttachmentBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_anexos_enviados_bytes",
			Help: "A quantidade em bytes de anexos enviados com sucesso",
		}),
		emailsSentWithAttachment: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados_com_anexo",
			Help: "A quantidade de emails enviados com sucesso e com anexo",
		}),
		emailsResent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_reenviados",
			Help: "A quantidade de emails reeenviados para a fila do rabbit",
		}),
		emailsSentTimeSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "emails_tempo_de_envio_segundos",
			Help: "O tempo de envio de lotes de emails em segundos",
		}),
		emailsCacheAttachment: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "emails_cache_anexo",
			Help: "A quantidade de anexos no cache",
		}),
		emailsCacheAttachmentBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "emails_cache_anexo_bytes",
			Help: "A quantidade em bytes de anexos no cache",
		}),
	}
}
