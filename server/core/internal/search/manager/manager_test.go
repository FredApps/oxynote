package manager

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/oxynote/oxynote/server/core/internal/document"
	"github.com/oxynote/oxynote/server/core/internal/job"
	"github.com/oxynote/oxynote/server/core/internal/search"
	"github.com/oxynote/oxynote/server/core/pkg/sqlutil"
	"github.com/oxynote/oxynote/server/core/pkg/testutil"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, testutil.IgnoreBleveWorkers())
}

// stubDocument builds a default-branch document with one paragraph.
func stubDocument(branchID xid.ID) *document.Document {
	return &document.Document{
		ID:             xid.New(),
		OrganizationID: "org-1",
		BranchID:       branchID,
		BranchName:     document.DefaultBranch,
		DocumentName:   "Runbook",
		Content: document.RootBlock{
			Type: document.BlockNodeDoc,
			Content: []document.Block{
				{
					Type:  document.BlockNodeParagraph,
					Attrs: document.Attributes{"uid": "p1"},
					Content: []document.Block{
						{Type: document.BlockNodeText, Text: "hello"},
					},
				},
			},
		},
		Default: true,
	}
}

// stubTx is a transaction that does nothing.
type stubTx struct{}

// Commit does nothing.
func (*stubTx) Commit() error {
	return nil
}

// Rollback does nothing.
func (*stubTx) Rollback() error {
	return nil
}

// emptySource is a search.Source with no branches.
type emptySource struct{}

func (emptySource) FetchDocumentBranchesAfter(context.Context, xid.ID, int) ([]document.Document, error) {
	return nil, nil
}

func Test_NewManager(t *testing.T) {
	t.Parallel()

	// error
	jobMan := &JobManagerMock{
		RegisterFunc: func(job.Registrable) error {
			return assert.AnError
		},
	}

	man, err := NewManager(&DBMock{}, &IndexMock{}, jobMan)
	assert.Error(t, err)
	assert.Nil(t, man)

	// success
	jobMan = &JobManagerMock{}

	man, err = NewManager(&DBMock{}, &IndexMock{}, jobMan)
	require.NoError(t, err)
	require.NotNil(t, man)
	assert.NotNil(t, man.db)
	assert.NotNil(t, man.index)
	require.Len(t, jobMan.RegisterCalls(), 1)
	assert.Equal(t, man.queue, jobMan.RegisterCalls()[0].Q)
}

func Test_Manager_Enqueue(t *testing.T) {
	t.Parallel()

	scope := search.BranchScope("org-1", xid.New(), xid.New())
	tx := &stubTx{}

	cc := map[string]struct {
		Queue *QueueMock
		Err   error
	}{
		"Error returned by Queue.Enqueue": {
			Queue: &QueueMock{
				EnqueueFunc: func(context.Context, sqlutil.Tx, search.Job) error {
					return assert.AnError
				},
			},
			Err: assert.AnError,
		},
		"Successful enqueue": {
			Queue: &QueueMock{},
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			man := &Manager{queue: c.Queue}

			err := man.Enqueue(context.Background(), tx, scope)
			testutil.AssertEqualError(t, c.Err, err)

			ff := c.Queue.EnqueueCalls()
			require.Len(t, ff, 1)
			assert.Same(t, tx, ff[0].Tx)
			assert.Equal(t, scope, ff[0].J)
		})
	}
}

