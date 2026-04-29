package importer

import (
	"encoding/json"
	"fmt"
	"io"
)

const bitwardenLoginType = 1

type bitwardenImporter struct{}

type bitwardenExport struct {
	Folders []bitwardenFolder `json:"folders"`
	Items   []bitwardenItem   `json:"items"`
}

type bitwardenFolder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type bitwardenItem struct {
	Type     int              `json:"type"`
	Name     string           `json:"name"`
	FolderID string           `json:"folderId"`
	Login    bitwardenLogin   `json:"login"`
	Notes    string           `json:"notes"`
	Fields   []bitwardenField `json:"fields"`
}

type bitwardenLogin struct {
	Username string         `json:"username"`
	Password string         `json:"password"`
	TOTP     string         `json:"totp"`
	URIs     []bitwardenURI `json:"uris"`
}

type bitwardenURI struct {
	URI string `json:"uri"`
}

type bitwardenField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (i *bitwardenImporter) Parse(r io.Reader) ([]ImportedEntry, error) {
	var export bitwardenExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, fmt.Errorf("parse bitwarden export: %w", err)
	}

	folders := make(map[string]string, len(export.Folders))
	for _, folder := range export.Folders {
		if folder.ID == "" {
			continue
		}
		folders[folder.ID] = folder.Name
	}

	entries := make([]ImportedEntry, 0, len(export.Items))
	for _, item := range export.Items {
		if item.Type != bitwardenLoginType {
			continue
		}

		data := map[string]any{
			"username": item.Login.Username,
			"password": item.Login.Password,
			"url":      bitwardenPrimaryURI(item.Login.URIs),
			"urls":     bitwardenURIs(item.Login.URIs),
			"notes":    item.Notes,
		}

		for _, field := range item.Fields {
			if field.Name == "" {
				continue
			}
			data[field.Name] = field.Value
		}

		if item.Login.TOTP != "" {
			data["totp"] = map[string]any{"secret": item.Login.TOTP}
		}

		entries = append(entries, ImportedEntry{
			Path: bitwardenPath(item, folders),
			Data: data,
		})
	}

	return entries, nil
}

func bitwardenPrimaryURI(uris []bitwardenURI) string {
	if len(uris) == 0 {
		return ""
	}
	return uris[0].URI
}

func bitwardenURIs(uris []bitwardenURI) []string {
	result := make([]string, 0, len(uris))
	for _, uri := range uris {
		if uri.URI == "" {
			continue
		}
		result = append(result, uri.URI)
	}
	return result
}

func bitwardenPath(item bitwardenItem, folders map[string]string) string {
	path := item.Name
	if folderName := folders[item.FolderID]; folderName != "" {
		path = ApplyPrefix(folderName, path)
	}
	return NormalizePath(path)
}
