package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"

	"github.com/kr/pretty"
	"github.com/sirupsen/logrus"
	qdconfig "sylr.dev/libqd/config"
	"sylr.dev/libqd/config/example/config"
)

var (
	version = "1.2.3"
)

type Logger struct {
	l *logrus.Logger
}

func (l *Logger) Tracef(format string, vals ...interface{}) {
	l.l.Tracef("libqd/config: "+format, vals...)
}

func (l *Logger) Debugf(format string, vals ...interface{}) {
	l.l.Debugf("libqd/config: "+format, vals...)
}

func (l *Logger) Infof(format string, vals ...interface{}) {
	l.l.Infof("libqd/config: "+format, vals...)
}

func (l *Logger) Warnf(format string, vals ...interface{}) {
	l.l.Warnf("libqd/config: "+format, vals...)
}

func (l *Logger) Errorf(format string, vals ...interface{}) {
	l.l.Errorf("libqd/config: "+format, vals...)
}

func (l *Logger) Fatalf(format string, vals ...interface{}) {
	l.l.Fatalf("libqd/config: "+format, vals...)
}

func main() {
	logrus.SetLevel(logrus.TraceLevel)
	logger := logrus.StandardLogger()
	qdlogger := &Logger{l: logger}

	conf := &config.MyAppConfiguration{
		HTTPPort: 8080,
		File:     "./config.yaml",
	}

	ctx := context.Background()

	// mutex to prevent data race around conf
	mu := sync.RWMutex{}
	cm := configMutex{&mu, qdlogger}

	// Manager
	configManager := qdconfig.GetManager(qdlogger)

	// Add a validator function
	configManager.AddValidators(nil, cm.configValidator)

	// Add an applier function
	configManager.AddAppliers(nil, cm.configApplier)

	// Make the config
	err := configManager.MakeConfig(ctx, nil, conf)

	if err != nil {
		logger.Fatal(err)
	}

	// Print version and exit
	if conf.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	// goroutine that listen for new config
	go func() {
		c := configManager.NewConfigChan(nil)

		for {
			tconf := (<-c).(*config.MyAppConfiguration)
			mu.Lock()
			conf = tconf
			mu.Unlock()
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		html := template.New(fmt.Sprintf("%# v", pretty.Formatter(conf)))

		w.Header().Set("Content-Type", "text/plain")
		html.Execute(w, nil)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", conf.HTTPPort)

	http.ListenAndServe(addr, nil)
}

type configMutex struct {
	*sync.RWMutex
	l *Logger
}

func (cm *configMutex) configValidator(currentConfig qdconfig.Config, newConfig qdconfig.Config) []error {
	var currentConf *config.MyAppConfiguration
	var newConf *config.MyAppConfiguration
	var ok bool

	// currentConfig is nil the first time the validator is called
	if currentConfig != nil {
		currentConf, ok = currentConfig.(*config.MyAppConfiguration)

		if !ok {
			return []error{fmt.Errorf("Can not cast currentConfig to (*config.MyAppConfiguration)")}
		}
	}

	newConf, ok = newConfig.(*config.MyAppConfiguration)

	if !ok {
		return []error{fmt.Errorf("Can not cast newConfig to (*config.MyAppConfiguration)")}
	}

	// ---------------------------------------------------------------------
	// Here begins the actual validation of the values of newConfig
	// ---------------------------------------------------------------------
	var errs []error

	if currentConfig == nil {
		if newConf.HTTPPort < 0 || newConf.HTTPPort > 65535 {
			errs = append(errs, fmt.Errorf("HTTPPort `%d` is not valid", newConf.HTTPPort))
		}
	} else {
		if newConf.HTTPPort != currentConf.HTTPPort {
			errs = append(errs, fmt.Errorf("HTTPPort `%d` can not be changed to `%d`", currentConf.HTTPPort, newConf.HTTPPort))
		}
	}

	return errs
}

func (cm *configMutex) configApplier(currentConfig qdconfig.Config, newConfig qdconfig.Config) error {
	var currentConf *config.MyAppConfiguration
	var newConf *config.MyAppConfiguration
	var ok bool

	// currentConfig is nil the first time the validator is called
	if currentConfig != nil {
		currentConf, ok = currentConfig.(*config.MyAppConfiguration)

		if !ok {
			return fmt.Errorf("Can not cast currentConfig to (*config.MyAppConfiguration)")
		}
	}

	newConf, ok = newConfig.(*config.MyAppConfiguration)

	if !ok {
		return fmt.Errorf("Can not cast newConfig to (*config.MyAppConfiguration)")
	}

	switch len(newConf.Verbose) {
	case 1:
		logrus.SetLevel(logrus.FatalLevel)
	case 2:
		logrus.SetLevel(logrus.ErrorLevel)
	case 3:
		logrus.SetLevel(logrus.WarnLevel)
	case 4:
		logrus.SetLevel(logrus.InfoLevel)
	case 5:
		logrus.SetLevel(logrus.DebugLevel)
	case 6:
		fallthrough
	default:
		logrus.SetLevel(logrus.TraceLevel)
	}

	if currentConf != nil {
		cm.Lock()
		newConf.Reloads = newConf.Reloads + 1
		cm.l.Debugf("Incrementing conf.Reloads to `%d`", newConf.Reloads)
		cm.Unlock()
	}

	return nil
}
