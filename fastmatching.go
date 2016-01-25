package fastmatching

import (
	"container/list"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

type searchItem struct {
	key   []rune
	value int32
}

type ByRune []*searchItem

func (a ByRune) Len() int {
	return len(a)
}
func (a ByRune) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByRune) Less(i, j int) bool {
	len1 := len(a[i].key)
	len2 := len(a[j].key)
	minlen := len1
	if minlen > len2 {
		minlen = len2
	}
	for k := 0; k < minlen; k++ {
		if a[i].key[k] != a[j].key[k] {
			return a[i].key[k] < a[j].key[k]
		}
	}
	return len1 < len2
}

type FastMatching struct {
	dataList    *list.List
	searchList  *[]*searchItem
	needReIndex bool
	locker      sync.Locker
	nullResult  []int32
}

func NewFastMatching() *FastMatching {
	svc := new(FastMatching)
	svc.dataList = list.New()
	svc.needReIndex = true
	svc.locker = &sync.Mutex{}
	svc.searchList = nil
	svc.nullResult = make([]int32, 0)
	return svc
}

func (svc *FastMatching) DumpSearchList() {
	sz := len(*svc.searchList)
	fmt.Println("Size of Search List:", sz)

	for i := 0; i < sz; i++ {
		fmt.Printf("Key=%q, Value=%d\n", (*svc.searchList)[i].key, (*svc.searchList)[i].value)
	}
}

func String2RuneList(str string) ([]rune, int) {
	buf := []byte(str)
	sz := utf8.RuneCount(buf)

	runeList := make([]rune, sz)

	for i := 0; len(buf) > 0; i++ {
		r, size := utf8.DecodeRune(buf)
		buf = buf[size:]
		runeList[i] = r
	}

	return runeList, sz
}

func (svc *FastMatching) reindex() {
	svc.locker.Lock()

	if svc.needReIndex {
		tmpList := list.New()

		for e := svc.dataList.Front(); e != nil; e = e.Next() {
			element := e.Value
			if item, ok := element.(*searchItem); ok {
				k := item.key
				v := item.value
				sz := len(k)

				tmpList.PushBack(item)

				for i := 1; i < sz; i++ {
					newSlice := make([]rune, sz-i)
					copy(newSlice, k[i:])
					tmpList.PushBack(&searchItem{newSlice, v})
				}
			} else {
				fmt.Println("Oops!")
			}
		}

		itemList := make([]*searchItem, tmpList.Len())
		i := 0

		for e := tmpList.Front(); e != nil; e = e.Next() {
			if item, ok := e.Value.(*searchItem); ok {
				itemList[i] = item
			}
			i++
		}

		svc.searchList = &itemList
	}
	svc.needReIndex = false

	sort.Sort(ByRune(*svc.searchList))

	svc.locker.Unlock()

	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	/*
		fmt.Printf("Sys:%d,Alloc:%d,Idle:%d,Released:%d\n", m.HeapSys, m.HeapAlloc,
			m.HeapIdle, m.HeapReleased)

	*/
	//svc.DumpSearchList()
}

func (svc *FastMatching) subRuneCompare(pos int, sub []rune) int {
	if pos < 0 {
		return -1
	}
	if pos >= len(*svc.searchList) {
		return 1
	}

	full := (*svc.searchList)[pos].key
	slen := len(sub)
	flen := len(full)
	cmplen := slen

	if flen < cmplen {
		cmplen = flen
	}

	for i := 0; i < cmplen; i++ {
		if full[i] > sub[i] {
			return 1
		} else if full[i] < sub[i] {
			return -1
		}
	}

	if slen > flen {
		return -1
	}
	return 0
}

func (svc *FastMatching) findMatches(target []rune) []int32 {
	if svc.needReIndex {
		svc.reindex()
	}

	from := 0
	to := len(*svc.searchList) - 1

	for from <= to {
		pos := (from + to) >> 1
		cmp := svc.subRuneCompare(pos, target)

		if cmp == 0 {
			from = pos
			to = pos
			for svc.subRuneCompare(from-1, target) == 0 {
				from--
			}
			for svc.subRuneCompare(to+1, target) == 0 {
				to++
			}

			// fmt.Printf("from=%d, to=%d\n", from, to)
			list := make([]int32, to-from+1)

			j := 0

			for i := from; i <= to; i++ {
				list[j] = (*svc.searchList)[i].value
				j++
			}
			// fmt.Printf("%v\n", list)
			return list
		} else if cmp < 0 {
			from = pos + 1
		} else {
			to = pos - 1
		}
	}

	return nil
}

func (svc *FastMatching) RegistData(keyword string, value int32) bool {
	lcstr := strings.ToLower(keyword)
	key := []byte(lcstr)

	if utf8.Valid(key) {
		runelist, _ := String2RuneList(lcstr)
		svc.dataList.PushBack(&searchItem{key: runelist, value: value})
		svc.needReIndex = true
		return true
	}
	return false
}

func (svc *FastMatching) RetrieveData(keyword string) []int32 {
	lcstr := strings.ToLower(keyword)
	key := []byte(lcstr)

	if utf8.Valid(key) {
		runelist, _ := String2RuneList(lcstr)
		if result := svc.findMatches(runelist); result == nil {
			return svc.nullResult
		} else {
			return result
		}
	}
	return nil
}

func (svc *FastMatching) Clear() {
	svc.dataList.Init()
	svc.needReIndex = true
}
