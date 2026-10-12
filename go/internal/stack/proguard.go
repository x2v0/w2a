package stack

import (
	"bytes"
	"fmt"

	"github.com/x2v0/w2a/go/internal/apk"
)

// StackProguardPath is where stack R8 rules are appended inside the APK.
// R8 (release builds) reads "META-INF/proguard/*" at build time, but for
// pre-built APKs this is informational — the rules were already applied
// to the stack DEX upstream.
const StackProguardPath = "META-INF/wta-stack-rules.pro"

// AppendProguard merges bundle R8 rules into the target APK under
// META-INF/wta-stack-rules.pro. Multiple stacks append to the same file.
// Returns the new total size in bytes.
func AppendProguard(target *apk.APK, b *Bundle) (int, error) {
	if len(b.Proguard) == 0 {
		return 0, nil
	}
	existing := target.Archive.Get(StackProguardPath)
	var buf bytes.Buffer
	if existing != nil {
		buf.Write(existing.Data)
		if existing.Data[len(existing.Data)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	buf.WriteString("# --- stack: " + b.Metadata.ID + " ---\n")
	buf.Write(b.Proguard)
	if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	target.Archive.ReplaceDeflated(StackProguardPath, buf.Bytes())
	return buf.Len(), nil
}

// ensure fmt import used
var _ = fmt.Sprintf
