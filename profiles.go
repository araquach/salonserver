package salonserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gomarkdown/markdown"
	"gopkg.in/yaml.v2"
)

type ProfileFrontMatter struct {
	Title           string `yaml:"title"`
	Name            string `yaml:"name"`
	InstagramURL    string `yaml:"instagram_url"`
	InstagramHandle string `yaml:"instagram_handle"`
}

type ProfileBlock struct {
	Key           string `json:"key"`
	Image         string `json:"image"`
	Alt           string `json:"alt"`
	ImageClass    string `json:"imageClass"`
	TextClass     string `json:"textClass"`
	TextWidth     int    `json:"textWidth"`
	ImageFirst    bool   `json:"imageFirst"`
	MobileReverse bool   `json:"mobileReverse"`
	HTML          string `json:"html"`
}

type Profile struct {
	Slug            string         `json:"slug"`
	Title           string         `json:"title"`
	Name            string         `json:"name"`
	InstagramURL    string         `json:"instagramUrl"`
	InstagramHandle string         `json:"instagramHandle"`
	Blocks          []ProfileBlock `json:"blocks"`
}

type profileLayoutItem struct {
	Key           string
	ImageIndex    int
	ImageClass    string
	TextClass     string
	TextWidth     int
	ImageFirst    bool
	MobileReverse bool
}

var (
	profileBlockRE = regexp.MustCompile(`(?s)<!--\s*block:([a-zA-Z0-9_-]+)\s*-->`)

	defaultProfileLayout = []profileLayoutItem{
		{
			Key:           "intro",
			ImageIndex:    1,
			ImageClass:    "is-square",
			TextClass:     "is-size-4",
			TextWidth:     7,
			ImageFirst:    false,
			MobileReverse: false,
		},
		{
			Key:           "section_1",
			ImageIndex:    2,
			ImageClass:    "is-square",
			TextClass:     "is-size-5",
			TextWidth:     7,
			ImageFirst:    true,
			MobileReverse: true,
		},
		{
			Key:           "section_2",
			ImageIndex:    3,
			ImageClass:    "is-square",
			TextClass:     "is-size-5",
			TextWidth:     7,
			ImageFirst:    false,
			MobileReverse: false,
		},
		{
			Key:           "section_3",
			ImageIndex:    4,
			ImageClass:    "",
			TextClass:     "is-size-5",
			TextWidth:     7,
			ImageFirst:    true,
			MobileReverse: true,
		},
	}

	profileCache   = map[string]*Profile{}
	profileCacheMu sync.RWMutex
)

func loadProfile(slug string) (*Profile, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("profile slug is required")
	}

	profileCacheMu.RLock()
	if p, ok := profileCache[slug]; ok {
		profileCacheMu.RUnlock()
		return p, nil
	}
	profileCacheMu.RUnlock()

	filePath := filepath.Join("profiles", slug+".md")

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read profile file %q: %w", filePath, err)
	}

	fm, body, err := parseProfileFile(content)
	if err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", slug, err)
	}

	if err := validateProfileFrontMatter(fm); err != nil {
		return nil, err
	}

	bodyBlocks, err := splitProfileBlocks(body)
	if err != nil {
		return nil, fmt.Errorf("split profile blocks for %q: %w", slug, err)
	}

	profile := &Profile{
		Slug:            slug,
		Title:           fm.Title,
		Name:            fm.Name,
		InstagramURL:    fm.InstagramURL,
		InstagramHandle: fm.InstagramHandle,
		Blocks:          buildProfileBlocks(slug, fm.Name, bodyBlocks),
	}

	profileCacheMu.Lock()
	profileCache[slug] = profile
	profileCacheMu.Unlock()

	return profile, nil
}

