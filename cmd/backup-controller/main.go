package main

import (
	"net/http"
	"os"
	"time"

	"github.com/bombsimon/logrusr/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/NectGmbH/db-backup-controller/pkg/generated/clientset/versioned"
	"github.com/NectGmbH/db-backup-controller/pkg/generated/informers/externalversions"

	"github.com/Luzifer/rconfig/v2"
)

const (
	workerProcessCount = 2
)

var (
	cfg = struct {
		ImagePrefix     string        `flag:"image-prefix" default:"" description:"Base of the engine image to start"`
		JSONLog         bool          `flag:"json-log" default:"false" description:"enable json-logging"`
		Kubeconfig      string        `flag:"kubeconfig" default:"" description:"Path to a kubeconfig. Only required if out-of-cluster."`
		Listen          string        `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		LogLevel        string        `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		Master          string        `flag:"master" default:"" description:"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster."` //nolint:lll // no way to shorten
		RescanInterval  time.Duration `flag:"rescan-interval" default:"1h" description:"How often to re-scan existing resources without events"`
		TargetNamespace string        `flag:"target-namespace" default:"" description:"Where to create the backup resources"`
		VersionAndExit  bool          `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func initApp() error {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		return errors.Wrap(err, "parsing cli options")
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log-level")
	}
	logrus.SetLevel(l)

	if cfg.JSONLog {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	klog.SetLogger(logrusr.New(logrus.StandardLogger()))

	return nil
}

func main() {
	var err error
	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		logrus.WithField("version", version).Info("backup-controller")
		os.Exit(0)
	}

	kubeCfg, err := clientcmd.BuildConfigFromFlags(cfg.Master, cfg.Kubeconfig)
	if err != nil {
		logrus.WithError(err).Fatal("building kubeconfig")
	}

	kubeClient, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		logrus.WithError(err).Fatal("building kubernetes clientset")
	}

	crdClient, err := versioned.NewForConfig(kubeCfg)
	if err != nil {
		logrus.WithError(err).Fatal("building CRD clientset")
	}

	crdInformerFactory := externalversions.NewSharedInformerFactory(crdClient, cfg.RescanInterval)

	ctrl := newController(
		crdClient,
		kubeClient,
	)

	if err = ctrl.RegisterDatabaseBackupInformer(crdInformerFactory); err != nil {
		logrus.WithError(err).Fatal("registering databaseBackup informer")
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	// metricsHandler.AddHandler()
	go func() {
		server := &http.Server{
			Addr:              cfg.Listen,
			Handler:           http.DefaultServeMux,
			ReadHeaderTimeout: time.Second,
		}

		if err := server.ListenAndServe(); err != nil {
			logrus.WithError(err).Fatal("HTTP server reported error")
		}
	}()

	logrus.WithField("version", version).Info("backup-controller started")

	crdInformerFactory.Start(stopCh)
	ctrl.Run(workerProcessCount, stopCh)
}
