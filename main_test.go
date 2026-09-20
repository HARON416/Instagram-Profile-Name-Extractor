package main

import (
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