func loadAllProfiles() ([]*Profile, error) {
	entries, err := os.ReadDir("profiles")
	if err != nil {
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	slugs := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}

		slug := strings.TrimSuffix(name, filepath.Ext(name))
		if strings.TrimSpace(slug) != "" {
			slugs = append(slugs, slug)
		}
	}

	sort.Strings(slugs)

	profiles := make([]*Profile, 0, len(slugs))
	for _, slug := range slugs {
		profile, err := loadProfile(slug)
		if err != nil {
			return nil, fmt.Errorf("load profile %q: %w", slug, err)
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func clearProfileCache() {
	profileCacheMu.Lock()
	defer profileCacheMu.Unlock()
	profileCache = map[string]*Profile{}
}

func profileToJSON(p *Profile) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(p); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func profilesToJSON(profiles []*Profile) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(profiles); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func parseProfileFile(content []byte) (ProfileFrontMatter, string, error) {
	var fm ProfileFrontMatter

	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) < 3 {
		return fm, "", fmt.Errorf("invalid profile markdown format: missing front matter")
	}

	frontMatter := strings.TrimSpace(parts[1])
	body := strings.TrimSpace(parts[2])

	if err := yaml.Unmarshal([]byte(frontMatter), &fm); err != nil {
		return fm, "", fmt.Errorf("unmarshal profile front matter: %w", err)
	}

	return fm, body, nil
}

func validateProfileFrontMatter(fm ProfileFrontMatter) error {
	if strings.TrimSpace(fm.Title) == "" {
		return fmt.Errorf("profile front matter: title is required")
	}
	if strings.TrimSpace(fm.Name) == "" {
		return fmt.Errorf("profile front matter: name is required")
	}
	return nil
}

func splitProfileBlocks(body string) (map[string]string, error) {
	matches := profileBlockRE.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no profile blocks found; expected <!-- block:key --> markers")
	}

	blocks := make(map[string]string, len(matches))

	for i, m := range matches {
		keyStart := m[2]
		keyEnd := m[3]
		key := strings.TrimSpace(body[keyStart:keyEnd])

		contentStart := m[1]
		contentEnd := len(body)
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		}

		content := strings.TrimSpace(body[contentStart:contentEnd])
		if content == "" {
			return nil, fmt.Errorf("profile block %q is empty", key)
		}

		if _, exists := blocks[key]; exists {
			return nil, fmt.Errorf("duplicate profile block %q", key)
		}

		blocks[key] = content
	}

	return blocks, nil
}

func buildProfileBlocks(slug, alt string, bodyBlocks map[string]string) []ProfileBlock {
	blocks := make([]ProfileBlock, 0, len(defaultProfileLayout))

	for _, item := range defaultProfileLayout {
		mdText := strings.TrimSpace(bodyBlocks[item.Key])

		html := ""
		if mdText != "" {
			html = string(markdown.ToHTML([]byte(mdText), nil, nil))
		}

		blocks = append(blocks, ProfileBlock{
			Key:           item.Key,
			Image:         resolveProfileImage(slug, item.ImageIndex),
			Alt:           alt,
			ImageClass:    item.ImageClass,
			TextClass:     item.TextClass,
			TextWidth:     item.TextWidth,
			ImageFirst:    item.ImageFirst,
			MobileReverse: item.MobileReverse,
			HTML:          html,
		})
	}

	return blocks
}

func resolveProfileImage(slug string, index int) string {
	candidates := []string{
		filepath.Join("dist", "img", "team", "profiles", slug, fmt.Sprintf("%s_%d.jpg", slug, index)),
		filepath.Join("dist", "img", "team", "profiles", slug, fmt.Sprintf("%s_%d.jpeg", slug, index)),
		filepath.Join("dist", "img", "team", "profiles", slug, fmt.Sprintf("%s_%d.png", slug, index)),
		filepath.Join("dist", "img", "team", "profiles", slug, fmt.Sprintf("%s_%d.webp", slug, index)),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return "/" + filepath.ToSlash(candidate)
		}
	}

	return "/" + filepath.ToSlash(
		filepath.Join("dist", "img", "team", "profiles", slug, fmt.Sprintf("%s_%d.jpg", slug, index)),
	)
}
