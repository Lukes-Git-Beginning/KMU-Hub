package repository

import "errors"

// ErrInstallationNotFound is returned by writes that derive tenant_id from an
// owning plugin_installations row via subquery (kv_store.go Set,
// execution_log.go Create) when that subquery matches zero rows - the
// installation_id does not exist, was uninstalled, or belongs to a different
// tenant than the caller's RLS scope.
var ErrInstallationNotFound = errors.New("plugin installation not found")
