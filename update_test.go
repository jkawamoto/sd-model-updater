// update_test.go
//
// Copyright (c) 2023 Junpei Kawamoto
//
// This software is released under the MIT License.
//
// http://opensource.org/licenses/mit-license.php

package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/zeebo/blake3"
)

func Test_fileHash(t *testing.T) {
	target := "README.md"

	f, err := os.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	h := blake3.New()
	if _, err = io.Copy(h, f); err != nil {
		t.Fatal(err)
	}
	expect := hex.EncodeToString(h.Sum(nil))

	res, err := fileHash(target)
	if err != nil {
		t.Fatal(err)
	}

	if res != expect {
		t.Errorf("expect %v, got %v", expect, res)
	}
}

func Test_modelVersionList(t *testing.T) {
	list := modelVersionList{
		{
			ID:          3,
			PublishedAt: strfmt.DateTime(time.Now().Add(3 * time.Hour)),
		},
		{
			ID:          1,
			PublishedAt: strfmt.DateTime(time.Now().Add(1 * time.Hour)),
		},
		{
			ID:          2,
			PublishedAt: strfmt.DateTime(time.Now().Add(2 * time.Hour)),
		},
	}

	t.Run("Len", func(t *testing.T) {
		res := list.Len()
		if res != len(list) {
			t.Errorf("expect %v, got %v", len(list), res)
		}
	})

	t.Run("Less", func(t *testing.T) {
		cases := []struct {
			i      int
			j      int
			expect bool
		}{
			{i: 0, j: 0, expect: false},
			{i: 0, j: 1, expect: false},
			{i: 0, j: 2, expect: false},
			{i: 1, j: 0, expect: true},
			{i: 1, j: 1, expect: false},
			{i: 1, j: 2, expect: true},
			{i: 2, j: 0, expect: true},
			{i: 2, j: 1, expect: false},
			{i: 2, j: 2, expect: false},
		}
		for _, c := range cases {
			t.Run(fmt.Sprintf("i:%v, j:%v", c.i, c.j), func(t *testing.T) {
				if res := list.Less(c.i, c.j); res != c.expect {
					t.Errorf("expect %v, got %v", c.expect, res)
				}
			})
		}
	})

	t.Run("Swap", func(t *testing.T) {
		l := make(modelVersionList, len(list))
		for i, v := range list {
			l[i] = v
		}

		l.Swap(0, 2)
		if l[0].ID != list[2].ID || l[1].ID != list[1].ID || l[2].ID != list[0].ID {
			t.Error("swapped list doesn't match")
		}
	})
}
