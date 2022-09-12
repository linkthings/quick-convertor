package main

func filterRecord(record []interface{}, fieldSlice []*Field, filters []*Filter) bool {

	// skip the filter function if no filter is defined
	if len(filters) == 0 {
		return true
	}

	recordLen := len(record)
	for _, iter := range filters {
		if iter.fieldPos > 0 && iter.fieldPos < recordLen {
			field := record[iter.fieldPos].(string)
			val := iter.valueMap[field]
			if val == 1 {
				return true
			}
		}
	}
	return false
}
