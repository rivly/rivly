package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

const (
	cloneTimeout = 3 * time.Minute
	listTimeout  = 30 * time.Second
)

var (
	ErrAuth     = errors.New("git authentication failed")
	ErrNotFound = errors.New("repository not found")
	ErrRef      = errors.New("branch or tag not found")
)

type Options struct {
	URL      string
	Ref      string
	Username string
	Token    string
}

func Clone(ctx context.Context, dir string, opts Options) (string, error) {
	target, err := NormalizeURL(opts.URL)
	if err != nil {
		return "", err
	}
	opts.URL = target

	ctx, cancel := context.WithTimeout(ctx, cloneTimeout)
	defer cancel()

	repo, err := attempt(ctx, dir, opts, plumbing.NewBranchReferenceName)
	if err != nil && opts.Ref != "" {
		repo, err = attempt(ctx, dir, opts, plumbing.NewTagReferenceName)
	}
	if err != nil {
		return "", mapError(err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("resolve head: %w", err)
	}
	return head.Hash().String(), nil
}

func attempt(ctx context.Context, dir string, opts Options, refName func(string) plumbing.ReferenceName) (*git.Repository, error) {
	if err := reset(dir); err != nil {
		return nil, err
	}

	cloneOpts := &git.CloneOptions{
		URL:          opts.URL,
		Depth:        1,
		SingleBranch: true,
		Tags:         git.NoTags,
	}
	if opts.Token != "" {
		username := opts.Username
		if username == "" {
			username = "git"
		}
		cloneOpts.Auth = &githttp.BasicAuth{Username: username, Password: opts.Token}
	}
	if opts.Ref != "" {
		cloneOpts.ReferenceName = refName(opts.Ref)
	}

	return git.PlainCloneContext(ctx, dir, false, cloneOpts)
}

func reset(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clear repository directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create repository directory: %w", err)
	}
	return nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, transport.ErrAuthenticationRequired),
		errors.Is(err, transport.ErrAuthorizationFailed):
		return ErrAuth
	case errors.Is(err, transport.ErrRepositoryNotFound):
		return ErrNotFound
	case errors.Is(err, plumbing.ErrReferenceNotFound),
		errors.Is(err, git.NoMatchingRefSpecError{}):
		return ErrRef
	}
	if strings.Contains(err.Error(), "couldn't find remote ref") ||
		strings.Contains(err.Error(), "reference not found") {
		return ErrRef
	}
	return err
}

func RemoteHash(ctx context.Context, opts Options) (string, error) {
	target, err := NormalizeURL(opts.URL)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{target},
	})

	listOpts := &git.ListOptions{}
	if opts.Token != "" {
		username := opts.Username
		if username == "" {
			username = "git"
		}
		listOpts.Auth = &githttp.BasicAuth{Username: username, Password: opts.Token}
	}

	refs, err := remote.ListContext(ctx, listOpts)
	if err != nil {
		return "", mapError(err)
	}
	return resolveRef(refs, strings.TrimSpace(opts.Ref))
}

func resolveRef(refs []*plumbing.Reference, ref string) (string, error) {
	hashes := make(map[plumbing.ReferenceName]plumbing.Hash, len(refs))
	var head *plumbing.Reference
	for _, r := range refs {
		hashes[r.Name()] = r.Hash()
		if r.Name() == plumbing.HEAD {
			head = r
		}
	}

	if ref == "" {
		if head == nil {
			return "", ErrRef
		}
		if head.Type() == plumbing.SymbolicReference {
			if hash, ok := hashes[head.Target()]; ok && !hash.IsZero() {
				return hash.String(), nil
			}
			return "", ErrRef
		}
		if !head.Hash().IsZero() {
			return head.Hash().String(), nil
		}
		return "", ErrRef
	}

	for _, name := range []plumbing.ReferenceName{
		plumbing.NewBranchReferenceName(ref),
		plumbing.NewTagReferenceName(ref),
	} {
		if hash, ok := hashes[name]; ok && !hash.IsZero() {
			return hash.String(), nil
		}
	}
	return "", ErrRef
}

func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("repository url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid repository url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("repository url must start with http or https")
	}
	if parsed.Host == "" {
		return "", errors.New("repository url is missing a host")
	}
	return raw, nil
}

func ComposePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("compose path is required")
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, `\`) {
		return "", errors.New("compose path must be relative to the repository root")
	}
	clean := filepath.Clean(filepath.FromSlash(raw))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("compose path must stay inside the repository")
	}
	return filepath.ToSlash(clean), nil
}
