package main

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

const (
	stateActive = iota
	stateDone   = iota
	statePaused = iota
)

type Movie struct {
	Name   string
	Size   int64
	State  int
	Magnet string
}

type Catalog struct {
	Quota     int64
	SpaceUsed int64
	Movies    []Movie
	path      string
	storage   string
	guard     *sync.Mutex
}

func NewCatalog(catalogPath string, storagePath string, quota int64) *Catalog {
	return &Catalog{Quota: quota, path: catalogPath, storage: storagePath, guard: &sync.Mutex{}}
}

func (c *Catalog) AddMovie(m Movie) {
	c.guard.Lock()
	defer c.guard.Unlock()
	c.Movies = append(c.Movies, m)
	c.save()
}

func (c Catalog) save() error {
	c.guard.Lock()
	defer c.guard.Unlock()
	file, err := os.Create(c.path)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(&c)
	}
	file.Close()
	return err
}

func (c *Catalog) load() error {
	c.guard.Lock()
	defer c.guard.Unlock()
	file, err := os.Open(c.path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(c)
	}
	file.Close()
	return err
}

func syncCatalog(catalogPath string, storagePath string, quota int64) (*Catalog, error) {
	c := NewCatalog(catalogPath, storagePath, quota)

	// Catalog does not exist yet
	if _, err := os.Stat(catalogPath); err != nil {
		files, err := ioutil.ReadDir(storagePath)
		if err != nil {
			return nil, fmt.Errorf("list files: %v", err)
		}

		for _, f := range files {
			var m Movie
			m.Name = f.Name()
			m.Size = f.Size()
			m.State = stateDone

			c.Movies = append(c.Movies, m)
			c.SpaceUsed += m.Size
		}

		if err := c.save(); err != nil {
			return c, fmt.Errorf("save catalog: %v", err)
		}

		return c, nil
	}

	// Catalog file already exists
	if err := c.load(); err != nil {
		return nil, fmt.Errorf("load catalog: %v", err)
	}

	return c, nil
}
