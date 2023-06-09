// client.go
//
// Copyright (c) 2023 Junpei Kawamoto
//
// This software is released under the MIT License.
//
// http://opensource.org/licenses/mit-license.php

package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/jkawamoto/go-civitai/client"
	"github.com/jkawamoto/go-civitai/client/operations"
	"github.com/jkawamoto/go-civitai/models"
	"github.com/zeebo/blake3"
	"golang.org/x/net/context/ctxhttp"
)

var (
	ErrFileNotFound     = errors.New("model files are not found in this version")
	ErrFileHashNotMatch = errors.New("file hash doesn't match")
	ErrGetFailure       = errors.New("failed to get a file")
	ErrNoFilename       = errors.New("failed to get a filename")
)

type Client struct {
	clientService operations.ClientService
	httpClient    *http.Client

	PreferredFormat string
}

func NewClient(preferredFormat string) Client {
	return Client{
		clientService:   client.Default.Operations,
		PreferredFormat: preferredFormat,
	}
}

func (cli Client) GetModelVersion(ctx context.Context, hash string) (*models.ModelVersion, error) {
	res, err := cli.clientService.GetModelVersionByHash(
		operations.NewGetModelVersionByHashParamsWithContext(ctx).WithHTTPClient(cli.httpClient).WithHash(hash))
	if err != nil {
		return nil, err
	}

	return res.GetPayload(), nil
}

func (cli Client) GetModel(ctx context.Context, id int64) (*models.Model, error) {
	res, err := cli.clientService.GetModel(
		operations.NewGetModelParamsWithContext(ctx).WithHTTPClient(cli.httpClient).WithModelID(id))
	if err != nil {
		return nil, err
	}

	return res.GetPayload(), nil
}

// Download gets a model file associated with the given version and stores it into the given directory.
func (cli Client) Download(ctx context.Context, ver *models.ModelVersion, dir string) (err error) {
	var file *models.File
	for _, f := range ver.Files {
		if strings.ToLower(f.Format) == cli.PreferredFormat {
			file = f
		}
		if f.Primary && file == nil {
			file = f
		}
	}
	if file == nil {
		return ErrFileNotFound
	}

	res, err := ctxhttp.Get(ctx, cli.httpClient, file.DownloadURL)
	if err != nil {
		return err
	}
	defer func() {
		if _, e := io.Copy(io.Discard, res.Body); e != nil {
			err = errors.Join(err, e)
		}
		err = errors.Join(err, res.Body.Close())
	}()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %v", ErrGetFailure, res.Status)
	}

	_, params, err := mime.ParseMediaType(res.Header.Get("Content-Disposition"))
	if err != nil {
		return errors.Join(ErrNoFilename, err)
	}
	name := params["filename"]

	bar := pb.New(int(file.SizeKB * 1024))
	bar.Set(pb.SIBytesPrefix, true)
	bar.Set("prefix", filepath.Base(name)+" ")
	bar.Start()
	defer bar.Finish()

	hash := blake3.New()
	dest := filepath.Join(dir, name)
	err = writeFile(dest, io.TeeReader(bar.NewProxyReader(res.Body), hash))
	if err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(file.Hashes.BLAKE3) {
		// if hash doesn't match, remove the downloaded file.
		return errors.Join(ErrFileHashNotMatch, os.Remove(dest))
	}
	return nil
}

func writeFile(name string, r io.Reader) (err error) {
	if _, err = os.Stat(name); err == nil {
		return fmt.Errorf("%v already exists: %w", name, os.ErrExist)
	}

	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	_, err = io.Copy(f, r)
	return err
}
