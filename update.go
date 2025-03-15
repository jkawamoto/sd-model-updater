// update.go
//
// Copyright (c) 2023-2025 Junpei Kawamoto
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
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	"github.com/jkawamoto/go-civitai/models"
	"github.com/zeebo/blake3"
)

const pbTemplate = `{{with string . "prefix"}}{{.}} {{end}}{{bar . }} {{percent . }}{{with string . "suffix"}} {{.}}{{end}}`

// fileHash returns the BLAKE3 hash of the given named file.
func fileHash(name string) (_ string, err error) {
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

// Update packs information about new versions for a model.
type Update struct {
	ModelName      string
	CurrentVersion string
	Candidates     map[string]*models.ModelVersion
}

// findUpdate retrieves the model information of the given model file.
func findUpdate(ctx context.Context, cli Client, name string) (*Update, error) {
	hash, err := fileHash(name)
	if err != nil {
		return nil, err
	}

	cur, err := cli.GetModelVersion(ctx, hash)
	if err != nil {
		return nil, err
	}

	m, err := cli.GetModel(ctx, cur.ID)
	if err != nil {
		return nil, err
	}

	res := &Update{
		ModelName:      m.Name,
		CurrentVersion: cur.Name,
		Candidates:     make(map[string]*models.ModelVersion),
	}
	for _, v := range m.ModelVersions {
		if time.Time(v.PublishedAt).After(time.Time(cur.PublishedAt)) {
			res.Candidates[v.Name] = v
		}
	}

	return res, nil
}

// modelVersionList is an alias of []*models.ModelVersion that implements sort.Interface.
type modelVersionList []*models.ModelVersion

func (m modelVersionList) Len() int {
	return len(m)
}

func (m modelVersionList) Less(i, j int) bool {
	return time.Time(m[i].PublishedAt).Before(time.Time(m[j].PublishedAt))
}

func (m modelVersionList) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func findUpdatesFromDir(ctx context.Context, cli Client, dir string) ([]*Update, error) {
	ms := make(map[int64]modelVersionList)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !isModelFile(path) {
			return nil
		}

		hash, err := fileHash(path)
		if err != nil {
			return err
		}

		v, err := cli.GetModelVersion(ctx, hash)
		if err != nil {
			if coder, ok := err.(interface {
				Code() int
			}); ok {
				if coder.Code() == http.StatusNotFound {
					fmt.Println(color.YellowString("Model information is not found"))
					return nil
				}
			}
			return err
		}

		ms[v.ID] = append(ms[v.ID], v)
		return nil
	})
	if err != nil {
		return nil, err
	}

	var res []*Update
	for modelID, versions := range ms {
		model, err := cli.GetModel(ctx, modelID)
		if err != nil {
			return nil, err
		}

		sort.Sort(sort.Reverse(versions))
		cur := versions[0]

		sort.Sort(sort.Reverse(modelVersionList(model.ModelVersions)))
		candidates := make(map[string]*models.ModelVersion)
		for _, v := range model.ModelVersions {
			if time.Time(v.PublishedAt).After(time.Time(cur.PublishedAt)) {
				candidates[v.Name] = v
			}
		}

		if len(candidates) != 0 {
			res = append(res, &Update{
				ModelName:      model.Name,
				CurrentVersion: cur.Name,
				Candidates:     candidates,
			})
		}
	}

	return res, nil
}

func (u Update) run(ctx context.Context, cli Client, dest string) error {
	switch len(u.Candidates) {
	case 0:
		fmt.Println(u.ModelName, "has no updates")
		return nil

	case 1:
		fmt.Println(color.GreenString("%v has a newer version", u.ModelName))
		var ver *models.ModelVersion
		for _, v := range u.Candidates {
			ver = v
		}

		var confirm bool
		err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Do you want to update %v \u279c %v", u.CurrentVersion, ver.Name),
		}, &confirm)
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Println(color.YellowString("Skipped downloading the newer model"))
			return nil
		}

		if err = cli.Download(ctx, ver, dest); err != nil {
			return err
		}

	default:
		fmt.Println(color.GreenString("%v has multiple newer versions", u.ModelName))

		var opts []string
		for n := range u.Candidates {
			opts = append(opts, n)
		}

		var selected []string
		err := survey.AskOne(&survey.MultiSelect{
			Message: fmt.Sprintf("Which versions do you want to download (current: %v)", u.CurrentVersion),
			Options: opts,
		}, &selected)
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			fmt.Println(color.YellowString("Skipped downloading any models"))
			return nil
		}

		for _, n := range selected {
			ver := u.Candidates[n]
			if err = cli.Download(ctx, ver, dest); err != nil {
				return err
			}
		}
	}

	return nil
}
