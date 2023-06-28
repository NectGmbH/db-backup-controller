package main

import (
	"os"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	v1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
)

const (
	flagBaseURL   = "base-url"
	flagConfigDir = "config-dir"
	flagListen    = "listen"
	flagLogLevel  = "log-level"
)

var (
	cmdRoot = &cobra.Command{
		Short:             "Executes backups / restores for the Nect database backup-controller",
		PersistentPreRunE: cmdRootPersistentPreRunE,
	}

	baseURL       string
	configBackup  v1.DatabaseBackup
	configStorage v1.DatabaseBackupStorageClassSpec
	httpMux       *mux.Router
)

func init() {
	cmdRoot.PersistentFlags().String(flagBaseURL, getEnvDefault("BASE_URL", ""), "specifies the URL the given port is reachable at")
	cmdRoot.PersistentFlags().String(flagConfigDir, getEnvDefault("CONFIG_DIR", ""), "specifies where to find the configurations")
	cmdRoot.PersistentFlags().String(flagListen, getEnvDefault("LISTEN", ":3000"), "specifies the IP/port combination to open the HTTP server on")
	cmdRoot.PersistentFlags().String(flagLogLevel, getEnvDefault("LOG_LEVEL", "info"), "specifies the log-level to use")
}

func cmdRootPersistentPreRunE(cmd *cobra.Command, _ []string) (err error) {
	if strings.HasPrefix(cmd.Use, "help ") {
		// Let's not try to parse stuff and possibly explode when user
		// only asked for help
		return nil
	}

	lls, err := cmd.Flags().GetString(flagLogLevel)
	if err != nil {
		// How?
		return errors.Wrapf(err, "getting %s flag value", flagLogLevel)
	}
	ll, err := logrus.ParseLevel(lls)
	if err != nil {
		return errors.Wrap(err, "parsing log-level")
	}
	logrus.SetLevel(ll)

	configPath, err := cmd.Flags().GetString(flagConfigDir)
	if err != nil {
		// How?
		return errors.Wrapf(err, "getting %s flag value", flagConfigDir)
	}

	baseURL, err = cmd.Flags().GetString(flagBaseURL)
	if err != nil {
		return errors.Wrapf(err, "getting %s flag value", flagBaseURL)
	}

	if err = yamlToAny(path.Join(configPath, "backup.yaml"), &configBackup); err != nil {
		return errors.Wrap(err, "loading backup from configuration")
	}

	if err = yamlToAny(path.Join(configPath, "storageSpec.yaml"), &configStorage); err != nil {
		return errors.Wrap(err, "loading storage spec from configuration")
	}

	httpMux = mux.NewRouter()

	return nil
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func yamlToAny(filename string, data any) error {
	content, err := os.ReadFile(filename) //#nosec:G304 // Loading a given config-file, this is fine
	if err != nil {
		return errors.Wrap(err, "reading file contents")
	}

	return errors.Wrap(yaml.Unmarshal(content, data), "unmarshaling file")
}
