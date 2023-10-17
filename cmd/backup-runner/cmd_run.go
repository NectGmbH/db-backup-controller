package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	httpHelper "github.com/Luzifer/go_helpers/v2/http"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/NectGmbH/db-backup-controller/pkg/backupengine"
	"github.com/NectGmbH/db-backup-controller/pkg/backupengine/opts"
)

type (
	ipcPayload struct {
		Action string   `json:"action"`
		Args   []string `json:"args"`
	}
)

var (
	actionRunning atomic.Bool
	engine        backupengine.Implementation
	cmdRun        = &cobra.Command{
		Use:   "run",
		Short: "Starts the periodic backup routine together with the IPC server for `backup` and `restore` commands",
		RunE:  cmdRunRunE,
	}
)

func init() {
	cmdRoot.AddCommand(cmdRun)
}

func cmdRunRunE(cmd *cobra.Command, _ []string) (err error) {
	// First lets tell our OPS a little about the backup
	logrus.WithFields(logrus.Fields{
		"encryption": isEncrypted(),
		"name":       configBackup.Name,
		"namespace":  configBackup.Namespace,
	}).Info("backup-runner started run-loop")

	// Initialize the engine once
	engine = backupengine.GetByName(configBackup.Spec.DatabaseType)
	if err = engine.Init(opts.InitOpts{
		BaseURL: baseURL,
		Mux:     httpMux,
		Spec:    configBackup.Spec,
	}); err != nil {
		return errors.Wrap(err, "initializing backup engine")
	}

	// Add the IPC route
	httpMux.HandleFunc("/ipc", handleIPCRequest).
		Methods(http.MethodPost).
		MatcherFunc(func(r *http.Request, _ *mux.RouteMatch) bool {
			// IPC route must be called through loopback interface
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			return host == "127.0.0.1" || host == "[::1]"
		})

	// Start and run the HTTP server
	listenAddr, err := cmd.Flags().GetString(flagListen)
	if err != nil {
		// How?
		return errors.Wrapf(err, "getting %s flag value", flagListen)
	}

	var (
		httpServer = &http.Server{
			Addr:              listenAddr,
			Handler:           httpHelper.NewHTTPLogHandlerWithLogger(httpMux, logrus.StandardLogger()),
			ReadHeaderTimeout: time.Second,
		}
		httpServerErr = make(chan error, 1)
	)

	go func() { httpServerErr <- httpServer.ListenAndServe() }()

	// Add a ticker routine to trigger backups
	var (
		triggerAutoBackup = make(chan struct{}, 1)
		triggerErr        = make(chan error, 1)
	)
	go func() { triggerErr <- tickAutoBackup(triggerAutoBackup) }()

	// Wait for something bad to happen...
	for {
		select {
		case <-triggerAutoBackup:
			if err := triggerRunAction("backup", nil); err != nil {
				logrus.WithError(err).Error("triggering automatic background backup")
			}

		case err := <-httpServerErr:
			return errors.Wrap(err, "running HTTP server")

		case err := <-triggerErr:
			return errors.Wrap(err, "running auto-backup-ticker")
		}
	}
}

func handleIPCRequest(w http.ResponseWriter, r *http.Request) {
	var payload ipcPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, errors.Wrap(err, "decoding request").Error(), http.StatusBadRequest)
		return
	}

	go func() {
		if err := triggerRunAction(payload.Action, payload.Args); err != nil {
			logrus.WithError(err).Error("triggering action from IPC request")
		}
	}()
	w.WriteHeader(http.StatusCreated)
}

func isEncrypted() string {
	var (
		hasPass   bool
		hasNoPass bool
	)

	for _, loc := range configStorage.BackupLocations {
		if loc.EncryptionPass.Value == "" {
			hasNoPass = true
		} else {
			hasPass = true
		}
	}

	switch {
	case hasPass && !hasNoPass:
		return "fully-encrypted"

	case hasPass && hasNoPass:
		return "partially-encrypted"

	default:
		return "not-encrypted"
	}
}

func triggerIPCRequest(cmd *cobra.Command, payload ipcPayload) error {
	listenAddr, err := cmd.Flags().GetString(flagListen)
	if err != nil {
		// How?
		return errors.Wrapf(err, "getting %s flag value", flagListen)
	}

	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return errors.Wrap(err, "getting port from listen address")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshalling IPC payload")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://127.0.0.1:%s/ipc", port), bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, "creating IPC request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "executing IPC request")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("closing IPC response body (leaked fd)")
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		return errors.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	logrus.Info("IPC request started successfully, see runner logs for details")

	return nil
}

func triggerRunAction(action string, args []string) error {
	if !actionRunning.CompareAndSwap(false, true) {
		// We did not switch from not-running to running: We must not run!
		return errors.New("concurrent action running")
	}
	defer actionRunning.Store(false)

	switch action {
	case "backup":
		if err := executeBackup(); err != nil {
			logrus.WithError(err).Error("executing backup action")
		}

	case "restore":
		if len(args) != 2 { //nolint:gomnd
			return errors.Errorf("invalid number of arguments")
		}

		if err := executeRestore(args[0], args[1]); err != nil {
			logrus.WithError(err).Error("executing restore action")
		}

	default:
		logrus.WithError(errors.Errorf("unknown action %s", action)).Error("invalid action called")
	}

	return nil
}
