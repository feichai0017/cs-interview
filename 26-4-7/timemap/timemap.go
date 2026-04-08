package timemap

type Entry struct {
	value 		string
	timestamp 	int
}

type TimeMap struct {
	data map[string][]Entry
}


func New() TimeMap {
	return TimeMap{
		data: make(map[string][]Entry),
	}
}

func (tm *TimeMap) Set(key string, value string, timestamp int)  {
	tm.data[key] = append(tm.data[key], Entry{
		value: value, 
		timestamp: timestamp,
	})
}

func (tm *TimeMap) Get(key string, timestamp int) string {
	entries, ok := tm.data[key]
	if !ok || len(entries) == 0 {
		return ""
	}
	l, r := 0, len(entries) - 1
	ans := -1
	
	for l <= r {
		mid := l + (r - l)/2
		if entries[mid].timestamp <= timestamp {
			ans = mid
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	if ans == -1 {
		return ""
	}
	return entries[ans].value
}