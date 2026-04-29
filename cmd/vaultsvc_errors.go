package cmd

import (
	"errors"
	"fmt"

	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

func mapVaultSvcError(err error, fallback string) error {
	var vaultErr *vaultsvc.Error
	if errors.As(err, &vaultErr) {
		switch vaultErr.Kind {
		case vaultsvc.ErrNotFound, vaultsvc.ErrFieldNotFound:
			return errorspkg.NewCLIError(errorspkg.ExitNotFound, vaultErr.Message, errorspkg.ErrEntryNotFound)
		case vaultsvc.ErrWriteFailed, vaultsvc.ErrReadFailed:
			return errorspkg.NewCLIError(errorspkg.ExitGeneralError, vaultErr.Message, vaultErr.Cause)
		}
	}

	return fmt.Errorf("%s: %w", fallback, err)
}
