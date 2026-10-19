package sshserver

import (
	"errors"
	"strings"
)

type fakeStore struct {
	data    map[string]map[string]any
	keys    []string
	deleted []string
	listed  []string
	failAll bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: map[string]map[string]any{}}
}

func (f *fakeStore) Read(path string) (map[string]any, error) {
	if f.failAll {
		return nil, errors.New("vault down")
	}
	d, ok := f.data[path]
	if !ok {
		return nil, errors.New("not found: " + path)
	}
	return d, nil
}

func (f *fakeStore) Write(path string, data map[string]any) error {
	if f.failAll {
		return errors.New("vault down")
	}
	f.data[path] = data
	return nil
}

func (f *fakeStore) List(path string) ([]string, error) {
	f.listed = append(f.listed, path)
	if f.failAll {
		return nil, errors.New("vault down")
	}
	return f.keys, nil
}

func (f *fakeStore) Delete(path string) error {
	if f.failAll {
		return errors.New("vault down")
	}
	f.deleted = append(f.deleted, path)
	delete(f.data, strings.Replace(path, "/metadata/", "/data/", 1))
	return nil
}
