package freecache

import (
	"sync"

	"github.com/coocood/freecache"
	"github.com/senizdegen/sdu-housing/user-service/pkg/cache"
)

type repository struct {
	sync.Mutex
	cache *freecache.Cache
}

func (r *repository) EntryCount() int64 {
	r.Lock()
	defer r.Unlock()

	return r.cache.EntryCount()
}

func (r *repository) HitCount() int64 {
	r.Lock()
	defer r.Unlock()

	return r.cache.HitCount()
}

func (r *repository) MissCount() int64 {
	r.Lock()
	defer r.Unlock()

	return r.cache.MissCount()
}

func NewCacheRepo(size int) cache.Repository {
	return &repository{cache: freecache.NewCache(size)}
}

func (r *repository) GetIterator() cache.Iterator {
	return &iterator{r.cache.NewIterator()}
}

func (r *repository) Get(uuid []byte) ([]byte, error) {
	r.Lock()
	defer r.Unlock()
	got, err := r.cache.Get(uuid)
	return got, err
}

func (r *repository) Set(key, val []byte, expireIn int) error {
	r.Lock()
	defer r.Unlock()

	err := r.cache.Set(key, val, expireIn)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) Del(key []byte) (affected bool) {
	r.Lock()
	defer r.Unlock()

	return r.cache.Del(key)
}
