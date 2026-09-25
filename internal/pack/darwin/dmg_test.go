package darwin

import "testing"

func TestMountPoint(t *testing.T) {
	out := "/dev/disk4          \tGUID_partition_scheme          \t\n" +
		"/dev/disk4s1        \tApple_HFS                      \t/Volumes/Chapar Installer 1\n"
	if got := mountPoint(out); got != "/Volumes/Chapar Installer 1" {
		t.Fatalf("mountPoint = %q", got)
	}
	if got := mountPoint("/dev/disk4\tGUID_partition_scheme\t\n"); got != "" {
		t.Fatalf("mountPoint without volume = %q", got)
	}
}
