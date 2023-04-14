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
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
	"github.com/jkawamoto/go-civitai/models"
)

const (
	SafetensorFormat = "safetensor"
	PickleFormat     = "pickle"
)

// ErrUnknownFormat returns if the given format is neither safetensor nor pickle.
var ErrUnknownFormat = fmt.Errorf("unknown format is specified")

var defaultTargets = []string{
	filepath.Join("models", "hypernetworks"),
	filepath.Join("models", "Lora"),
	filepath.Join("models", "Stable-diffusion"),
	filepath.Join("models", "VAE"),
	"embeddings",
}

var modelFileExtensions = []string{".safetensors", ".ckpt", ".pt"}

func isModelFile(name string) bool {
	ext := filepath.Ext(name)
	for _, e := range modelFileExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

func update(ctx context.Context, cli Client, name string) error {
	model, err := GetModel(ctx, cli, name)
	if err != nil {
		if coder, ok := err.(interface {
			Code() int
		}); ok {
			if coder.Code() == http.StatusNotFound {
				fmt.Println(color.YellowString("Model information is not found"))
				fmt.Println()
				return nil
			}
		}
		return err
	}

	switch len(model.Candidates) {
	case 0:
		// no updated models.
		fmt.Println("No updates are found")
		fmt.Println()
		return nil

	case 1:
		fmt.Println(color.GreenString("Newer version is found"))
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
			fmt.Println(color.YellowString("Skipped downloading the newer model"))
			fmt.Println()
			return nil
		}

		if err = cli.Download(ctx, ver, filepath.Dir(name)); err != nil {
			return err
		}

	default:
		fmt.Println(color.GreenString("Multiple newer versions are found"))

		var opts []string
		for n := range model.Candidates {
			opts = append(opts, n)
		}

		var selected []string
		err = survey.AskOne(&survey.MultiSelect{
			Message: "Which versions do you want to download",
			Options: opts,
		}, &selected)
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			fmt.Println(color.YellowString("Skipped downloading any models"))
			fmt.Println()
			return nil
		}

		for _, n := range selected {
			ver := model.Candidates[n]
			if err = cli.Download(ctx, ver, filepath.Dir(name)); err != nil {
				return err
			}
		}
	}

	var confirm bool
	err = survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Do you want to remove the old version: %v", filepath.Base(name)),
	}, &confirm)
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}

	fmt.Println()
	return os.Remove(name)
}

func run(ctx context.Context) error {
	preferredFormat := SafetensorFormat
	flag.Func(
		"format",
		fmt.Sprintf("prefered file format: %v or %v (default %v)", SafetensorFormat, PickleFormat, SafetensorFormat),
		func(s string) error {
			if s != SafetensorFormat && s != PickleFormat {
				return ErrUnknownFormat
			}
			preferredFormat = s
			return nil
		},
	)

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

	cli := NewClient(preferredFormat)
	for _, name := range targets {
		stat, err := os.Stat(name)
		if err != nil {
			return err
		}

		if !stat.IsDir() {
			err = update(ctx, cli, name)
			if err != nil {
				if errors.Is(err, terminal.InterruptErr) {
					return err
				}
				fmt.Printf(color.RedString("Failed to update %v: %v\n"), filepath.Base(name), err)
			}
		} else {
			fmt.Println("\u279c", name)
			err = filepath.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if !isModelFile(path) {
					return nil
				}

				err = update(ctx, cli, path)
				if err != nil {
					if errors.Is(err, terminal.InterruptErr) {
						return err
					}
					fmt.Printf(color.RedString("Failed to update %v: %v\n"), filepath.Base(name), err)
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
		fmt.Println(color.RedString("Failed to check for updates: %v", err))
		os.Exit(1)
	}
}
