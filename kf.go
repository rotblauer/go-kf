package kf

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// Store is the root directory in which to keep the key-file store.
type Store struct {
	baseDir string
}

// NewStore creates a new storage instance.
func NewStore(baseDir string) *Store {
	baseDir = filepath.Clean(baseDir)
	baseDir, e := filepath.Abs(baseDir)
	if e != nil {
		log.Fatalln(e)
	}
	f, e := os.Open(baseDir)

	// initialize new dir
	if e != nil && os.IsNotExist(e) {
		if e := os.MkdirAll(baseDir, os.ModePerm); e != nil {
			log.Fatalln(e)
		}
		return &Store{baseDir: baseDir}
	} else if e != nil && os.IsExist(e) {
		log.Fatalln(e)
	}

	stat, err := f.Stat()
	if err != nil {
		log.Fatalln(err)
	}
	if !stat.IsDir() {
		log.Fatalln("there is a file in the way", baseDir)
	}
	return &Store{baseDir: baseDir}
}

// Set saves data.
func (s *Store) Set(key string, value []byte) (err error) {
	key = s.join(key)
	if e := os.MkdirAll(filepath.Dir(key), os.ModePerm); e != nil {
		return e
	}
	return ioutil.WriteFile(key, value, os.ModePerm)
}

// GetValue gets data from a key store.
func (s *Store) GetValue(key string) (value []byte, err error) {
	key = s.join(key)
	return ioutil.ReadFile(key)
}

// GetKeys gets all keys under a key store.
func (s *Store) GetKeys(bucket string) (keys []string, err error) {
	bucket = s.join(bucket)
	if !existsDir(bucket) {
		return nil, errors.New("uninitialized bucket")
	}
	fs, err := ioutil.ReadDir(bucket)
	if err != nil {
		return nil, err
	}
	for _, f := range fs {
		keys = append(keys, filepath.Base(f.Name()))
	}
	return keys, err
}

// Delete deletes data.
func (s *Store) Delete(key string) (err error) {
	key = s.join(key)
	return os.RemoveAll(key)
}

// BaseDir returns the base directory for storage..
func (s *Store) BaseDir() string {
	return s.baseDir
}

func (s *Store) join(key string) (fullpath string) {
	return filepath.Join(s.baseDir, filepath.Clean(key))
}

func existsDir(dpath string) bool {
	f, e := os.Open(dpath)
	if e != nil && os.IsNotExist(e) {
		return false
	}
	stat, e := f.Stat()
	return e != nil && stat.IsDir()
}

func existsFile(fpath string) bool {
	f, e := os.Open(fpath)
	if e != nil && os.IsNotExist(e) {
		return false
	}
	stat, e := f.Stat()
	return e != nil && !stat.IsDir()
}
