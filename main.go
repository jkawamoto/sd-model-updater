// main.go
//
// Copyright (c) 2023 Junpei Kawamoto
//
// This software is released under the MIT License.
//
// http://opensource.org/licenses/mit-license.php

package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/jkawamoto/go-civitai/models"
)

var defaultTargets = []string{
	filepath.Join("models", "hypernetworks"),
	filepath.Join("models", "Lora"),
	filepath.Join("models", "Stable-diffusion"),
	filepath.Join("models", "VAE"),
	"embeddings",
}

func update(ctx context.Context, cli Client, name string) error {
	fmt.Printf("Checking for updates of %v\n", filepath.Base(name))
	model, err := GetModel(ctx, cli, name)
	if err != nil {
		if coder, ok := err.(interface {
			Code() int
		}); ok {
			if coder.Code() == http.StatusNotFound {
				fmt.Println("Model information is not found")
				return nil
			}
		}
		return err
	}

	switch len(model.Candidates) {
	case 0:
		// no updated models.
		fmt.Println("No updates are found")
		return nil

	case 1:
		fmt.Println("One new version is found")
		var ver *models.ModelVersion
		for _, v := range model.Candidates {
			ver = v
		}

		var confirm bool
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Do you want to update %v \u279c %v", model.CurrentVersion, ver.Name),
		}, &confirm)
		if err != nil {
			return err
		}
		if !confirm {
			return nil
		}

		fmt.Printf("Downloading %v from %v\n", ver.Name, ver.DownloadURL)
		if err = Download(ctx, ver.DownloadURL, filepath.Dir(name)); err != nil {
			return err
		}
		fmt.Println("Done")

	default:
		fmt.Println("New versions are found")

		var opts []string
		for n := range model.Candidates {
			opts = append(opts, n)
		}

		var selected []string
		err = survey.AskOne(&survey.MultiSelect{
			Message: "Which models do you want to download",
			Options: opts,
		}, &selected)
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			fmt.Println("Skip downloading any models")
			return nil
		}

		for _, n := range selected {
			ver := model.Candidates[n]
			fmt.Printf("Downloading %v from %v\n", n, ver.DownloadURL)
			if err = Download(ctx, ver.DownloadURL, filepath.Dir(name)); err != nil {
				return err
			}
			fmt.Println("Done")
		}
	}

	var confirm bool
	err = survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Do you want to remove the old version: %v", name),
	}, &confirm)
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}

	return os.Remove(name)
}

func run(ctx context.Context) error {
	flag.Parse()

	targets := flag.Args()
	if len(targets) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		for _, t := range defaultTargets {
			targets = append(targets, filepath.Join(wd, t))
		}
	}

	cli := NewClient()
	for _, name := range targets {
		stat, err := os.Stat(name)
		if err != nil {
			return err
		}

		if !stat.IsDir() {
			err = update(ctx, cli, name)
			if err != nil {
				fmt.Printf("Failed to update %v: %v\n", name, err)
			}
		} else {
			fmt.Println("Retrieving models in", name)
			err = filepath.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}

				ext := filepath.Ext(path)
				if ext != ".safetensors" && ext != ".ckpt" && ext != ".pt" {
					return nil
				}

				err = update(ctx, cli, path)
				if err != nil {
					fmt.Printf("Failed to update %v: %v\n", path, err)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
