package kf

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// StoreConfig configures a new Store.
type StoreConfig struct {
	BaseDir string
	Locking bool
}

// Store is the root directory in which to keep the key-file store.
type Store struct {
	baseDir string
	locking bool
	locked  bool
}

// NewStore creates a new storage instance.
func NewStore(c *StoreConfig) (*Store, error) {
	if c.BaseDir == "" {
		return nil, errors.New("base directory not specified")
	}
	s := &Store{baseDir: c.BaseDir, locking: c.Locking}

	f, e := os.Open(c.BaseDir)

	// initialize new dir
	// IF the baseDir does not exists, create it (os.ModePerm)
	//   If fail to create, FAIL
	// IF the baseDir does exist and it's a directory, OK
	// IF the baseDir does exist and it's NOT a directory, FAIL
	if e != nil && os.IsNotExist(e) {
		if e := os.MkdirAll(c.BaseDir, os.ModePerm); e != nil {
			log.Fatalln(e)
		}
		return s, nil
	} else if e != nil && os.IsExist(e) {
		log.Fatalln(e)
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("there is a file in the way: %s", c.BaseDir)
	}
	return s, nil
}

// Set saves data.
func (s *Store) Set(value []byte, nkey ...string) (err error) {
	for s.isLocked() {
	}
	if s.locking {
		s.lock()
		defer s.unlock()
	}
	var key string
	if len(key) > 1 {
		key = filepath.Join(nkey...)
	} else if len(key) == 1 {
		key = nkey[0]
	} else {
		return errors.New("no key provided")
	}

	key = s.join(key)

	var p = filepath.Dir(key)

	// if key has a trailing slack (eg. path/to/key/), then intrepret as command to establish dir
	if strings.HasSuffix(key, string(filepath.Separator)) {
		p = key
		if value != nil {
			return fmt.Errorf("cannot use %s as endpoint for value storage: %s; trailing slash for key denotes dir, and a value must be stored in a file", key, string(value))
		}

		return os.MkdirAll(p, os.ModePerm)
	}
	if e := os.MkdirAll(p, os.ModePerm); e != nil {
		return e
	}
	return ioutil.WriteFile(key, value, os.ModePerm)
}

// GetValue gets data from a key store.
func (s *Store) GetValue(key string) (value []byte, err error) {
	for s.isLocked() {
	}
	if s.locking {
		s.lock()
		defer s.unlock()
	}
	key = s.join(key)
	return ioutil.ReadFile(key)
}

// GetKeys gets all keys under a key store.
func (s *Store) GetKeys(ppath string) (keys []string, err error) {
	for s.isLocked() {
	}
	if s.locking {
		s.lock()
		defer s.unlock()
	}
	ppath = s.join(ppath)
	if !ExistsDir(ppath) {
		return nil, errors.New("uninitialized ppath (dir)")
	}
	filepath.Walk(ppath, func(p string, f os.FileInfo, e error) error {
		if e != nil {
			err = e
			return err
		}
		if f.IsDir() {
			return nil
		}
		keys = append(keys, f.Name())
		return nil
	})
	// fs, err := ioutil.ReadDir(ppath) // nonrecursive?
	if err != nil {
		return nil, err
	}
	// for _, f := range fs {
	// 	keys = append(keys, fileppathth.Base(f.Name()))
	// }
	return keys, err
}

// Delete deletes data. Equivalently to 'rm -rf', it does not care if it gets a dir path or file path.
func (s *Store) Delete(key string) (err error) {
	for s.isLocked() {
	}
	if s.locking {
		s.lock()
		defer s.unlock()
	}
	key = s.join(key)
	return os.RemoveAll(key)
}

// BaseDir returns the base directory for storage..
func (s *Store) BaseDir() string {
	return s.baseDir
}

func (s *Store) lock() {
	s.locked = true
	os.Create(s.join(".LOCKDONOTFUCKWITHMEORUSEMEASAKEY"))
}

func (s *Store) unlock() {
	s.locked = false
	os.Remove(s.join(".LOCKDONOTFUCKWITHMEORUSEMEASAKEY"))
}

func (s *Store) isLocked() bool {
	return s.locking && (s.locked || ExistsFile(s.join(".LOCKDONOTFUCKWITHMEORUSEMEASAKEY")))
}

func (s *Store) join(key string) (fullpath string) {
	return filepath.Join(s.baseDir, filepath.Clean(key))
}

// ExistsDir returns whether dpath exists as a directory.
func ExistsDir(dpath string) bool {
	f, e := os.Open(dpath)
	if e != nil && os.IsNotExist(e) {
		return false
	}
	stat, e := f.Stat()
	return e == nil && stat.IsDir()
}

// ExistsFile returns whether fpath exists as a file.
func ExistsFile(fpath string) bool {
	f, e := os.Open(fpath)
	if e != nil && os.IsNotExist(e) {
		return false
	}
	stat, e := f.Stat()
	return e == nil && !stat.IsDir()
}
