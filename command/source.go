package command

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/tools/git"
)

type SourceLoader func() ([]Searcher, error)

type Source struct {
	Workspace        string
	Name             string
	GroupName        string
	CommandDirectory string
	Configurator     *config.Configurator
}

func (s *Source) EnvSource() ([]Searcher, error) {
	source := os.Getenv(config.EnvSource)
	if source == "" {
		return nil, nil
	}
	if !git.IsGitUrl(source) {
		return nil, fmt.Errorf("invalid repository url: %s", source)
	}
	return []Searcher{RepoSearcher(source, s.CommandDirectory)}, nil
}

func (s *Source) ProjectSource() ([]Searcher, error) {
	cmd := filepath.Join(s.Workspace, "."+s.Name)
	if d, err := os.Stat(cmd); err == nil && d.IsDir() {
		return []Searcher{FileSearcher(cmd, s.CommandDirectory)}, nil
	}
	return nil, nil
}

func (s *Source) ProjectGroupSource() ([]Searcher, error) {
	gi, err := git.Instance(s.Workspace)
	if err != nil {
		return nil, fmt.Errorf("invalid repository: %w", err)
	}
	if !gi.IsRepository() {
		return nil, nil
	}
	repos, err := gi.GetRemoteUrls()
	if err != nil {
		return nil, fmt.Errorf("read remote urls failed, %w", err)
	}
	var groups []string
	for _, repo := range repos {
		groups = append(groups, s.groupUrls(repo)...)
	}
	searchers := make([]Searcher, 0, len(groups))
	for _, group := range groups {
		searchers = append(searchers, RepoSearcher(group, s.CommandDirectory))
	}
	return searchers, nil
}

func (s *Source) ConfigSource() ([]Searcher, error) {
	pc := s.Configurator.LoadConfig()
	if pc.Command.Repo != "" {
		return []Searcher{RepoSearcher(pc.Command.Repo, s.CommandDirectory)}, nil
	}
	if pc.Command.Path != "" {
		return []Searcher{FileSearcher(pc.Command.Path, "")}, nil
	}
	return nil, nil
}

func (s *Source) RepositorySource(url string) ([]Searcher, error) {
	return []Searcher{RepoSearcher(url, s.CommandDirectory)}, nil
}

func (s *Source) groupUrls(repo string) []string {
	if !git.IsGitUrl(repo) {
		return nil
	}
	repo, _ = git.ToHttp(repo, true)
	uri, err := url.Parse(strings.TrimSuffix(repo, ".git"))
	if err != nil {
		return nil
	}
	groups := strings.Split(strings.Trim(uri.Path, "/"), "/")
	if len(groups) == 0 {
		return nil
	}
	groups = groups[:len(groups)-1]
	repos := make([]string, 0, len(groups))
	nextGroup := ""
	for _, group := range groups {
		nextGroup += group + "/"
		groupRepo := fmt.Sprintf("%s://%s/%s%s.git", uri.Scheme,
			uri.Host, nextGroup, s.GroupName)
		repos = append(repos, groupRepo)
	}
	return repos
}
