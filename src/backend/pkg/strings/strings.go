package strings

// Echo: returns the string.
// Returns the string.
func Echo(s string) string {
    return s
}

// Reverse: reverses the string.
// Returns the reversed string.
func Reverse(s string) string {
    rev := []rune(s)
    
    for i, j := 0, len(rev) - 1; i < j; i, j = i + 1, j - 1 {
        rev[i], rev[j] = rev[j], rev[i]
    }
    
    return string(rev)
}