package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
)

const (
	metaHeight  = "__meta_height"
	metaAppHash = "__meta_apphash"
)

type State struct {
	db *badger.DB
}

func NewState(path string) (*State, error) {
	log.Printf("[kvstore-state] NewState: opening database path=%q", path)
	opts := badger.DefaultOptions(path).WithLoggingLevel(badger.WARNING)
	db, err := badger.Open(opts)
	if err != nil {
		log.Printf("[kvstore-state] NewState: failed to open database path=%q err=%v", path, err)
		return nil, fmt.Errorf("open badger: %w", err)
	}
	log.Printf("[kvstore-state] NewState: database opened successfully path=%q", path)
	return &State{db: db}, nil
}

func (s *State) Set(key, value []byte) error {
	log.Printf("[kvstore-state] Set: key=%q valueLen=%d", string(key), len(value))
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	if err != nil {
		log.Printf("[kvstore-state] Set: error key=%q err=%v", string(key), err)
	}
	return err
}

func (s *State) Get(key []byte) ([]byte, error) {
	var val []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		log.Printf("[kvstore-state] Get: key=%q found=false err=%v", string(key), err)
		return nil, err
	}
	log.Printf("[kvstore-state] Get: key=%q found=true valueLen=%d", string(key), len(val))
	return val, nil
}

func (s *State) BatchSet(pairs [][2][]byte) error {
	log.Printf("[kvstore-state] BatchSet: pairCount=%d", len(pairs))
	err := s.db.Update(func(txn *badger.Txn) error {
		for _, pair := range pairs {
			log.Printf("[kvstore-state] BatchSet: key=%q valueLen=%d", string(pair[0]), len(pair[1]))
			if err := txn.Set(pair[0], pair[1]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[kvstore-state] BatchSet: error pairCount=%d err=%v", len(pairs), err)
	}
	return err
}

func (s *State) SaveMeta(height int64, appHash []byte) error {
	log.Printf("[kvstore-state] SaveMeta: height=%d appHash=%x", height, appHash)
	err := s.db.Update(func(txn *badger.Txn) error {
		hb := make([]byte, 8)
		binary.BigEndian.PutUint64(hb, uint64(height))
		if err := txn.Set([]byte(metaHeight), hb); err != nil {
			return err
		}
		return txn.Set([]byte(metaAppHash), appHash)
	})
	if err != nil {
		log.Printf("[kvstore-state] SaveMeta: error height=%d err=%v", height, err)
	}
	return err
}

func (s *State) LoadMeta() (int64, []byte, error) {
	log.Printf("[kvstore-state] LoadMeta: loading metadata")
	var height int64
	var appHash []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(metaHeight))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		hb, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		height = int64(binary.BigEndian.Uint64(hb))

		item, err = txn.Get([]byte(metaAppHash))
		if err != nil {
			return err
		}
		appHash, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		log.Printf("[kvstore-state] LoadMeta: error err=%v", err)
	} else {
		log.Printf("[kvstore-state] LoadMeta: height=%d appHash=%x", height, appHash)
	}
	return height, appHash, err
}

func (s *State) Hash() []byte {
	h := sha256.New()
	s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			// skip meta keys
			if string(key) == metaHeight || string(key) == metaAppHash {
				continue
			}
			h.Write(key)
			item.Value(func(val []byte) error {
				h.Write(val)
				return nil
			})
		}
		return nil
	})
	return h.Sum(nil)
}

func (s *State) Close() error {
	log.Printf("[kvstore-state] Close: shutting down database")
	err := s.db.Close()
	if err != nil {
		log.Printf("[kvstore-state] Close: error err=%v", err)
	} else {
		log.Printf("[kvstore-state] Close: database closed successfully")
	}
	return err
}
