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
	App          config.Application
	Configurator *config.Configurator
	Workspace    string
}

func (s *Source) EnvSource() ([]Searcher, error) {
	source := os.Getenv(config.EnvSource)
	if source == "" {
		return nil, nil
	}
	if !git.IsGitUrl(source) {
		return nil, fmt.Errorf("invalid repository url: %s", source)
	}
	return []Searcher{RepoSearcher(s.App, source, s.App.CommandDirectory)}, nil
}

func (s *Source) ProjectSource() ([]Searcher, error) {
	cmd := filepath.Join(s.Workspace, "."+s.App.Name)
	if d, err := os.Stat(cmd); err == nil && d.IsDir() {
		return []Searcher{FileSearcher(s.App, cmd, s.App.CommandDirectory)}, nil
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
		searchers = append(searchers, RepoSearcher(s.App, group, s.App.CommandDirectory))
	}
	return searchers, nil
}

func (s *Source) ConfigSource() ([]Searcher, error) {
	pc := s.Configurator.LoadConfig()
	if pc.Command.Repo != "" {
		return []Searcher{RepoSearcher(s.App, pc.Command.Repo, s.App.CommandDirectory)}, nil
	}
	if pc.Command.Path != "" {
		return []Searcher{FileSearcher(s.App, pc.Command.Path, "")}, nil
	}
	return nil, nil
}

func (s *Source) RepositorySource(url string) ([]Searcher, error) {
	return []Searcher{RepoSearcher(s.App, url, s.App.CommandDirectory)}, nil
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
			uri.Host, nextGroup, s.App.GroupName)
		repos = append(repos, groupRepo)
	}
	return repos
}
