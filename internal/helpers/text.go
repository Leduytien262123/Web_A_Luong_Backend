package helpers

import (
	"strings"
)

// RemoveVietnameseDiacritics removes Vietnamese diacritics from text for fuzzy search
func RemoveVietnameseDiacritics(text string) string {
	// Chuyển về lowercase trước
	text = strings.ToLower(text)
	
	// Map các ký tự có dấu thành không dấu
	diacriticsMap := map[rune]rune{
		// A
		'à': 'a', 'á': 'a', 'ả': 'a', 'ã': 'a', 'ạ': 'a',
		'ă': 'a', 'ằ': 'a', 'ắ': 'a', 'ẳ': 'a', 'ẵ': 'a', 'ặ': 'a',
		'â': 'a', 'ầ': 'a', 'ấ': 'a', 'ẩ': 'a', 'ẫ': 'a', 'ậ': 'a',
		
		// E
		'è': 'e', 'é': 'e', 'ẻ': 'e', 'ẽ': 'e', 'ẹ': 'e',
		'ê': 'e', 'ề': 'e', 'ế': 'e', 'ể': 'e', 'ễ': 'e', 'ệ': 'e',
		
		// I
		'ì': 'i', 'í': 'i', 'ỉ': 'i', 'ĩ': 'i', 'ị': 'i',
		
		// O
		'ò': 'o', 'ó': 'o', 'ỏ': 'o', 'õ': 'o', 'ọ': 'o',
		'ô': 'o', 'ồ': 'o', 'ố': 'o', 'ổ': 'o', 'ỗ': 'o', 'ộ': 'o',
		'ơ': 'o', 'ờ': 'o', 'ớ': 'o', 'ở': 'o', 'ỡ': 'o', 'ợ': 'o',
		
		// U
		'ù': 'u', 'ú': 'u', 'ủ': 'u', 'ũ': 'u', 'ụ': 'u',
		'ư': 'u', 'ừ': 'u', 'ứ': 'u', 'ử': 'u', 'ữ': 'u', 'ự': 'u',
		
		// Y
		'ỳ': 'y', 'ý': 'y', 'ỷ': 'y', 'ỹ': 'y', 'ỵ': 'y',
		
		// D
		'đ': 'd',
	}
	
	var result strings.Builder
	for _, r := range text {
		if normalized, exists := diacriticsMap[r]; exists {
			result.WriteRune(normalized)
		} else {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// NormalizeSearchText normalizes text for search by removing diacritics and extra spaces
func NormalizeSearchText(text string) string {
	// Remove diacritics
	normalized := RemoveVietnameseDiacritics(text)
	
	// Remove extra spaces and trim
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	return normalized
}

// IsSearchMatch checks if the search term matches the target text using fuzzy matching
func IsSearchMatch(searchTerm, targetText string) bool {
	if searchTerm == "" {
		return true
	}
	
	normalizedSearch := NormalizeSearchText(searchTerm)
	normalizedTarget := NormalizeSearchText(targetText)
	
	return strings.Contains(normalizedTarget, normalizedSearch)
}