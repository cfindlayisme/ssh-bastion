package mocks

import _ "embed"

//go:embed valid_config.yaml
var ValidConfig []byte

//go:embed missing_authorized_key.yaml
var MissingAuthorizedKey []byte

//go:embed no_targets.yaml
var NoTargets []byte

//go:embed defaults_only.yaml
var DefaultsOnly []byte
