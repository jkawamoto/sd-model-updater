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
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jkawamoto/go-civitai/client"
	"github.com/jkawamoto/go-civitai/client/operations"
	"github.com/jkawamoto/go-civitai/models"
	"golang.org/x/net/context/ctxhttp"
)

type Client struct {
	clientService operations.ClientService
	httpClient    *http.Client
}

func NewClient() Client {
	return Client{
		clientService: client.Default.Operations,
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

func writeFile(name string, r io.Reader) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(f.Close())
	}()

	_, err = io.Copy(f, r)
	return err
}

func (cli Client) Download(ctx context.Context, url, dir string) (err error) {
	res, err := ctxhttp.Get(ctx, cli.httpClient, url)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(res.Body.Close())
	}()

	_, params, err := mime.ParseMediaType(res.Header.Get("Content-Disposition"))
	if err != nil {
		return err
	}

	return writeFile(filepath.Join(dir, params["filename"]), res.Body)
}
