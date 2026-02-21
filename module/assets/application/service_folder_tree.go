package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mannaiah/module/assets/domain"
)

// GetFolderTree returns all folders as a hierarchical tree.
func (s *AssetService) GetFolderTree(ctx context.Context) ([]domain.Folder, error) {
	flatFolders, err := s.repository.ListAllFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list asset folders for tree: %w", err)
	}

	return buildFolderTree(flatFolders), nil
}

// buildFolderTree assembles parent-child folder relations from flat rows.
func buildFolderTree(folders []domain.Folder) []domain.Folder {
	nodes := make(map[string]domain.Folder, len(folders))
	for _, folder := range folders {
		id := strings.TrimSpace(folder.ID)
		if id == "" {
			continue
		}

		folder.ID = id
		folder.ParentFolderID = strings.TrimSpace(folder.ParentFolderID)
		folder.Children = nil
		nodes[id] = folder
	}
	if len(nodes) == 0 {
		return []domain.Folder{}
	}

	childrenByParent := make(map[string][]string, len(nodes))
	rootIDs := make([]string, 0, len(nodes))

	for id, folder := range nodes {
		parentID := strings.TrimSpace(folder.ParentFolderID)
		if parentID == "" || parentID == id {
			rootIDs = append(rootIDs, id)
			continue
		}
		if _, exists := nodes[parentID]; !exists {
			rootIDs = append(rootIDs, id)
			continue
		}

		childrenByParent[parentID] = append(childrenByParent[parentID], id)
	}

	if len(rootIDs) == 0 {
		for id := range nodes {
			rootIDs = append(rootIDs, id)
		}
	}

	sortFolderIDs(rootIDs, nodes)
	for parentID := range childrenByParent {
		sortFolderIDs(childrenByParent[parentID], nodes)
	}

	tree := make([]domain.Folder, 0, len(rootIDs))
	emitted := make(map[string]struct{}, len(nodes))
	for _, rootID := range rootIDs {
		if _, exists := emitted[rootID]; exists {
			continue
		}

		tree = append(tree, buildFolderNode(rootID, nodes, childrenByParent, map[string]struct{}{}, emitted))
	}

	return tree
}

// buildFolderNode materializes one tree branch while preventing cyclic recursion.
func buildFolderNode(
	folderID string,
	nodes map[string]domain.Folder,
	childrenByParent map[string][]string,
	path map[string]struct{},
	emitted map[string]struct{},
) domain.Folder {
	folder := nodes[folderID]
	if _, exists := path[folderID]; exists {
		folder.Children = []domain.Folder{}
		return folder
	}

	path[folderID] = struct{}{}
	emitted[folderID] = struct{}{}

	childIDs := childrenByParent[folderID]
	folder.Children = make([]domain.Folder, 0, len(childIDs))
	for _, childID := range childIDs {
		if _, inPath := path[childID]; inPath {
			continue
		}

		folder.Children = append(folder.Children, buildFolderNode(childID, nodes, childrenByParent, path, emitted))
	}

	delete(path, folderID)

	return folder
}

// sortFolderIDs sorts folder identifiers by display name, slug, and id values.
func sortFolderIDs(folderIDs []string, nodes map[string]domain.Folder) {
	sort.Slice(folderIDs, func(i int, j int) bool {
		left := nodes[folderIDs[i]]
		right := nodes[folderIDs[j]]

		leftName := strings.ToLower(strings.TrimSpace(left.Name))
		rightName := strings.ToLower(strings.TrimSpace(right.Name))
		if leftName != rightName {
			return leftName < rightName
		}

		leftSlug := strings.ToLower(strings.TrimSpace(left.Slug))
		rightSlug := strings.ToLower(strings.TrimSpace(right.Slug))
		if leftSlug != rightSlug {
			return leftSlug < rightSlug
		}

		return strings.TrimSpace(left.ID) < strings.TrimSpace(right.ID)
	})
}
