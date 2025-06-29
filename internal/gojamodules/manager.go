package turbo_modules

import (
	"fmt"

	"github.com/dop251/goja"
)

// GojaModuleManager manages external goja modules
type GojaModuleManager struct {
	modules map[string]GojaModule
}

// GojaModule represents a loadable goja module
type GojaModule interface {
	Register() error
}

// EventLoopRunner represents the interface for running functions on the event loop
type EventLoopRunner interface {
	RunOnLoop(fn func(*goja.Runtime)) bool
}

// NewGojaModuleManager creates a new module manager
func NewGojaModuleManager() *GojaModuleManager {
	return &GojaModuleManager{
		modules: make(map[string]GojaModule),
	}
}

// RegisterModule registers a module with the manager
func (gmm *GojaModuleManager) RegisterModule(name string, module GojaModule) {
	gmm.modules[name] = module
}

// LoadModules loads all registered modules
func (gmm *GojaModuleManager) LoadModules() error {
	for name, module := range gmm.modules {
		if err := module.Register(); err != nil {
			return fmt.Errorf("failed to load module %s: %w", name, err)
		}
	}
	return nil
}

// GetModule retrieves a module by name
func (gmm *GojaModuleManager) GetModule(name string) (GojaModule, bool) {
	module, exists := gmm.modules[name]
	return module, exists
}

// ListModules returns a list of registered module names
func (gmm *GojaModuleManager) ListModules() []string {
	names := make([]string, 0, len(gmm.modules))
	for name := range gmm.modules {
		names = append(names, name)
	}
	return names
}
