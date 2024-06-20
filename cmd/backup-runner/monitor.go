package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricsLabelBackupName      = "name"
	metricsLabelBackupNamespace = "namespace"
	metricsLabelEngine          = "engine"
	metricsLabelJobType         = "job_type"

	metricsLabelValueJobTypeBackup  = "backup"
	metricsLabelValueJobTypeRestore = "restore"

	metricsNameLastJobSuccess       = "last_job_success"
	metricsNameLastSuccessfulBackup = "last_successful_backup"
	metricsNameNextScheduledBackup  = "next_scheduled_backup"
	metricsNameRunnerStartedAt      = "runner_started_at"
	metricsNameStoredBackupCount    = "stored_backup_count"

	metricsNamespace = "db_backup_controller"
)

type (
	appMonitor struct {
		mGLastSuccessfulBackup prometheus.Gauge
		mGNextScheduledBackup  prometheus.Gauge
		mGRunnerStartedAt      prometheus.Gauge
		mGStoredBackupCount    prometheus.Gauge
		mGVKLastJobStatus      *prometheus.GaugeVec
	}
)

func newAppMonitor() *appMonitor {
	am := appMonitor{}

	am.mGVKLastJobStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   metricsNamespace,
		Name:        metricsNameLastJobSuccess,
		Help:        "status of the last execution of specified job type (0 = failed, 1 = success)",
		ConstLabels: am.InstanceConstLabels(),
	}, []string{metricsLabelJobType})

	am.mGLastSuccessfulBackup = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   metricsNamespace,
		Name:        metricsNameLastSuccessfulBackup,
		Help:        "timestamp of last successful job of type backup (unix-timestamp)",
		ConstLabels: am.InstanceConstLabels(),
	})

	am.mGNextScheduledBackup = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   metricsNamespace,
		Name:        metricsNameNextScheduledBackup,
		Help:        "timestamp of next scheduled automatic backup (unix-timestamp)",
		ConstLabels: am.InstanceConstLabels(),
	})

	am.mGRunnerStartedAt = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   metricsNamespace,
		Name:        metricsNameRunnerStartedAt,
		Help:        "timestamp when the runner run-command was started (unix-timestamp)",
		ConstLabels: am.InstanceConstLabels(),
	})

	am.mGStoredBackupCount = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   metricsNamespace,
		Name:        metricsNameStoredBackupCount,
		Help:        "amount of backups held in the storage backend for this instance (0..N)",
		ConstLabels: am.InstanceConstLabels(),
	})

	// We're cheating a little here and set this on creation of the
	// appMonitor and not to the value of the process start but should
	// not really matter as there are only a few milliseconds between.
	am.mGRunnerStartedAt.SetToCurrentTime()

	return &am
}

func (appMonitor) InstanceConstLabels() prometheus.Labels {
	return prometheus.Labels{
		metricsLabelBackupName:      configBackup.Name,
		metricsLabelBackupNamespace: configBackup.Namespace,
		metricsLabelEngine:          configBackup.Spec.DatabaseType,
	}
}

//revive:disable-next-line:flag-parameter // That's not a flag but a value
func (a *appMonitor) RegisterJobStatus(jobType string, successful bool) {
	if a == nil {
		// Monitoring is not initialized, drop silently
		return
	}

	var sv float64
	if successful {
		sv = 1
	}

	a.mGVKLastJobStatus.WithLabelValues(jobType).Set(sv)
	if successful && jobType == metricsLabelValueJobTypeBackup {
		a.mGLastSuccessfulBackup.SetToCurrentTime()
	}
}

func (a *appMonitor) UpdateNextScheduled(t time.Time) {
	if a == nil {
		// Monitoring is not initialized, drop silently
		return
	}

	// Copied from the implementation of SetToCurrentTime
	a.mGNextScheduledBackup.Set(float64(t.UnixNano()) / 1e9) //nolint:mnd
}

func (a *appMonitor) UpdateStoredBackupCount(n int) {
	if a == nil {
		// Monitoring is not initialized, drop silently
		return
	}

	a.mGStoredBackupCount.Set(float64(n))
}
