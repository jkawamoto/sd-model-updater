// client_test.go
//
// Copyright (c) 2023 Junpei Kawamoto
//
// This software is released under the MIT License.
//
// http://opensource.org/licenses/mit-license.php

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/jkawamoto/go-civitai/models"
)

func sha256hash(t *testing.T, name string) string {
	t.Helper()

	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	hash := sha256.New()
	if _, err = io.Copy(hash, f); err != nil {
		t.Fatal(err)
	}

	return hex.EncodeToString(hash.Sum(nil))
}

func joinURL(t *testing.T, base string, path ...string) string {
	t.Helper()
	res, err := url.JoinPath(base, path...)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestNewClient(t *testing.T) {
	format := "test"

	c := NewClient(format)
	if c.clientService == nil {
		t.Error("expect not nil")
	}
	if c.PreferredFormat != format {
		t.Errorf("expect %v, got %v", format, c.PreferredFormat)
	}
}

func TestClient_Download(t *testing.T) {
	ctx := context.Background()
	target := "LICENSE"
	hash := sha256hash(t, target)

	mux := http.NewServeMux()
	mux.HandleFunc("/"+target, func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v;", target))
		res.WriteHeader(http.StatusOK)

		f, err := os.Open(target)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}()

		if _, err = io.Copy(res, f); err != nil {
			t.Fatal()
		}
	})
	mux.HandleFunc("/no-content-disposition", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})

	cases := []struct {
		name            string
		preferredFormat string
		ver             *models.ModelVersion
		err             error
	}{
		{
			name:            "prefer: safetensor, exists",
			preferredFormat: SafetensorFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, "not_found"),
						Format:      "Pickle",
						Primary:     true,
					},
					{
						DownloadURL: joinURL(t, server.URL, target),
						Format:      "SafeTensor",
						Hashes: &models.Hash{
							SHA256: hash,
						},
					},
				},
			},
		},
		{
			name:            "prefer: safetensor, not exists, primary: pickle",
			preferredFormat: SafetensorFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, target),
						Format:      "Pickle",
						Primary:     true,
						Hashes: &models.Hash{
							SHA256: hash,
						},
					},
				},
			},
		},
		{
			name:            "prefer: safetensor, not exists, primary: none",
			preferredFormat: SafetensorFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, target),
						Format:      "Pickle",
					},
				},
			},
			err: ErrFileNotFound,
		},
		{
			name:            "prefer: pickle, exists",
			preferredFormat: PickleFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, "not_found"),
						Format:      "SafeTensor",
						Primary:     true,
					},
					{
						DownloadURL: joinURL(t, server.URL, target),
						Format:      "Pickle",
						Hashes: &models.Hash{
							SHA256: hash,
						},
					},
				},
			},
		},
		{
			name:            "prefer: pickle, not exists, primary: safetensor",
			preferredFormat: PickleFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, target),
						Format:      "SafeTensor",
						Primary:     true,
						Hashes: &models.Hash{
							SHA256: hash,
						},
					},
				},
			},
		},
		{
			name:            "prefer: pickle, not exists, primary: none",
			preferredFormat: PickleFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, target),
						Format:      "SafeTensor",
					},
				},
			},
			err: ErrFileNotFound,
		},
		{
			name:            "404 error",
			preferredFormat: SafetensorFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, "another_file"),
						Format:      "SafeTensor",
					},
				},
			},
			err: ErrGetFailure,
		},
		{
			name:            "no content disposition",
			preferredFormat: SafetensorFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, "no-content-disposition"),
						Format:      "SafeTensor",
					},
				},
			},
			err: ErrNoFilename,
		},
		{
			name:            "hash not match",
			preferredFormat: PickleFormat,
			ver: &models.ModelVersion{
				Files: []*models.File{
					{
						DownloadURL: joinURL(t, server.URL, target),
						Format:      "SafeTensor",
						Primary:     true,
						Hashes: &models.Hash{
							SHA256: "hash",
						},
					},
				},
			},
			err: ErrFileHashNotMatch,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()

			cli := NewClient(c.preferredFormat)
			cli.httpClient = server.Client()

			err := cli.Download(ctx, c.ver, dir)
			if (c.err == nil && err != nil) || (c.err != nil && !errors.Is(err, c.err)) {
				t.Errorf("expect %v, got %v", c.err, err)
			}

			if c.err == nil {
				h := sha256hash(t, filepath.Join(dir, target))
				if h != hash {
					t.Errorf("expect %v, got %v", hash, h)
				}
			}
		})
	}
}
