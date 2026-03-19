package mysql

// SplitMultiQuery 将多条用分号分隔的SQL拆分为单条语句
//
// 该函数会正确处理字符串中的分号,不会在字符串中间拆分。
func SplitMultiQuery(sql string) []string {
	var queries []string
	var currentQuery []rune
	var inString bool
	var stringChar rune
	var escapeNext bool

	for _, char := range sql {
		if escapeNext {
			currentQuery = append(currentQuery, char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			currentQuery = append(currentQuery, char)
			escapeNext = true
			continue
		}

		if char == '\'' || char == '"' {
			if !inString {
				inString = true
				stringChar = char
			} else if char == stringChar {
				inString = false
				stringChar = 0
			}
			currentQuery = append(currentQuery, char)
			continue
		}

		if char == ';' && !inString {
			query := string(currentQuery)
			query = trimSpace(query)
			if query != "" {
				queries = append(queries, query)
			}
			currentQuery = nil
			continue
		}

		currentQuery = append(currentQuery, char)
	}

	// 添加最后一条查询
	query := string(currentQuery)
	query = trimSpace(query)
	if query != "" {
		queries = append(queries, query)
	}

	return queries
}

// trimSpace 去除字符串首尾的空白字符
func trimSpace(s string) string {
	if len(s) == 0 {
		return s
	}

	// 去除开头空白
	start := 0
	for start < len(s) && isWhitespace(rune(s[start])) {
		start++
	}

	// 去除结尾空白
	end := len(s)
	for end > start && isWhitespace(rune(s[end-1])) {
		end--
	}

	return s[start:end]
}

// isWhitespace 判断是否是空白字符
func isWhitespace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}
