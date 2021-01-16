package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	manager *Manager
)

// -----------------------------------------------------------------------------

// Config is an interface describing functions needed by this module.
type Config interface {
	// DeepCopyConfig returns a copy of the current struct.
	DeepCopyConfig() Config
	// ConfigFile returns the path of the config file.
	ConfigFile() string
}

// Chan is a channel within which pointers to a new configuration will be sent.
type Chan chan Config

// Validator is a function type which will ensure that the content of the config
// file is valid and can be applied
type Validator func(currentConfig Config, newConfig Config) []error

// Applier is a function type which will apply the new configuration
type Applier func(currentConfig Config, newConfig Config) error

// -----------------------------------------------------------------------------

// GetManager returns the Manager that will give you access to all your configs.
func GetManager(logger Logger) *Manager {
	if manager == nil {
		manager = &Manager{
			logger: logger,
		}
	}

	return manager
}

// Manager is a struct that stores configuration watchers and the chans used to
// broadcasts new configurations when configurations files are updated.
type Manager struct {
	logger     Logger
	watchers   map[interface{}]*watcher
	chans      map[interface{}][]Chan
	validators map[interface{}][]Validator
	appliers   map[interface{}][]Applier
	mu         sync.RWMutex
}

// GetConfig returns an existing configuration, nil otherwise.
func (m *Manager) GetConfig(name interface{}) Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if w, ok := m.watchers[name]; ok {
		return w.config
	}

	return nil
}

// GetWatcher returns an existing configuration, nil otherwise.
func (m *Manager) GetWatcher(name interface{}) *watcher {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if w, ok := m.watchers[name]; ok {
		return w
	}

	return nil
}

// MakeConfig creates a named configuration. If config.ConfigFile() returns anything
// but an empty string it will spawn a goroutine which will watch for changes
// in the file. The file does not have to exists, it can be created after the
// config has been created.
func (m *Manager) MakeConfig(ctx context.Context, name interface{}, config Config) error {
	var err error

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watchers == nil {
		m.watchers = make(map[interface{}]*watcher)
	}

	if _, ok := m.watchers[name]; ok {
		return fmt.Errorf("The configuration `%v` already exists", name)
	}

	fsnotify, err := fsnotify.NewWatcher()

	if err != nil {
		return err
	}

	m.watchers[name] = &watcher{fsnotify, m, m.logger, name, config}

	// Load config from cli args and then from config file if exists
	err = m.watchers[name].loadConfig(config)

	if err != nil {
		return err
	}

	// Execute validators
	errs := m.runValidators(name, nil, config)

	if len(errs) > 0 {
		for _, err := range errs {
			m.logger.Errorf("Error while validating new conf: %v", err)
		}

		err = fmt.Errorf("New configuration not applied because error(s) have been found")
		return err
	}

	// Execute appliers
	err = m.runAppliers(name, nil, config)

	if err != nil {
		return err
	}

	// Spawn a goroutine to watch the config file it has been defined
	if len(config.ConfigFile()) > 0 {
		go m.watchers[name].watchConfigFile(ctx)

		// Sleep a bit to let the watchConfigFile go routine the time to watch
		// the configuration file it is supposed to.
		time.Sleep(time.Millisecond)
	}

	return err
}

// NewConfigChan returns a channel that will be used to send new configurations
// when the configuration file associated to the Config has been updated.
func (m *Manager) NewConfigChan(name interface{}) Chan {
	c := make(chan Config)
	m.registerChan(name, c)

	return c
}

func (m *Manager) registerChan(name interface{}, c Chan) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.chans == nil {
		m.chans = make(map[interface{}][]Chan)
	}

	m.chans[name] = append(m.chans[name], c)
}

// broadcastNewConfig sends a configuration pointer in all registered channels.
func (m *Manager) broadcastNewConfig(name interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.chans[name] {
		m.logger.Tracef("Signaling new conf %p in chan %p", m.watchers[name].config, m.chans[name][i])
		m.chans[name][i] <- m.watchers[name].config
	}
}

// AddValidators ...
func (m *Manager) AddValidators(name interface{}, validators ...Validator) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, validator := range validators {
		m.addValidator(name, validator)
	}
}

// addValidator ...
func (m *Manager) addValidator(name interface{}, validator Validator) {
	if m.validators == nil {
		m.validators = make(map[interface{}][]Validator)
	}

	m.validators[name] = append(m.validators[name], validator)
}

// AddAppliers ...
func (m *Manager) AddAppliers(name interface{}, appliers ...Applier) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, applier := range appliers {
		m.addApplier(name, applier)
	}
}

// addApplier ...
func (m *Manager) addApplier(name interface{}, applier Applier) {
	if m.appliers == nil {
		m.appliers = make(map[interface{}][]Applier)
	}

	m.appliers[name] = append(m.appliers[name], applier)
}

func (m *Manager) runValidators(name interface{}, currentConfig Config, newConfig Config) []error {
	var errs []error
	for _, validator := range m.validators[name] {
		verrs := validator(currentConfig, newConfig)

		if verrs != nil {
			errs = append(errs, verrs...)
		}
	}

	return errs
}

func (m *Manager) runAppliers(name interface{}, currentConfig Config, newConfig Config) error {
	for _, applier := range m.appliers[name] {
		err := applier(currentConfig, newConfig)

		if err != nil {
			return err
		}
	}

	return nil
}
