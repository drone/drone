package capability

import (
	"testing"

	"code.google.com/p/go.net/context"
	"github.com/franela/goblin"
)

func TestBlobstore(t *testing.T) {
	caps := new(Capability)
	caps.Set(Registration, true)

	ctx := NewContext(context.Background(), caps)

	g := goblin.Goblin(t)
	g.Describe("Capabilities", func() {

		g.It("Should get capabilities from context", func() {
			g.Assert(Enabled(ctx, Registration)).Equal(true)
			g.Assert(Enabled(ctx, "Fake Key")).Equal(false)
		})
	})
}