func Test_Manager_processJob(t *testing.T) {
	t.Parallel()

	documentID, branchID := xid.New(), xid.New()

	cc := map[string]struct {
		Job  search.Job
		DB   *DBMock
		Err  error
		Call func(t *testing.T, idx *IndexMock)
	}{
		"Branch scope syncs the branch": {
			Job: search.BranchScope("org-1", documentID, branchID),
			DB: &DBMock{
				FetchDocumentUnsafeByBranchIDFunc: func(_ context.Context, branchID xid.ID) (*document.Document, error) {
					return stubDocument(branchID), nil
				},
			},
			Call: func(t *testing.T, idx *IndexMock) {
				require.Len(t, idx.ReplaceBranchCalls(), 1)
				assert.Equal(t, branchID, idx.ReplaceBranchCalls()[0].BranchID)
			},
		},
		"Document scope syncs the document": {
			Job: search.DocumentScope("org-1", documentID),
			DB: &DBMock{
				FetchDocumentBranchesUnsafeFunc: func(context.Context, xid.ID) ([]document.BranchSummary, error) {
					return nil, nil
				},
			},
			Call: func(t *testing.T, idx *IndexMock) {
				require.Len(t, idx.DeleteDocumentCalls(), 1)
				assert.Equal(t, documentID, idx.DeleteDocumentCalls()[0].DocumentID)
			},
		},
		"Organization scope removes the organization": {
			Job: search.OrganizationScope("org-1"),
			DB:  &DBMock{},
			Call: func(t *testing.T, idx *IndexMock) {
				require.Len(t, idx.DeleteOrganizationCalls(), 1)
				assert.Equal(t, "org-1", idx.DeleteOrganizationCalls()[0].OrganizationID)
			},
		},
		"Error returned by Index.DeleteOrganization": {
			Job: search.OrganizationScope("org-1"),
			DB:  &DBMock{},
			Err: assert.AnError,
			Call: func(t *testing.T, idx *IndexMock) {
				require.Len(t, idx.DeleteOrganizationCalls(), 1)
			},
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			idx := &IndexMock{
				DeleteOrganizationFunc: func(context.Context, string) error {
					return c.Err
				},
			}

			man := &Manager{db: c.DB, index: idx}

			err := man.processJob(context.Background(), c.Job)

			if c.Err != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			c.Call(t, idx)
		})
	}

	// applied against a real index, a set of jobs yields the same entries in
	// every order, and applying one twice changes nothing.
	t.Run("Jobs are order-independent and idempotent", func(t *testing.T) {
		t.Parallel()

		kept, gone := stubDocument(xid.New()), stubDocument(xid.New())
		removed := xid.New()

		db := &DBMock{
			FetchDocumentUnsafeByBranchIDFunc: func(_ context.Context, branchID xid.ID) (*document.Document, error) {
				if branchID == kept.BranchID {
					return kept, nil
				}

				return nil, sql.ErrNoRows
			},
			FetchDocumentBranchesUnsafeFunc: func(context.Context, xid.ID) ([]document.BranchSummary, error) {
				return nil, nil
			},
		}

		set := []search.Job{
			search.BranchScope("org-1", kept.ID, kept.BranchID),
			search.BranchScope("org-1", gone.ID, gone.BranchID),
			search.DocumentScope("org-1", removed),
			search.OrganizationScope("org-2"),
		}

		orders := [][]int{
			{0, 1, 2, 3},
			{3, 2, 1, 0},
			{1, 0, 3, 2},
			{0, 0, 2, 1, 3, 1},
		}

		var expected []search.Block

		for i, order := range orders {
			idx, err := search.Open(context.Background(), slog.New(slog.DiscardHandler), filepath.Join(t.TempDir(), "index"), emptySource{})
			require.NoError(t, err)

			// the branches to remove start out present, so the removals
			// have something to do.
			require.NoError(t, idx.ReplaceBranch(context.Background(), gone.BranchID, search.Entries(*gone)))

			man := &Manager{db: db, index: idx}

			for _, j := range order {
				require.NoError(t, man.processJob(context.Background(), set[j]))
			}

			blocks, err := idx.SearchDocumentBlocks(context.Background(), "org-1", "hello runbook", 50)
			require.NoError(t, err)
			require.NoError(t, idx.Close())

			if i == 0 {
				expected = blocks
				require.Len(t, expected, 2, "the kept branch's paragraph and name")

				continue
			}

			assert.ElementsMatch(t, expected, blocks)
		}
	})
}

