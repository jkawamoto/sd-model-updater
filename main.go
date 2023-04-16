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
	"net/http"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
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

// isModelFile returns true if the given name represents a model file.
func isModelFile(name string) bool {
	ext := filepath.Ext(name)
	for _, e := range modelFileExtensions {
		if ext == e {
			return true
		}
	}
	return false
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
			update, err := findUpdate(ctx, cli, name)
			if err != nil {
				if coder, ok := err.(interface {
					Code() int
				}); ok {
					if coder.Code() == http.StatusNotFound {
						fmt.Println(color.YellowString("Model information is not found"))
						continue
					}
				}
				fmt.Println(color.RedString("Failed to find updates to %v: %v", filepath.Base(name), err))
				continue
			}

			err = update.run(ctx, cli, filepath.Dir(name))
			if err != nil {
				if errors.Is(err, terminal.InterruptErr) {
					return err
				}
				fmt.Println(color.RedString("Failed to update %v: %v", filepath.Base(name), err))
			}
		} else {
			fmt.Println("Retrieving models in", name)

			updates, err := findUpdatesFromDir(ctx, cli, name)
			if err != nil {
				fmt.Println(color.RedString("Failed to find updates to models in %v: %v", name, err))
				continue
			}

			for _, u := range updates {
				err = u.run(ctx, cli, name)
				if err != nil {
					if errors.Is(err, terminal.InterruptErr) {
						return err
					}
					fmt.Println(color.RedString("Failed to update %v: %v", u.ModelName, err))
				}
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
