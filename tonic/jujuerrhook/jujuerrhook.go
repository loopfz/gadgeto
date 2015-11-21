package jujuerrhook

import "github.com/juju/errors"

func ErrHook(e error) (int, interface{}) {

	switch {
	case errors.IsBadRequest(e) || errors.IsNotValid(e) || errors.IsAlreadyExists(e) || errors.IsNotSupported(e) || errors.IsNotAssigned(e) || errors.IsNotProvisioned(e):
		return 400, e.Error()
	case errors.IsMethodNotAllowed(e):
		return 405, e.Error()
	case errors.IsNotFound(e) || errors.IsUserNotFound(e):
		return 404, e.Error()
	case errors.IsUnauthorized(e):
		return 401, e.Error()
	case errors.IsNotImplemented(e):
		return 501, e.Error()
	}
	return 500, e.Error()
}
