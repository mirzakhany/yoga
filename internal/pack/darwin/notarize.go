package darwin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// notarizeApp zips the signed app, submits it with notarytool, waits for
// the verdict and staples the ticket onto the app.
//
// Credentials come from the environment: NOTARY_KEYCHAIN_PROFILE (a
// profile saved with `xcrun notarytool store-credentials`), or APPLE_ID,
// APPLE_TEAM_ID and APPLE_APP_SPECIFIC_PASSWORD.
func notarizeApp(appRoot string) error {
	creds, err := notaryCredentials()
	if err != nil {
		return err
	}

	zip := filepath.Join(filepath.Dir(appRoot), "."+filepath.Base(appRoot)+".notarize.zip")
	_ = os.Remove(zip)
	defer os.Remove(zip)
	if err := run("ditto", "-c", "-k", "--keepParent", appRoot, zip); err != nil {
		return fmt.Errorf("darwin: notarize: zip: %w", err)
	}

	args := append([]string{"notarytool", "submit", zip, "--wait"}, creds...)
	if err := run("xcrun", args...); err != nil {
		return fmt.Errorf("darwin: notarize: submit: %w", err)
	}
	if err := run("xcrun", "stapler", "staple", appRoot); err != nil {
		return fmt.Errorf("darwin: notarize: staple: %w", err)
	}
	return nil
}

func notaryCredentials() ([]string, error) {
	if p := os.Getenv("NOTARY_KEYCHAIN_PROFILE"); p != "" {
		return []string{"--keychain-profile", p}, nil
	}
	id, team, pass := os.Getenv("APPLE_ID"), os.Getenv("APPLE_TEAM_ID"), os.Getenv("APPLE_APP_SPECIFIC_PASSWORD")
	if id == "" || team == "" || pass == "" {
		return nil, fmt.Errorf("darwin: notarize: set NOTARY_KEYCHAIN_PROFILE, or APPLE_ID, APPLE_TEAM_ID and APPLE_APP_SPECIFIC_PASSWORD")
	}
	return []string{"--apple-id", id, "--team-id", team, "--password", pass}, nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
