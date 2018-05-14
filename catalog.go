package main

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

const (
	stateIndexing = iota
	stateActive   = iota
	stateDone     = iota
	statePaused   = iota
)

type Movie struct {
	Title  string
	Name   string
	Size   int64
	State  int
	Magnet string
	//progress chan int
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
	c.saveUnsafe()
}

func (c *Catalog) AddMovieInfo(magnet, name string, size int64) {
	c.guard.Lock()
	defer c.guard.Unlock()
	pos := c.scanUnsafe(magnet)
	c.Movies[pos].Name = name
	c.Movies[pos].State = stateActive
	c.Movies[pos].Size = size
	c.SpaceUsed += size
	c.saveUnsafe()
}

func (c *Catalog) AlreadyHas(magnet string) bool {
	c.guard.Lock()
	defer c.guard.Unlock()
	return c.scanUnsafe(magnet) == -1
}

func (c *Catalog) ChangeMovieState(movieName string, newState int) error {
	c.guard.Lock()
	defer c.guard.Unlock()
	pos := c.scanUnsafe(movieName)
	if pos == -1 {
		return fmt.Errorf("no movie %q found in catalog", movieName)
	}
	c.Movies[pos].State = newState
	return nil
}

func (c Catalog) scanUnsafe(magnet string) int {
	for i, m := range c.Movies {
		if m.Magnet == magnet {
			return i
		}
	}
	return -1
}

func (c Catalog) saveUnsafe() error {
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

		if err := c.saveUnsafe(); err != nil {
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
