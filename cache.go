package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

// Item struct for each key/value item in the cache
type Item struct {
	key   string
	value []byte
}

// The cache to hold a map of all items sync controls the
// cache will only be accessed one at a time
type cache struct {
	items map[string]Item
	mu    sync.RWMutex
}

// REST function to add to cache
func (c *cache) Add(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	c.addKV(vars["key"], []byte(vars["value"]))
	w.Write([]byte(vars["key"] + " has been added\n"))
}

// REST function to delete from cache
func (c *cache) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	c.delKV(vars["key"])
	w.Write([]byte(vars["key"] + " has been deleted\n"))
}

// REST function to update key in the cache
func (c *cache) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	c.updateKV(vars["key"], []byte(vars["value"]))
	w.Write([]byte(vars["key"] + " has been updated\n"))
}

// REST function to retrieve value from cache and return as json
func (c *cache) Retrieve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	v := c.retKV(vars["key"])
	kv := map[string]string{"key": vars["key"], "value": string(v)}
	json.NewEncoder(w).Encode(kv)
}

// Check if key already exists
func (c *cache) ifexist(k string) bool {
	_, found := c.items[k]
	if !found {
		return false
	}
	return true
}

// Return the size of the cache
func (c *cache) cachesize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	n := len(c.items)
	return n
}

// Add to cache
func (c *cache) addKV(k string, v []byte) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.ifexist(k)
	if e != false {
		return []byte("The Key: " + k + " already exists\n")
	}
	c.items[k] = Item{
		value: v,
	}
	return []byte(k + " Added to cache\n")
}

// Update cache
func (c *cache) updateKV(k string, v []byte) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.ifexist(k)
	if e == false {
		return []byte("The Key: " + k + " doesn't exists\n")
	}
	c.items[k] = Item{
		value: v,
	}
	return []byte(k + " has been updated\n")
}

// Delete from cache
func (c *cache) delKV(k string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.ifexist(k)
	if e == false {
		return []byte("The Key: " + k + " doesn't exists\n")
	}
	delete(c.items, k)
	return []byte(k + " deleted\n")

}

// return the value for the given key
func (c *cache) retKV(k string) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.ifexist(k)
	if e == false {
		return nil
	}
	item, _ := c.items[k]
	return item.value
}

// Create new cache
func newcache() *cache {
	m := make(map[string]Item)
	c := &cache{
		items: m,
	}
	return c
}

// Writes the caches items to an IO writer using Gob
func (c *cache) Save(w io.Writer) (err error) {
	nc := make(map[string]string)
	for k := range c.items {
		nc[k] = string(c.items[k].value)
	}
	err = json.NewEncoder(w).Encode(nc)
	return err
}

// Saves the cache to a given filename
func (c *cache) SaveFile(fname string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	fp, err := os.Create(fname)
	if err != nil {
		return err
	}
	err = c.Save(fp)
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

// Clears cache and loads from given filename
func (c *cache) LoadFile(fname string) (*cache, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cont, err := ioutil.ReadFile(fname)
	if err != nil {
		return c, false
	}
	jm := make(map[string]string)
	if len(c.items) > 0 {
		for k := range c.items {
			delete(c.items, k)
		}
	}
	json.Unmarshal(cont, &jm)
	for k, v := range jm {
		c.items[k] = Item{
			value: []byte(v),
		}
	}
	return c, true
}

// Prints all items in cache in json format
func (c *cache) Printdb() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.items) == 0 {
		return []byte("Cache is empty")
	}
	js := "["
	for k, v := range c.items {
		js = js + fmt.Sprintf("%q:%q,", k, string(v.value))
	}
	js = js[:len(js)-1] + "]"
	return []byte(js)
}
