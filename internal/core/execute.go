// package: core / orchestration
// type:    logic
// job:     turn an Operation into the library call that answers it, and its result into a Stream
// limits:  a seam, no graph behaviour; one archive snapshot per request (-> ranke-go)
//
// Every arm is one call on the archive; an arm that grows logic is in the wrong repository.
//
// A request resolves its snapshot once. RA_k is immutable, so answering from the one
// opened at the start is what keeps a streaming query consistent under a merge.
package core

import (
	"context"
	"errors"
	"fmt"
	"io"

	ranke "github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// execute runs the operation against the ports and returns its response stream.
func (c *Core) execute(ctx context.Context, req *Request) (Stream, error) {
	switch req.Op {
	case OpHealthGet:
		// Health answers from the signer alone, never the archive: it is wanted
		// precisely when the stack is too broken to open one.
		return c.health(ctx, req)
	case OpLayerList, OpLayerInfo:
		return c.layers(req)
	}

	archive, err := c.archive(ctx)
	if err != nil {
		return nil, err
	}
	req.Report.step("archive snapshot at head %s", archive.Head())

	switch req.Op {
	case OpClaimContribute:
		return c.contribute(ctx, req)
	case OpClaimQuery:
		return c.query(ctx, req, archive)
	case OpClaimGet:
		return c.claim(ctx, req, archive)
	case OpClaimContent:
		return c.claimContent(ctx, req, archive)
	case OpBranchHead:
		return c.branchHead(ctx, req, archive)
	case OpBranchList:
		return c.branchList(ctx, req, archive)
	case OpVerificationStart, OpVerificationList, OpVerificationGet,
		OpVerificationCancel, OpVerificationDelete:
		return c.verification(ctx, req, archive)
	default:
		req.Report.step("execute %v: not yet implemented", req.Op)
		return nil, ErrNotImplemented
	}
}

// archive resolves the immutable snapshot this request reads from.
func (c *Core) archive(ctx context.Context) (ranke.Archive, error) {
	if c.seq == nil {
		return nil, fmt.Errorf("%w: no sequencer, so no archive to read", ErrNotImplemented)
	}
	archive, err := c.seq.GetArchive(ctx)
	if err != nil {
		return nil, mapLibError(err)
	}
	return archive, nil
}

// --- the read arms --------------------------------------------------------

// query runs the RQL read and serves its result set.
func (c *Core) query(ctx context.Context, req *Request, archive ranke.Archive) (Stream, error) {
	if req.Query == nil {
		return nil, fmt.Errorf("%w: no query", ErrNotFound)
	}
	// Ask for an explicit encoding: the engine's native form is Go objects.
	q := *req.Query
	if q.Output.Encoding == "" || q.Output.Encoding == ranke.ResultNative {
		q.Output.Encoding = ranke.ResultJSON
	}
	results, err := archive.Query(ctx, q)
	if err != nil {
		return nil, mapLibError(err)
	}
	req.Report.step("query executed")
	return newQueryStream(results, q.Output.Encoding), nil
}

// claim serves one claim as the stored CBOR its id signs.
func (c *Core) claim(ctx context.Context, req *Request, archive ranke.Archive) (Stream, error) {
	claim, err := c.scopedClaim(ctx, req, archive)
	if err != nil {
		return nil, err
	}
	payload, err := claim.EncodeCBOR(ranke.FormOriginal)
	if err != nil {
		return nil, mapLibError(err)
	}
	req.Report.step("claim %s read", req.ClaimID)
	return &bytesStream{payload: payload, contentType: mediaCBOR}, nil
}

// claimContent streams one claim's content, which inherits that claim's scope.
func (c *Core) claimContent(ctx context.Context, req *Request, archive ranke.Archive) (Stream, error) {
	if req.ClaimID == nil {
		return nil, fmt.Errorf("%w: no claim id", ErrNotFound)
	}
	var (
		content io.Reader
		err     error
	)
	if req.Branch == Universe {
		content, err = archive.GetClaimContent(ctx, req.ClaimID)
	} else {
		var branch ranke.Branch
		if branch, err = c.branch(ctx, req, archive); err == nil {
			content, err = branch.GetClaimContent(ctx, req.ClaimID)
		}
	}
	if err != nil {
		return nil, mapLibError(err)
	}
	req.Report.step("content of %s read", req.ClaimID)
	return &blobStream{content: readCloser(content)}, nil
}

// branchHead serves a branch's current head id.
func (c *Core) branchHead(ctx context.Context, req *Request, archive ranke.Archive) (Stream, error) {
	branch, err := c.branch(ctx, req, archive)
	if err != nil {
		return nil, err
	}
	req.Report.step("branch %q head read", req.Branch)
	return &jsonStream{value: branchHead{Head: branch.Head().String()}}, nil
}

// branchList serves the branch table's branches by name and head.
func (c *Core) branchList(ctx context.Context, req *Request, archive ranke.Archive) (Stream, error) {
	branches, err := archive.GetBranches(ctx)
	if err != nil {
		return nil, mapLibError(err)
	}
	named := make([]branchEntry, 0, len(branches))
	for _, b := range branches {
		named = append(named, branchEntry{Name: b.Name(), Head: b.Head().String()})
	}
	req.Report.step("%d branches listed", len(named))
	return &jsonStream{value: branchList{Branches: named}}, nil
}

// scopedClaim reads the request's claim within its scope: a branch confines the read to
// that branch's closure, $universe reaches any claim the Universe holds.
func (c *Core) scopedClaim(ctx context.Context, req *Request, archive ranke.Archive) (ranke.Claim, error) {
	if req.ClaimID == nil {
		return nil, fmt.Errorf("%w: no claim id", ErrNotFound)
	}
	if req.Branch == Universe {
		claim, err := archive.GetClaim(ctx, req.ClaimID)
		return claim, mapLibError(err)
	}
	branch, err := c.branch(ctx, req, archive)
	if err != nil {
		return nil, err
	}
	claim, err := branch.GetClaim(ctx, req.ClaimID)
	return claim, mapLibError(err)
}

// branch resolves the request's branch. GetBranch's not-found is an unexported sentinel
// wrapping nothing, so HasBranch is the only way to classify it.
func (c *Core) branch(ctx context.Context, req *Request, archive ranke.Archive) (ranke.Branch, error) {
	branch, err := archive.GetBranch(ctx, req.Branch)
	if err == nil {
		return branch, nil
	}
	if exists, hasErr := archive.HasBranch(ctx, req.Branch); hasErr == nil && !exists {
		return nil, fmt.Errorf("%w: branch %q", ErrNotFound, req.Branch)
	}
	return nil, mapLibError(err)
}

// --- operational arms -----------------------------------------------------

// health reports liveness and the signing identity, taken from the signer so it answers
// when no archive opened.
func (c *Core) health(ctx context.Context, req *Request) (Stream, error) {
	report := Health{Status: "ok"}
	if c.signer != nil {
		report.Signer = signer.Identity(ctx, c.signer)
	}
	req.Report.step("health reported")
	return &jsonStream{value: report}, nil
}

// layers reports what config retained of each layer: a name and a type.
func (c *Core) layers(req *Request) (Stream, error) {
	req.Report.step("%d storage layers listed", len(c.layerInfo))
	return &jsonStream{value: layerList{Layers: c.layerInfo}}, nil
}

// --- wire values ----------------------------------------------------------

// branchHead is one branch's current head.
type branchHead struct {
	Head string `json:"head"`
}

// branchEntry names one branch and the head it points at.
type branchEntry struct {
	Name string `json:"name"`
	Head string `json:"head"`
}

// branchList is the branch table's branches.
type branchList struct {
	Branches []branchEntry `json:"branches"`
}

// layerList is the storage stack's layers, top to bottom.
type layerList struct {
	Layers []StorageLayer `json:"layers"`
}

// --- error mapping --------------------------------------------------------

// mapLibError resolves a library failure onto its sentinel. Absent and out-of-closure
// arrive as one ErrNotFound, which is what keeps them indistinguishable.
func mapLibError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ranke.ErrNotFound):
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	case errors.Is(err, ranke.ErrUnsupported):
		return fmt.Errorf("%w: %v", ErrNotImplemented, err)
	default:
		return err
	}
}

