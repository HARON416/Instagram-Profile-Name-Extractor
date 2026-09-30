package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExtractInstagramProfileLinksFromWorkbook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.xlsx")

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Name")
	f.SetCellValue(sheet, "B1", "Instagram")
	f.SetCellValue(sheet, "A2", "Alpha")
	f.SetCellValue(sheet, "B2", "https://www.instagram.com/alpha")
	f.SetCellValue(sheet, "A3", "Beta")
	f.SetCellValue(sheet, "B3", "https://instagram.com/beta/")
	f.SetCellValue(sheet, "A4", "Gamma")
	f.SetCellValue(sheet, "B4", "https://example.com/not-instagram")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save workbook: %v", err)
	}

	got, err := extractInstagramProfileLinks(path)
	if err != nil {
		t.Fatalf("extract links: %v", err)
	}

	want := []string{
		"https://www.instagram.com/alpha",
		"https://www.instagram.com/beta",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("links mismatch\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestExtractInstagramProfileLinksRejectsEmptyWorkbook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.xlsx")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatalf("write empty workbook: %v", err)
	}

	_, err := extractInstagramProfileLinks(path)
	if err == nil {
		t.Fatal("expected an error for an empty workbook")
	}
	if got, want := err.Error(), "empty workbook"; !strings.Contains(got, want) {
		t.Fatalf("error = %q, want it to mention %q", got, want)
	}
}

func TestUpdateInstagramProfileNamesInWorkbook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.xlsx")
	f := excelize.NewFile()
	defer f.Close()

	layouts := []struct {
		sheet          string
		usernameHeader string
		nameColumn     string
	}{
		{"Sheet1", "", "B"},
		{"WithUsername", "USERNAME", "C"},
		{"EmptyUsername", "USERNAME", "C"},
		{"NormalizedHeader", "  UserName  ", "C"},
	}
	for i, layout := range layouts {
		if i > 0 {
			if _, err := f.NewSheet(layout.sheet); err != nil {
				t.Fatal(err)
			}
		}
		cells := map[string]string{
			"A1": "PROFILE",
			"A2": "https://www.instagram.com/alpha",
			"A3": "https://www.instagram.com/beta",
		}
		if layout.usernameHeader != "" {
			cells["B1"] = layout.usernameHeader
			cells["B2"] = "alpha"
			if layout.sheet == "EmptyUsername" {
				cells["B2"] = ""
				cells["C1"] = "NAME"
			}
		}
		for cell, value := range cells {
			if err := f.SetCellValue(layout.sheet, cell, value); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save workbook: %v", err)
	}

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	previousResults := make(map[string][]byte)
	// Repeated runs create separate results and never change the input or earlier results.
	for _, name := range []string{"Alpha Smith", "Alpha Updated"} {
		if err := updateInstagramProfileNamesInWorkbook(path, map[string]string{
			"https://www.instagram.com/alpha": name,
		}); err != nil {
			t.Fatalf("update workbook names: %v", err)
		}
		input, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(input, original) {
			t.Fatalf("source workbook changed: %v", err)
		}
		outputs, err := filepath.Glob(filepath.Join(filepath.Dir(path), "profiles_results-*.xlsx"))
		if err != nil || len(outputs) != len(previousResults)+1 {
			t.Fatalf("expected one new results file, got %v: %v", outputs, err)
		}
		var outputPath string
		for _, candidate := range outputs {
			data, err := os.ReadFile(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if previous, exists := previousResults[candidate]; exists {
				if !bytes.Equal(data, previous) {
					t.Fatal("previous results were changed")
				}
			} else {
				outputPath = candidate
				previousResults[candidate] = data
			}
		}
		updated, err := excelize.OpenFile(outputPath)
		if err != nil {
			t.Fatalf("open updated workbook: %v", err)
		}
		for _, layout := range layouts {
			want := map[string]string{
				"A1":                    "PROFILE",
				"A2":                    "https://www.instagram.com/alpha",
				"A3":                    "https://www.instagram.com/beta",
				layout.nameColumn + "1": "NAME",
				layout.nameColumn + "2": name,
				layout.nameColumn + "3": "",
			}
			if layout.usernameHeader != "" {
				want["B1"] = layout.usernameHeader
				want["B2"] = "alpha"
				if layout.sheet == "EmptyUsername" {
					want["B2"] = ""
				}
			} else {
				want["C1"] = ""
				want["C2"] = ""
			}
			for cell, expected := range want {
				if got, err := updated.GetCellValue(layout.sheet, cell); err != nil || got != expected {
					t.Errorf("%s!%s = %q, want %q, err = %v", layout.sheet, cell, got, expected, err)
				}
			}
		}
		updated.Close()
	}
}

func TestSaveResultsWorkbookReportsCreationFailure(t *testing.T) {
	file := excelize.NewFile()
	defer file.Close()
	path, err := saveResultsWorkbook(file, filepath.Join(t.TempDir(), "missing"))
	if err == nil || path != "" || !strings.Contains(err.Error(), "create results workbook") {
		t.Fatalf("path = %q, err = %v", path, err)
	}
}
