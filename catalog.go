package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
)

const (
	stateIndexing = iota
	stateActive   = iota
	stateDone     = iota
	statePaused   = iota
)

type Movie struct {
	Btih    string // bittorrent info hash used as a key
	Title   string
	Name    string
	RelPath string
	Size    int64
	State   int
	Magnet  string
	//progress chan int
}

type Catalog struct {
	Quota     int64
	SpaceUsed int64
	Movies    []*Movie // TODO: make map with magnet as the key
	path      string
	storage   string
	guard     *sync.Mutex
}

func NewCatalog(catalogPath string, storagePath string, quota int64) *Catalog {
	return &Catalog{Quota: quota, path: catalogPath, storage: storagePath, guard: &sync.Mutex{}}
}

func (c *Catalog) AddMovie(title string, magnet string) (*Movie, error) {
	m := &Movie{Title: title, Magnet: magnet, State: stateIndexing}
	logI.Printf("adding movie with magnet %q to catalog", m.Magnet)

	m.RelPath = path.Join(c.storage, m.Title)
	var err error
	if m.Btih, err = extractBtih(m.Magnet); err != nil {
		return m, fmt.Errorf("extract btih: %v", err)
	}
	dupIndex := 0
	for _, err := os.Stat(m.RelPath); err == nil; {
		logI.Printf("movie with path %q already exists", m.RelPath)
		dupIndex += 1
	}

	m.RelPath += strconv.Itoa(dupIndex)
	c.guard.Lock()
	defer c.guard.Unlock()
	c.Movies = append(c.Movies, m)
	c.saveUnsafe()
	logI.Println("movie added to catalog")

	return m, nil
}

func (c *Catalog) RemoveMovie(btih string) error {
	c.guard.Lock()
	defer c.guard.Unlock()
	pos := c.scanUnsafe(btih)
	if pos == -1 {
		return fmt.Errorf("no movie with btih %q found in catalog", btih)
	}
	mpath := path.Join(c.path, c.Movies[pos].RelPath)
	if err := os.Remove(mpath); err != nil {
		return fmt.Errorf("remove %q from fs: %v", mpath, err)
	}
	c.Movies = append(c.Movies[:pos], c.Movies[pos+1:]...)
	return nil
}

func (c *Catalog) AddMovieInfo(key, name string, size int64) error {
	c.guard.Lock()
	defer c.guard.Unlock()
	pos := c.scanUnsafe(key)
	if pos == -1 {
		return fmt.Errorf("no movie with key %q found", key)
	}
	c.Movies[pos].Name = name
	c.Movies[pos].State = stateActive
	c.Movies[pos].Size = size
	c.SpaceUsed += size
	c.saveUnsafe()
	logI.Println("movie info added to catalog")
	return nil
}

func (c *Catalog) AlreadyHas(key string) bool {
	c.guard.Lock()
	defer c.guard.Unlock()
	return c.scanUnsafe(key) == -1
}

func (c *Catalog) ChangeMovieState(key string, newState int) error {
	c.guard.Lock()
	defer c.guard.Unlock()
	pos := c.scanUnsafe(key)
	if pos == -1 {
		return fmt.Errorf("no movie %q found in catalog", key)
	}
	c.Movies[pos].State = newState
	return nil
}

func (c Catalog) scanUnsafe(key string) int {
	for i, m := range c.Movies {
		logI.Printf("iterate btih=%q", m.Btih)
		if m.Btih == key {
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
