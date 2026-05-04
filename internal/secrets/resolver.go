// Package secrets provides secret reference resolution and command execution
// with environment variable injection for the Secure Secret Execution Flow.
package secrets

import (
	"errors"
	"fmt"
	"strings"

	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

// ResolveSecretRef resolves a secret reference against the vault service.
// The ref can be a bare entry path (e.g. "work/aws") which returns the full
// entry data, or a path.field reference (e.g. "work/aws.password") which
// returns a specific field value. The path.field syntax is only used if the
// candidate path and field actually exist in the vault.
func ResolveSecretRef(svc *vaultsvc.Service, ref string) (string, error) {
	path := ref
	field := ""

	if idx := strings.LastIndex(ref, "."); idx > 0 {
		candidatePath := ref[:idx]
		candidateField := ref[idx+1:]

		if _, readErr := svc.GetField(candidatePath, candidateField); readErr == nil {
			path = candidatePath
			field = candidateField
		}
	}

	value, err := svc.GetField(path, field)
	if err != nil {
		var svcErr *vaultsvc.Error
		if errors.As(err, &svcErr) {
			if svcErr.Kind == vaultsvc.ErrNotFound {
				return "", fmt.Errorf("secret ref not found: %s", path)
			}
			if svcErr.Kind == vaultsvc.ErrFieldNotFound {
				return "", fmt.Errorf("field not found in secret ref %s.%s", path, field)
			}
		}
		return "", fmt.Errorf("cannot resolve secret ref %s: %w", ref, err)
	}

	return fmt.Sprintf("%v", value), nil
}
