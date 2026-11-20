//go:build windows

package runtime

import "os"

func fileOwnerID(os.FileInfo) (uint32, bool) { return 0, false }
func currentUserID() uint32                 { return 0 }
func isRootUser() bool                      { return false }
func dockerHostIdentity() (int, int, bool)   { return 0, 0, false }