func Test_Manager_syncBranch(t *testing.T) {
	t.Parallel()

	branchID := xid.New()

	cc := map[string]struct {
		FetchErr   error
		ReplaceErr error
		DeleteErr  error
		Err        error
		Replaced   int
		Deleted    int
	}{
		"Error returned by DB.FetchDocumentUnsafeByBranchID": {
			FetchErr: assert.AnError,
			Err:      assert.AnError,
		},
		"Missing branch is removed from the index": {
			FetchErr: sql.ErrNoRows,
			Deleted:  1,
		},
		"Error returned by Index.DeleteBranch": {
			FetchErr:  sql.ErrNoRows,
			DeleteErr: assert.AnError,
			Err:       assert.AnError,
			Deleted:   1,
		},
		"Error returned by Index.ReplaceBranch": {
			ReplaceErr: assert.AnError,
			Err:        assert.AnError,
			Replaced:   1,
		},
		"Branch entries are replaced": {
			Replaced: 1,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			doc := stubDocument(branchID)

			db := &DBMock{
				FetchDocumentUnsafeByBranchIDFunc: func(context.Context, xid.ID) (*document.Document, error) {
					if c.FetchErr != nil {
						return nil, c.FetchErr
					}

					return doc, nil
				},
			}

			idx := &IndexMock{
				ReplaceBranchFunc: func(context.Context, xid.ID, map[string]search.Block) error {
					return c.ReplaceErr
				},
				DeleteBranchFunc: func(context.Context, xid.ID) error {
					return c.DeleteErr
				},
			}

			man := &Manager{db: db, index: idx}

			err := man.syncBranch(context.Background(), branchID)

			if c.Err != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, idx.ReplaceBranchCalls(), c.Replaced)
			require.Len(t, idx.DeleteBranchCalls(), c.Deleted)

			if c.Replaced == 1 {
				call := idx.ReplaceBranchCalls()[0]
				assert.Equal(t, branchID, call.BranchID)
				assert.Equal(t, search.Entries(*doc), call.Entries)
			}

			if c.Deleted == 1 {
				assert.Equal(t, branchID, idx.DeleteBranchCalls()[0].BranchID)
			}
		})
	}
}

func Test_Manager_syncDocument(t *testing.T) {
	t.Parallel()

	documentID := xid.New()
	branches := []document.BranchSummary{{BranchID: xid.New()}, {BranchID: xid.New()}}

	cc := map[string]struct {
		Branches   []document.BranchSummary
		FetchErr   error
		BranchErr  error
		DeleteErr  error
		Err        error
		Replaced   int
		DocDeleted int
	}{
		"Error returned by DB.FetchDocumentBranchesUnsafe": {
			FetchErr: assert.AnError,
			Err:      assert.AnError,
		},
		"Document without branches is removed from the index": {
			DocDeleted: 1,
		},
		"Error returned by Index.DeleteDocument": {
			DeleteErr:  assert.AnError,
			Err:        assert.AnError,
			DocDeleted: 1,
		},
		"Every branch is synced": {
			Branches: branches,
			Replaced: 2,
		},
		"Error returned by a branch sync": {
			Branches:  branches,
			BranchErr: assert.AnError,
			Err:       assert.AnError,
			Replaced:  1,
		},
	}

	for cn, c := range cc {
		t.Run(cn, func(t *testing.T) {
			t.Parallel()

			db := &DBMock{
				FetchDocumentBranchesUnsafeFunc: func(context.Context, xid.ID) ([]document.BranchSummary, error) {
					return c.Branches, c.FetchErr
				},
				FetchDocumentUnsafeByBranchIDFunc: func(_ context.Context, branchID xid.ID) (*document.Document, error) {
					return stubDocument(branchID), nil
				},
			}

			idx := &IndexMock{
				ReplaceBranchFunc: func(context.Context, xid.ID, map[string]search.Block) error {
					return c.BranchErr
				},
				DeleteDocumentFunc: func(context.Context, xid.ID) error {
					return c.DeleteErr
				},
			}

			man := &Manager{db: db, index: idx}

			err := man.syncDocument(context.Background(), documentID)

			if c.Err != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, idx.ReplaceBranchCalls(), c.Replaced)
			require.Len(t, idx.DeleteDocumentCalls(), c.DocDeleted)

			if c.DocDeleted == 1 {
				assert.Equal(t, documentID, idx.DeleteDocumentCalls()[0].DocumentID)
			}
		})
	}
}
