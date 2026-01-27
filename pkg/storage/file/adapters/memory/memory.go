package memory

import (
	"bytes"
	"context"
	"io"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/file"
)

// node represents a file or directory in the in-memory file system.
type node struct {
	isDir       bool
	data        []byte
	modTime     time.Time
	mode        uint32
	contentType string
	children    map[string]*node // only for directories
}

// Store implements an in-memory file store for testing.
type Store struct {
	mu   sync.RWMutex
	root *node
}

// New creates a new in-memory file store.
func New() *Store {
	return &Store{
		root: &node{
			isDir:    true,
			modTime:  time.Now(),
			mode:     0755,
			children: make(map[string]*node),
		},
	}
}

// NewWithConfig creates a new in-memory file store with config (config is ignored for memory).
func NewWithConfig(_ file.Config) *Store {
	return New()
}

func (s *Store) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n, err := s.getNode(filePath)
	if err != nil {
		return nil, err
	}

	if n.isDir {
		return nil, errors.InvalidArgument("path is a directory", nil)
	}

	return io.NopCloser(bytes.NewReader(n.data)), nil
}

func (s *Store) Write(ctx context.Context, filePath string, data io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create parent directories
	dir := path.Dir(filePath)
	if dir != "/" && dir != "." {
		if err := s.mkdirAll(dir); err != nil {
			return err
		}
	}

	// Read all data
	buf, err := io.ReadAll(data)
	if err != nil {
		return errors.Internal("failed to read data", err)
	}

	// Get or create the file
	parent, err := s.getNodeOrCreate(path.Dir(filePath), true)
	if err != nil {
		return err
	}

	name := path.Base(filePath)
	parent.children[name] = &node{
		isDir:   false,
		data:    buf,
		modTime: time.Now(),
		mode:    0644,
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := path.Dir(filePath)
	name := path.Base(filePath)

	parent, err := s.getNode(dir)
	if err != nil {
		return errors.NotFound("file not found", nil)
	}

	if _, exists := parent.children[name]; !exists {
		return errors.NotFound("file not found", nil)
	}

	delete(parent.children, name)
	return nil
}

func (s *Store) List(ctx context.Context, prefix string, opts file.ListOptions) ([]file.FileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []file.FileInfo

	// Normalize prefix
	if prefix == "" {
		prefix = "/"
	}

	// Find the starting node
	startNode, err := s.getNode(prefix)
	if err != nil {
		// If prefix doesn't exist, return empty list
		return results, nil
	}

	if !startNode.isDir {
		// If prefix is a file, return just that file
		results = append(results, nodeToFileInfo(prefix, startNode))
		return results, nil
	}

	// List directory contents
	s.listRecursive(prefix, startNode, opts.Recursive, &results)

	// Sort by path
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	// Apply pagination
	if opts.Offset > 0 && opts.Offset < len(results) {
		results = results[opts.Offset:]
	} else if opts.Offset >= len(results) {
		return []file.FileInfo{}, nil
	}

	if opts.Limit > 0 && opts.Limit < len(results) {
		results = results[:opts.Limit]
	}

	return results, nil
}

func (s *Store) listRecursive(basePath string, n *node, recursive bool, results *[]file.FileInfo) {
	for name, child := range n.children {
		childPath := path.Join(basePath, name)
		*results = append(*results, nodeToFileInfo(childPath, child))

		if recursive && child.isDir {
			s.listRecursive(childPath, child, recursive, results)
		}
	}
}

func (s *Store) Stat(ctx context.Context, filePath string) (*file.FileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n, err := s.getNode(filePath)
	if err != nil {
		return nil, err
	}

	info := nodeToFileInfo(filePath, n)
	return &info, nil
}

func (s *Store) Mkdir(ctx context.Context, filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.mkdirAll(filePath)
}

func (s *Store) Rename(ctx context.Context, oldPath, newPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get the node to move
	n, err := s.getNode(oldPath)
	if err != nil {
		return err
	}

	// Create parent directories for new path
	newDir := path.Dir(newPath)
	if err := s.mkdirAll(newDir); err != nil {
		return err
	}

	// Get new parent
	newParent, err := s.getNode(newDir)
	if err != nil {
		return err
	}

	// Get old parent
	oldDir := path.Dir(oldPath)
	oldParent, err := s.getNode(oldDir)
	if err != nil {
		return err
	}

	// Move the node
	newName := path.Base(newPath)
	oldName := path.Base(oldPath)

	newParent.children[newName] = n
	delete(oldParent.children, oldName)

	return nil
}

func (s *Store) Copy(ctx context.Context, srcPath, dstPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get source node
	src, err := s.getNode(srcPath)
	if err != nil {
		return err
	}

	if src.isDir {
		return errors.InvalidArgument("cannot copy directories", nil)
	}

	// Create parent directories for destination
	dstDir := path.Dir(dstPath)
	if err := s.mkdirAll(dstDir); err != nil {
		return err
	}

	// Get destination parent
	dstParent, err := s.getNode(dstDir)
	if err != nil {
		return err
	}

	// Create copy
	dstName := path.Base(dstPath)
	dataCopy := make([]byte, len(src.data))
	copy(dataCopy, src.data)

	dstParent.children[dstName] = &node{
		isDir:       false,
		data:        dataCopy,
		modTime:     time.Now(),
		mode:        src.mode,
		contentType: src.contentType,
	}

	return nil
}

// getNode traverses the tree to find the node at the given path.
func (s *Store) getNode(filePath string) (*node, error) {
	if filePath == "/" || filePath == "" || filePath == "." {
		return s.root, nil
	}

	filePath = strings.TrimPrefix(filePath, "/")
	parts := strings.Split(filePath, "/")

	current := s.root
	for _, part := range parts {
		if part == "" {
			continue
		}
		if !current.isDir {
			return nil, errors.NotFound("path not found", nil)
		}
		child, exists := current.children[part]
		if !exists {
			return nil, errors.NotFound("path not found", nil)
		}
		current = child
	}

	return current, nil
}

// getNodeOrCreate traverses and creates directories as needed.
func (s *Store) getNodeOrCreate(filePath string, createDirs bool) (*node, error) {
	if filePath == "/" || filePath == "" || filePath == "." {
		return s.root, nil
	}

	filePath = strings.TrimPrefix(filePath, "/")
	parts := strings.Split(filePath, "/")

	current := s.root
	for _, part := range parts {
		if part == "" {
			continue
		}
		if !current.isDir {
			return nil, errors.InvalidArgument("path is not a directory", nil)
		}
		child, exists := current.children[part]
		if !exists {
			if !createDirs {
				return nil, errors.NotFound("path not found", nil)
			}
			child = &node{
				isDir:    true,
				modTime:  time.Now(),
				mode:     0755,
				children: make(map[string]*node),
			}
			current.children[part] = child
		}
		current = child
	}

	return current, nil
}

// mkdirAll creates all directories in the path.
func (s *Store) mkdirAll(dirPath string) error {
	_, err := s.getNodeOrCreate(dirPath, true)
	return err
}

func nodeToFileInfo(filePath string, n *node) file.FileInfo {
	size := int64(0)
	if !n.isDir {
		size = int64(len(n.data))
	}
	return file.FileInfo{
		Path:        filePath,
		Name:        path.Base(filePath),
		Size:        size,
		IsDir:       n.isDir,
		ModTime:     n.modTime,
		Mode:        n.mode,
		ContentType: n.contentType,
	}
}
