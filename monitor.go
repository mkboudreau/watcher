package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	IntervalStartToken string = "--- STARTING TO MONITOR FOR CHANGES ---"
	IntervalEndToken          = "--- DONE MONITORING CHANGES ---"
)

type DirectoryMonitor struct {
	Dir             string
	NoTraverse      bool
	Interval        int
	IncludesPattern string
	ExcludesPattern string
	cache           map[string]os.FileInfo
}

func NewDirectoryMonitor() *DirectoryMonitor {
	return &DirectoryMonitor{
		cache: make(map[string]os.FileInfo),
	}
}

func (monitor *DirectoryMonitor) StartDirectoryMonitor(changes chan<- string) <-chan bool {
	interval := time.Duration(monitor.Interval) * time.Second
	quit := make(chan bool)
	go func() {
		monitor.buildInitialCache()
		ticker := time.Tick(interval)
		for {
			changes <- string(IntervalStartToken)
			err := monitor.walkDirectoryForChanges(monitor.Dir, changes)
			changes <- string(IntervalEndToken)
			if err != nil && err != filepath.SkipDir {
				log.Println("Caught fatal error:", err)
				break
			}
			<-ticker
		}
		close(changes)
		close(quit)
	}()
	return quit
}

func (monitor *DirectoryMonitor) buildInitialCache() error {
	doNothingChannel := make(chan string)
	defer close(doNothingChannel)
	go func() {
		for val := range doNothingChannel {
			val = val
		}
	}()
	err := monitor.walkDirectoryForChanges(monitor.Dir, doNothingChannel)
	if err != nil && err != filepath.SkipDir {
		return err
	}
	return nil
}

func (monitor *DirectoryMonitor) walkDirectoryForChanges(dirname string, changes chan<- string) error {
	if debug {
		log.Println("processing directory", dirname)
	}
	return filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if monitor.isExcluded(path) {
			if debug {
				log.Println("skipping [", path, "]")
			}
			return filepath.SkipDir
		}

		if info.IsDir() {
			if !monitor.NoTraverse && dirname != path {
				walkErr := monitor.walkDirectoryForChanges(path, changes)
				if walkErr != nil {
					return walkErr
				}
			}
			dirErr := monitor.checkForDirectoryChanges(path, changes)
			if dirErr != nil {
				return dirErr
			}
		} else {
			if monitor.isIncluded(path) {
				isChanged, message := monitor.isFileChanged(path, info)
				if isChanged {
					changes <- message
					monitor.removeCacheEntry(path)
					monitor.addCacheEntry(path, info)
				}
			} else {
				if debug {
					log.Println("skipping [", path, "]")
				}
				//return filepath.SkipDir
			}
		}

		return nil
	})
}

func (monitor *DirectoryMonitor) isIncluded(path string) bool {
	patterns := strings.Split(monitor.IncludesPattern, ",")
	basename := filepath.Base(path)

	for _, pattern := range patterns {
		if trace {
			log.Println("checking include pattern", pattern, "for path", basename)
		}
		ok, err := filepath.Match(pattern, basename)
		if ok && err == nil {
			if trace {
				log.Println(" > include matched pattern", pattern)
			}
			return true
		} else if err != nil {
			if trace {
				log.Println(" > include error", err)
			}
			return false
		}
	}

	return false
}
func (monitor *DirectoryMonitor) isExcluded(path string) bool {
	patterns := strings.Split(monitor.ExcludesPattern, ",")
	basename := filepath.Base(path)

	for _, pattern := range patterns {
		if trace {
			log.Println("checking exclude pattern", pattern, "for path", basename)
		}
		ok, err := filepath.Match(pattern, basename)
		if ok && err == nil {
			if trace {
				log.Println(" > exclude matched pattern", pattern)
			}
			return true
		} else if err != nil {
			if trace {
				log.Println(" > exclude matched pattern", pattern)
			}
			return false
		}
	}

	return false
}
func (monitor *DirectoryMonitor) checkForDirectoryChanges(dirname string, changes chan<- string) error {
	_, ok := monitor.cache[dirname]
	if !ok {
		err := monitor.handleNewDirectory(dirname, changes)
		if err != nil {
			return err
		}
	} else {
		err := monitor.handleExistingDirectory(dirname, changes)
		if err != nil {
			return err
		}
	}
	return nil
}

func (monitor *DirectoryMonitor) handleNewDirectory(dirname string, changes chan<- string) error {
	changes <- fmt.Sprintf("Found new directory: %v", dirname)
	dirinfo, err := os.Stat(dirname)
	if err != nil {
		log.Println("Found error in CheckForDeletedItemsInDirectory: ", err)
		return err
	}
	monitor.addCacheEntry(dirname, dirinfo)
	return nil
}

func (monitor *DirectoryMonitor) handleExistingDirectory(dirname string, changes chan<- string) error {
	err := monitor.handleRemovedItemsFromDirectory(dirname, changes)
	if err != nil {
		return err
	}
	return nil
}

func (monitor *DirectoryMonitor) handleRemovedItemsFromDirectory(dirname string, changes chan<- string) error {
	items, err := ioutil.ReadDir(dirname)
	if err != nil {
		log.Println("Found error: ", err)
		return err
	}

	for _, item := range items {
		itemPath := filepath.Join(dirname, item.Name())
		_, err := os.Open(itemPath)
		if err != nil {
			if os.IsNotExist(err) {
				changes <- fmt.Sprintf("Found removed file or directory: %v", err)
				monitor.removeCacheEntry(dirname)
			} else {
				log.Println("Found error that isnt IsNotExist: ", err)
				return err
			}
		}
	}

	return nil
}

func (monitor *DirectoryMonitor) isFileChanged(path string, info os.FileInfo) (bool, string) {
	cachedInfo, ok := monitor.cache[path]
	if !ok {
		return true, fmt.Sprintf("Found new file: %v", path)
	}

	if info.Size() != cachedInfo.Size() {
		return true, fmt.Sprintf("Found modified file: %v", path)
	}

	return false, ""
}

func (monitor *DirectoryMonitor) removeCacheEntry(dirname string) {
	delete(monitor.cache, dirname)
}

func (monitor *DirectoryMonitor) addCacheEntry(dirname string, info os.FileInfo) {
	monitor.cache[dirname] = info
}

func (monitor *DirectoryMonitor) String() string {
	return fmt.Sprintf("Directory [%v]; No Traverse [%v]; Interval [%v]; IncludesPattern [%v]; ExcludesPattern [%v]", monitor.Dir, monitor.NoTraverse, monitor.Interval, monitor.IncludesPattern, monitor.ExcludesPattern)
}
