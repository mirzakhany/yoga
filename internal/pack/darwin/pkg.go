package darwin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// createPKG builds a macOS Installer package via productbuild.
// With installer_identity set to a "3rd Party Mac Developer Installer"
// certificate, the result is suitable for Mac App Store upload (after the
// .app is signed with the matching Application identity + entitlements).
func createPKG(appRoot string, opts Options) (string, error) {
	pkgName := fmt.Sprintf("%s-%s.pkg", sanitizeFile(opts.Name), opts.Version)
	pkgPath := filepath.Join(opts.OutDir, pkgName)
	_ = os.Remove(pkgPath)

	args := []string{
		"--component", appRoot,
		"/Applications",
	}
	if opts.Sign.InstallerIdentity != "" {
		args = append(args, "--sign", opts.Sign.InstallerIdentity)
	}
	args = append(args, pkgPath)

	cmd := exec.Command("productbuild", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("darwin: productbuild: %w", err)
	}
	return pkgPath, nil
}
