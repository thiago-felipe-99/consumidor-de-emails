package main

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
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

func newMetrics() *metrics {
	return &metrics{
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

func serverMetrics(metrics *metrics) {
	registryMetrics := prometheus.NewRegistry()

	registryMetrics.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		metrics.emailsReceived,
		metrics.emailsReceivedBytes,
		metrics.emailsSent,
		metrics.emailsSentBytes,
		metrics.emailsSentAttachment,
		metrics.emailsSentAttachmentBytes,
		metrics.emailsSentWithAttachment,
		metrics.emailsResent,
		metrics.emailsSentTimeSeconds,
		metrics.emailsCacheAttachment,
		metrics.emailsCacheAttachmentBytes,
	)

	http.Handle("/metrics", promhttp.HandlerFor(registryMetrics, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	server := &http.Server{
		WriteTimeout: serverWriteTimeout,
		ReadTimeout:  serverReadTImeout,
		Addr:         ":8001",
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("[ERROR] - Error starting metrics server")
	}
}

func cacheMetrics(cache *cache, metrics *metrics) {
	ticker := time.NewTicker(time.Second)

	for range ticker.C {
		metrics.emailsCacheAttachment.Set(float64(cache.data.Len()))
		metrics.emailsCacheAttachmentBytes.Set(float64(cache.data.Capacity()))
	}
}
