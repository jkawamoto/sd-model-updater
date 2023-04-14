// model.go
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
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/jkawamoto/go-civitai/models"
	"github.com/zeebo/blake3"
)

const pbTemplate = `{{with string . "prefix"}}{{.}} {{end}}{{bar . }} {{percent . }}{{with string . "suffix"}} {{.}}{{end}}`

// FileHash returns the SHA256 hash of the given named file.
func FileHash(name string) (_ string, err error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	bar := pb.New(int(info.Size()))
	bar.SetTemplate(pbTemplate)
	bar.Set(pb.SIBytesPrefix, true)
	bar.Set("prefix", filepath.Base(name)+" ")
	bar.Start()
	defer bar.Finish()

	hash := blake3.New()
	_, err = io.Copy(hash, bar.NewProxyReader(f))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

type Model struct {
	ModelName      string
	CurrentVersion string
	Candidates     map[string]*models.ModelVersion
}

// GetModel retrieves the model information of the given model file.
func GetModel(ctx context.Context, cli Client, name string) (*Model, error) {
	hash, err := FileHash(name)
	if err != nil {
		return nil, err
	}

	cur, err := cli.GetModelVersion(ctx, hash)
	if err != nil {
		return nil, err
	}

	m, err := cli.GetModel(ctx, cur.ModelID)
	if err != nil {
		return nil, err
	}

	res := &Model{
		ModelName:      m.Name,
		CurrentVersion: cur.Name,
		Candidates:     make(map[string]*models.ModelVersion),
	}
	for _, v := range m.ModelVersions {
		if time.Time(v.CreatedAt).After(time.Time(cur.CreatedAt)) {
			res.Candidates[v.Name] = v
		}
	}

	return res, nil
}
