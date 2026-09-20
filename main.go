package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/xuri/excelize/v2"
)

var instagramURLPattern = regexp.MustCompile(`(?i)https?://(?:www\.|m\.)?instagram\.com/([A-Za-z0-9._]+)(?:/)?(?:\?.*)?$`)

func main() {
	installBrowserInterruptCleanup()

	workbookPath := filepath.Join(".", "profiles.xlsx")
	profileLinks, err := extractInstagramProfileLinks(workbookPath)
	if err != nil {
		Errorf("Unable to read Instagram profile links from %s: %v", workbookPath, err)
		os.Exit(1)
	}
	if len(profileLinks) == 0 {
		Warnf("No Instagram profile links were found in %s", workbookPath)
		return
	}

	Infof("Found %d Instagram profile link(s) in %s", len(profileLinks), workbookPath)

	browser, page := openBrowser()
	defer closeActiveBrowser()

	for index, profileURL := range profileLinks {
		Infof("Visiting profile %d/%d: %s", index+1, len(profileLinks), profileURL)
		body, ok := scrapeInstagramProfile(page, profileURL)
		if !ok {
			Warnf("Skipping %s because the profile data could not be captured", profileURL)
			if index < len(profileLinks)-1 {
				pauseBetweenProfileVisits()
			}
			continue
		}
		if _, err := parseInstagramProfileResponse(body); err != nil {
			Warnf("Profile data for %s was captured but did not parse: %v", profileURL, err)
			if index < len(profileLinks)-1 {
				pauseBetweenProfileVisits()
			}
			continue
		}
		Infof("Profile %s processed successfully", profileURL)
		if index < len(profileLinks)-1 {
			pauseBetweenProfileVisits()
		}
	}

	_ = browser
	_ = page
}

func extractInstagramProfileLinks(workbookPath string) ([]string, error) {
	info, err := os.Stat(workbookPath)
	if err != nil {
		return nil, err
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("empty workbook: %s", workbookPath)
	}

	file, err := excelize.OpenFile(workbookPath)
	if err != nil {
		if strings.Contains(err.Error(), "zip: not a valid zip file") || strings.Contains(err.Error(), "EOF") {
			return nil, fmt.Errorf("empty workbook: %s", workbookPath)
		}
		return nil, err
	}
	defer file.Close()

	seen := make(map[string]struct{})
	links := make([]string, 0)
	for _, sheetName := range file.GetSheetList() {
		rows, err := file.GetRows(sheetName)
		if err != nil {
			return nil, fmt.Errorf("read sheet %q: %w", sheetName, err)
		}
		for _, row := range rows {
			for _, cellValue := range row {
				link := normalizeInstagramProfileLink(cellValue)
				if link == "" {
					continue
				}
				if _, exists := seen[link]; exists {
					continue
				}
				seen[link] = struct{}{}
				links = append(links, link)
			}
		}
	}
	return links, nil
}

func normalizeInstagramProfileLink(value string) string {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return ""
	}

	if !strings.Contains(strings.ToLower(candidate), "instagram.com") {
		return ""
	}

	parsed, err := parseInstagramLink(candidate)
	if err == nil {
		return parsed
	}

	schemaless := strings.TrimSpace(candidate)
	if strings.Contains(strings.ToLower(schemaless), "instagram.com") {
		rate := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(schemaless), "https://"), "http://")
		if idx := strings.Index(rate, "instagram.com/"); idx >= 0 {
			path := rate[idx+len("instagram.com/"):]
			path = strings.Trim(path, "/")
			if path == "" {
				return ""
			}
			path = strings.SplitN(path, "/", 2)[0]
			return "https://www.instagram.com/" + path
		}
	}
	return ""
}

func parseInstagramLink(raw string) (string, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return "", fmt.Errorf("empty Instagram link")
	}

	if !strings.Contains(strings.ToLower(candidate), "instagram.com") {
		return "", fmt.Errorf("not an Instagram link: %q", raw)
	}

	trimmed := strings.TrimRight(candidate, "/")
	if matches := instagramURLPattern.FindStringSubmatch(trimmed); len(matches) >= 2 {
		return "https://www.instagram.com/" + matches[1], nil
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := instagramUsernameFromURL(trimmed)
	if err != nil {
		return "", err
	}
	return "https://www.instagram.com/" + parsed, nil
}

func scrapeInstagramProfile(page *rod.Page, profileURL string) ([]byte, bool) {
	router, responses := startInstagramReplayInterceptor(page)
	defer router.MustStop()

	if err := page.Navigate(profileURL); err != nil {
		Warnf("Unable to navigate to %s: %v", profileURL, err)
		return nil, false
	}
	if err := page.Timeout(instagramResponseWaitTimeout).WaitLoad(); err != nil {
		Warnf("Waiting for %s to finish loading failed: %v", profileURL, err)
	}
	if err := page.Timeout(instagramResponseWaitTimeout).WaitStable(2 * time.Second); err != nil {
		Warnf("Waiting for %s to become stable failed: %v", profileURL, err)
	}

	body, ok := waitForInstagramProfileResponse(responses, filepath.Base(profileURL), profileURL)
	if !ok {
		return nil, false
	}
	return body, true
}

func waitForProfilePage(page *rod.Page, url string) error {
	if err := page.Navigate(url); err != nil {
		return err
	}
	return page.WaitLoad()
}

func pauseBetweenProfileVisits() {
	delay := time.Duration(rand.Intn(4500)+1500) * time.Millisecond
	Infof("Waiting %.1f seconds before the next profile visit", delay.Seconds())
	time.Sleep(delay)
}

func init() {
	_ = waitForProfilePage
	_ = time.Now
	rand.Seed(time.Now().UnixNano())
}
