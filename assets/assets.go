// Package assets exposes cache-busted URLs for static assets.
//
// The version suffix on a stylesheet URL (?v=<hash>) invalidates browser
// caches exactly when the file's bytes change, and remains stable across
// restarts when the bytes don't change. Without this, a redeploy that
// changes site.css but leaves the URL unchanged leaves users looking at
// whatever CSS their browser cached last — which is how a profile-aware
// accent cascade can ship correctly in code and still render wrong in
// the browser.
package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sync"
)

const cssPath = "static/css/site.css"

var (
	cssVersionOnce sync.Once
	cssVersion     string
)

// CSSURL returns the versioned URL for site.css. The version is the
// first 8 hex chars of the sha256 of the file contents, computed once
// per process at first call. If the file can't be read, the version
// falls back to "dev" so pages still render in degraded local setups.
func CSSURL() string {
	cssVersionOnce.Do(func() {
		data, err := os.ReadFile(cssPath)
		if err != nil {
			cssVersion = "dev"
			return
		}
		sum := sha256.Sum256(data)
		cssVersion = hex.EncodeToString(sum[:])[:8]
	})
	return "/static/css/site.css?v=" + cssVersion
}
