//go:build !embeddata

package embedded

import "io/fs"

var files fs.FS
