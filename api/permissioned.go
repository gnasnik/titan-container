package api

import (
	"github.com/filecoin-project/go-jsonrpc/auth"
)

const (
	// When changing these, update docs/API.md too

	PermRead     auth.Permission = "read" // default
	PermWrite    auth.Permission = "write"
	PermSign     auth.Permission = "sign"     // Use wallet keys for signing
	PermProvider auth.Permission = "provider" // only for provider
	PermAdmin    auth.Permission = "admin"    // Manage permissions
)

var AllPermissions = []auth.Permission{PermRead, PermWrite, PermSign, PermProvider, PermAdmin}
var DefaultPerms = []auth.Permission{PermRead}

func permissionedProxies(in, out interface{}) {
	outs := GetInternalStructs(out)
	for _, o := range outs {
		auth.PermissionedProxy(AllPermissions, DefaultPerms, in, o)
	}
}

func PermissionedManagerAPI(a Manager) Manager {
	var out ManagerStruct
	permissionedProxies(a, &out)
	return &out
}

func PermissionedProviderAPI(a Provider) Provider {
	var out ProviderStruct
	permissionedProxies(a, &out)
	return &out
}