// readCloser adapts a reader the library handed back, which may or may not own
// resources, to the Close the stream contract requires.
func readCloser(r io.Reader) io.ReadCloser {
	if rc, ok := r.(io.ReadCloser); ok {
		return rc
	}
	return io.NopCloser(r)
}

// --- the contribute arm ---------------------------------------------------

// contribute merges a contribution body: authorize what it declares, fill, verify,
// persist, merge — every step the library's, in order.
//
// The stream declares its branches in its first record, so the C right is settled before
// any of the body is read, and the rest then streams through the library's own drain.
func (c *Core) contribute(ctx context.Context, req *Request) (Stream, error) {
	if req.Body == nil {
		return nil, fmt.Errorf("%w: no contribution body", ErrInvalidRequest)
	}

	wire := ranke.NewWireReader(req.Body)
	branches, err := wire.Branches()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if len(branches) == 0 {
		return nil, fmt.Errorf("%w: the contribution declares no branches", ErrInvalidRequest)
	}
	for _, branch := range branches {
		if err := c.allow(req, access.Contribute, branch); err != nil {
			return nil, err
		}
	}

	contribution, err := c.seq.NewContribution(ctx)
	if err != nil {
		return nil, mapLibError(err)
	}
	if err := contribution.AddWire(ctx, wire); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	verified, err := contribution.CompleteAndVerify(ctx)
	if err != nil {
		return nil, mapLibError(err)
	}
	mergable, err := verified.Persist(ctx)
	if err != nil {
		return nil, mapLibError(err)
	}
	receipt, err := c.seq.Merge(ctx, mergable)
	if err != nil {
		return nil, mapLibError(err)
	}

	ids := make([]string, 0, len(verified.Ids()))
	for _, id := range verified.Ids() {
		ids = append(ids, id.String())
	}
	req.Report.step("merged %d claim(s) onto %v at head %s", len(ids), branches, receipt.Head())
	return &jsonStream{value: Contribution{Head: receipt.Head().String(), Ids: ids}}, nil
}
