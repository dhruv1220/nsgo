package pkg

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"sync"
)

// A Trust verifies packages: it holds the roots an app trusts, each root's newest revocation list and the highest build accepted of each app. Its methods may be called from any goroutine; verifying hashes the whole package, so call Verify off the engine's.
type Trust struct {
	mu           sync.Mutex
	roots        []ed25519.PublicKey
	state        trustState
	dir          string
	requireFresh bool
}

type trustState struct {
	// Lists holds each root's newest revocation list, by the root's key ID.
	Lists  map[string]*Revocations `json:"revocationLists,omitempty"`
	Builds map[string]uint64       `json:"builds,omitempty"`
	// Revocations is the one list a state written before lists were kept per root holds, which OpenTrust files under the root that signed it.
	Revocations *Revocations `json:"revocations,omitempty"`
}

// NewTrust returns a Trust whose state lasts as long as it does.
func NewTrust() *Trust {
	return &Trust{state: trustState{Lists: map[string]*Revocations{}, Builds: map[string]uint64{}}}
}

// OpenTrust returns a Trust whose state is kept in dir, made when first written, so rollback protection and revocations survive restarts. On the web, which has no files, the page's localStorage keeps it under a key of dir's.
func OpenTrust(dir string) (*Trust, error) {
	t := NewTrust()
	t.dir = dir
	data, err := readState(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrState, err)
	}
	if err := json.Unmarshal(data, &t.state); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrState, err)
	}
	if t.state.Lists == nil {
		t.state.Lists = map[string]*Revocations{}
	}
	if t.state.Builds == nil {
		t.state.Builds = map[string]uint64{}
	}
	// A signed list names its signer, and is checked against that root whenever it is used.
	if old := t.state.Revocations; old != nil {
		if _, ok := t.state.Lists[old.s.Signer]; !ok {
			t.state.Lists[old.s.Signer] = old
		}
		t.state.Revocations = nil
	}
	return t, nil
}

// AddRoot trusts the certificates root signs, and its revocation lists.
func (t *Trust) AddRoot(root ed25519.PublicKey) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !slices.ContainsFunc(t.roots, func(r ed25519.PublicKey) bool { return r.Equal(root) }) {
		t.roots = append(t.roots, root)
	}
}

// RequireFreshRevocations has Verify and Accept refuse every package, with ErrExpired, while a trusted root's newest revocation list has passed its NotAfter, until UpdateRevocations takes a newer one. A root the Trust holds no list of yet is not refused, so a host that requires fresh lists takes each root's list before it verifies. Without it, Verify reports such roots in Package.ExpiredRevocations.
func (t *Trust) RequireFreshRevocations(require bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.requireFresh = require
}

// UpdateRevocations takes list when a trusted root signed it and it is newer than the list of that root the Trust holds. A root's lists replace only its own, so while several roots are trusted, as when they rotate, none undoes what another revoked.
func (t *Trust) UpdateRevocations(list *Revocations) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	info, err := list.open(t.roots)
	if err != nil {
		return err
	}
	signer := list.s.Signer
	if held, ok := t.state.Lists[signer]; ok {
		if have, err := held.open(t.roots); err == nil && info.Sequence <= have.Sequence {
			return fmt.Errorf("%w: list %d, where the Trust holds %d of root %s", ErrStale, info.Sequence, have.Sequence, signer)
		}
	}
	t.state.Lists[signer] = list
	return t.save()
}

// revoked is what the lists of the roots still trusted revoke together.
type revoked struct {
	keys, packages []string
	// expired names the roots whose list has passed its NotAfter.
	expired []string
}

func (t *Trust) revoked() revoked {
	var r revoked
	for _, root := range t.roots {
		id := KeyID(root)
		list, ok := t.state.Lists[id]
		if !ok {
			continue
		}
		info, err := list.open([]ed25519.PublicKey{root})
		if err != nil {
			continue
		}
		r.keys = append(r.keys, info.Keys...)
		r.packages = append(r.packages, info.Packages...)
		if info.expired() {
			r.expired = append(r.expired, id)
		}
	}
	return r
}

// check refuses a package r revokes, and every package while a list expired if fresh ones are required.
func (r revoked) check(requireFresh bool, p *Package) error {
	if requireFresh && len(r.expired) > 0 {
		return fmt.Errorf("%w: root %s", ErrExpired, r.expired[0])
	}
	if slices.Contains(r.keys, p.KeyID) {
		return fmt.Errorf("%w: release key %s", ErrRevoked, p.KeyID)
	}
	if slices.Contains(r.packages, p.SHA256) {
		return fmt.Errorf("%w: package %s", ErrRevoked, p.SHA256)
	}
	return nil
}

// Verify checks a package: its certificate's root, its release key and itself not revoked by any trusted root's list, its signature, its certificate covering its app and signing time, its minimum runtime, its files' hashes and its build no older than one accepted before.
func (t *Trust) Verify(data []byte) (*Package, error) {
	t.mu.Lock()
	roots := slices.Clone(t.roots)
	r := t.revoked()
	requireFresh := t.requireFresh
	t.mu.Unlock()
	p, err := open(data, roots, r.keys)
	if err != nil {
		return nil, err
	}
	if err := r.check(requireFresh, p); err != nil {
		return nil, err
	}
	if accepted, ok := t.AcceptedBuild(p.App.ID); ok && p.App.Build < accepted {
		return nil, fmt.Errorf("%w: %s build %d, where build %d was accepted", ErrRollback, p.App.ID, p.App.Build, accepted)
	}
	p.ExpiredRevocations = r.expired
	return p, nil
}

// Accept records a package's build as the oldest its app may run from now on. A host accepts a package once it proved healthy, so until then the app may go back to what it accepted before. A package or key revoked since it was verified is not accepted.
func (t *Trust) Accept(p *Package) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.revoked().check(t.requireFresh, p); err != nil {
		return err
	}
	if have, ok := t.state.Builds[p.App.ID]; ok && p.App.Build <= have {
		return nil
	}
	t.state.Builds[p.App.ID] = p.App.Build
	return t.save()
}

// AcceptedBuild is the highest build of app accepted.
func (t *Trust) AcceptedBuild(app string) (uint64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	b, ok := t.state.Builds[app]
	return b, ok
}

// save writes the state whole.
func (t *Trust) save() error {
	if t.dir == "" {
		return nil
	}
	data, err := json.MarshalIndent(t.state, "", "  ")
	if err != nil {
		return err
	}
	if err := writeState(t.dir, data); err != nil {
		return fmt.Errorf("%w: %v", ErrState, err)
	}
	return nil
}
