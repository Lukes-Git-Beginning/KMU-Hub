package plugin

import "errors"

var (
	ErrManifestNotFound       = errors.New("plugin manifest not found")
	ErrManifestSlugExists     = errors.New("plugin manifest with this slug already exists")
	ErrInstallationNotFound   = errors.New("plugin installation not found")
	ErrAlreadyInstalled       = errors.New("plugin is already installed for this tenant")
	ErrInstallationNotActive  = errors.New("plugin installation is not active")
	ErrPermissionNotDeclared  = errors.New("permission not declared in plugin manifest")
	ErrPermissionDenied       = errors.New("plugin does not have required permission")
	ErrKVKeyNotFound          = errors.New("key not found in plugin KV store")
	ErrValidationRuleNotFound = errors.New("validation rule not found")
	ErrWorkflowRuleNotFound   = errors.New("workflow rule not found")
	ErrTemplateNotFound       = errors.New("industry template not found")
	ErrInvalidSettingsSchema  = errors.New("invalid plugin settings schema")
	ErrInvalidSettings        = errors.New("plugin settings do not match schema")
	ErrInvalidRuleConfig      = errors.New("invalid validation rule configuration")
	ErrPluginHasInstallations = errors.New("cannot delete manifest with active installations")
	ErrWASMBinaryRequired     = errors.New("WASM binary is required for WASM plugins")
	ErrWASMBinaryNotAllowed   = errors.New("config plugins must not have a WASM binary")
)
