package controller

import (
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, *http.Client) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, c *http.Client) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, c); err != nil {
			return err
		}
	}
	return nil
}
