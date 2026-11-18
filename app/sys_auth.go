package app

import (
	"fmt"

	"github.com/webitel/webitel-go-kit/pkg/presign"

	"github.com/webitel/storage/pkg/sysurl"
)

func (a *App) initSysVerifier() error {
	signer, err := presign.NewMAC()
	if err != nil {
		return fmt.Errorf("unable to init /sys signer: %w", err)
	}
	a.sysVerifier, err = sysurl.NewVerifier(signer)
	if err != nil {
		return fmt.Errorf("unable to init /sys verifier: %w", err)
	}
	return nil
}

func (a *App) SysVerifier() *sysurl.Verifier {
	return a.sysVerifier
}
