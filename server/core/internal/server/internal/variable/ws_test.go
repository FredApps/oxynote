package variable

import (
	"context"
	"testing"

	"github.com/guregu/null/v5"
	"github.com/oxynote/oxynote/server/core/internal/server/internal/auth"
	wsMock "github.com/oxynote/wetsocks/wsserver/_mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sessionCtx returns a subscriber context carrying a session of the user.
func sessionCtx(userID, organizationID string) context.Context {
	return auth.AddSessionToContext(context.Background(), auth.Session{
		UserID:               userID,
		ActiveOrganizationID: organizationID,
	})
}

func Test_Handler_BindChange(t *testing.T) {
	t.Parallel()

	hdl := &Handler{}

	hdl.BindChange(&wsMock.Topic{})
	require.NotNil(t, hdl.changeCallback)
}

func Test_Handler_NotifyChange(t *testing.T) {
	t.Parallel()

	cc := map[string]struct {
		UserID null.String
		Allow  []context.Context
		Deny   []context.Context
	}{
		"Organization change reaches every member": {
			Allow: []context.Context{sessionCtx("u1", "org1"), sessionCtx("u2", "org1")},
			Deny:  []context.Context{sessionCtx("u1", "org2"), context.Background()},
		},
		"Value change reaches only the user": {
			UserID: null.StringFrom("u1"),
			Allow:  []context.Context{sessionCtx("u1", "org1")},
			Deny:   []context.Context{sessionCtx("u2", "org1"), context.Background()},
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			hdl := &Handler{}

			// notifying before binding must be a safe no-op.
			hdl.NotifyChange("org1", c.UserID)

			tpc := &wsMock.Topic{}

			hdl.BindChange(tpc)
			hdl.NotifyChange("org1", c.UserID)

			pubs := tpc.PublishManyCalls()
			require.Len(t, pubs, 1)
			assert.Equal(t, ChangeMessage{}, pubs[0].Payload)

			for _, ctx := range c.Allow {
				assert.True(t, pubs[0].Filter(ctx, "topic"))
			}

			for _, ctx := range c.Deny {
				assert.False(t, pubs[0].Filter(ctx, "topic"))
			}
		})
	}
}
