package kf

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	bolt "github.com/coreos/bbolt"
)

// StoreConfig configures a new Store.
type StoreConfig struct {
	BaseDir string
	Locking bool
	KV      bool
}

// Store is the root directory in which to keep the key-file store.
type Store struct {
	baseDir  string
	locking  bool
	locked   bool
	kv       bool
	db       *bolt.DB
	kvBucket []byte
}

// NewStore creates a new storage instance.
func NewStore(c *StoreConfig) (*Store, error) {
	if c.BaseDir == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		c.BaseDir = filepath.Join(usr.HomeDir, ".kf")
	}
	baseDir := filepath.Clean(c.BaseDir)
	baseDir, e := filepath.Abs(baseDir)
	if e != nil {
		return nil, e
	}
	s := &Store{baseDir: baseDir, locking: c.Locking, kv: c.KV}

	// getting too many open files error, turnin to darkside
	if s.kv {
		db, e := bolt.Open(s.join(".goddamnboltdatabasedontnameyourkeysthis"), os.ModePerm, nil)
		if e != nil {
			return nil, e
		}
		s.db = db
		s.kvBucket = []byte("data")
		if e := s.db.Update(func(tx *bolt.Tx) error {
			_, e := tx.CreateBucketIfNotExists(s.kvBucket)
			return e
		}); e != nil {
			return nil, e
		}
		return s, nil
	}

	f, e := os.Open(baseDir)

	// initialize new dir
	if e != nil && os.IsNotExist(e) {
		if e := os.MkdirAll(baseDir, os.ModePerm); e != nil {
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
		return nil, fmt.Errorf("there is a file in the way: %s", baseDir)
	}
	return s, nil
}

// Close closes a KV database.
func (s *Store) Close() error {
	if s.kv {
		return s.db.Close()
	}
	return errors.New("close only applies to KV db")
}

// Set saves data.
func (s *Store) Set(key string, value []byte) (err error) {
	if s.kv {
		keys := strings.Split(key, string(os.PathSeparator))
		return s.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.kvBucket)
			if len(keys) > 1 {
				for i, k := range keys {
					if i != len(keys)-1 {
						b, _ = b.CreateBucketIfNotExists([]byte(k))
					}
				}
			}
			return b.Put([]byte(keys[len(keys)-1]), value)
		})
	}

	for s.locking && s.locked {
	}
	if s.locking {
		s.lock()
		defer s.unlock()
	}
	key = s.join(key)
	if e := os.MkdirAll(filepath.Dir(key), os.ModePerm); e != nil {
		return e
	}
	return ioutil.WriteFile(key, value, os.ModePerm)
}

// GetValue gets data from a key store.
func (s *Store) GetValue(key string) (value []byte, err error) {
	if s.kv {
		keys := strings.Split(key, string(os.PathSeparator))
		err = s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.kvBucket)
			if len(keys) > 1 {
				for i, k := range keys {
					if i != len(keys)-1 {
						b = b.Bucket([]byte(k))
						if b == nil {
							return errors.New("bucket does not exist")
						}
					}
				}
			}
			value = b.Get([]byte(keys[len(keys)-1]))
			return nil
		})
		return value, err
	}

	for s.locking && s.locked {
	}
	if s.locking {
		s.lock()
		defer s.unlock()
	}
	key = s.join(key)
	return ioutil.ReadFile(key)
}

// GetKeys gets all keys under a key store.
func (s *Store) GetKeys(bucket string) (keys []string, err error) {
	if s.kv {
		keys := strings.Split(bucket, string(os.PathSeparator))
		err = s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.kvBucket)
			for _, k := range keys {
				b = b.Bucket([]byte(k))
				if b == nil {
					return errors.New("bucket does not exist")
				}
			}
			return b.ForEach(func(k []byte, v []byte) error {
				keys = append(keys, string(k))
				return nil
			})
		})
		return keys, err
	}

	for s.locking && s.locked {
	}
	if s.locking {
		s.lock()
		defer s.unlock()
	}
	bucket = s.join(bucket)
	if !existsDir(bucket) {
		return nil, errors.New("uninitialized bucket")
	}
	filepath.Walk(bucket, func(p string, f os.FileInfo, e error) error {
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
	// fs, err := ioutil.ReadDir(bucket)
	if err != nil {
		return nil, err
	}
	// for _, f := range fs {
	// 	keys = append(keys, filepath.Base(f.Name()))
	// }
	return keys, err
}

// Delete deletes data.
// TODO. Handle deleting buckets, too.
func (s *Store) Delete(key string) (err error) {
	if s.kv {
		delBucket := strings.HasSuffix(key, string(os.PathSeparator))
		keys := strings.Split(key, string(os.PathSeparator))
		return s.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.kvBucket)
			if len(keys) > 1 {
				for i, k := range keys {
					if !delBucket && i != len(keys)-1 {
						b = b.Bucket([]byte(k))
						if b == nil {
							return errors.New("uninitialized bucket")
						}
					} else {
						b = b.Bucket([]byte(k))
						if b == nil {
							return errors.New("uninitialized bucket")
						}
					}
				}
			}
			if delBucket {
				return tx.DeleteBucket([]byte(keys[len(keys)-1]))
			}
			return b.Delete([]byte(keys[len(keys)-1]))
		})
	}

	for s.locking && s.locked {
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
	os.Create(s.join(".LOCKDONOTFUCKWITHMEORUSEMEASAKEY"))
}

func (s *Store) unlock() {
	os.Remove(s.join(".LOCKDONOTFUCKWITHMEORUSEMEASAKEY"))
}

func (s *Store) isLocked() bool {
	return s.locked || existsFile(s.join(".LOCKDONOTFUCKWITHMEORUSEMEASAKEY"))
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
	return e == nil && stat.IsDir()
}

func existsFile(fpath string) bool {
	f, e := os.Open(fpath)
	if e != nil && os.IsNotExist(e) {
		return false
	}
	stat, e := f.Stat()
	return e == nil && !stat.IsDir()
}
