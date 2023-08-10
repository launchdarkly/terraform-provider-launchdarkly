//go:build tools
// +build tools

package tools

import (
	_ "github.com/ashanbrown/gofmts/cmd/gofmts"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
