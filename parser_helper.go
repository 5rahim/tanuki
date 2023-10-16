package tanuki

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const dashes = "-\u2010\u2011\u2012\u2013\u2014\u2015"
const separators = "&~-\u2010\u2011\u2012\u2013\u2014\u2015"

func (p *parser) checkAnimeSeasonKeyword(tkn *token) bool {

	// Handle "4th Season", etc...
	prevToken, found := p.tokenizer.tokens.findPrevious(*tkn, tokenFlagsNotDelimiter)
	if found {
		num := getNumberFromOrdinal(prevToken.Content)
		if num != 0 {
			p.setAnimeSeason(prevToken, tkn, strconv.Itoa(num))
			return true
		}
	}

	nextToken, found := p.tokenizer.tokens.findNext(*tkn, tokenFlagsNotDelimiter)

	if len(tkn.Content) > 3 {
		if tkn.Content[0] == 'S' {
			// "S1-2" "S1&2" "S1~2" "S1-S2"
			seasonRe := regexp.MustCompile(`[Ss](?P<a>\d{1,2})[-&~](?P<b>\d{1,2})`)
			n1 := seasonRe.SubexpNames()
			r2 := seasonRe.FindAllStringSubmatch(tkn.Content, -1)
			if r2 != nil {
				md := map[string]string{}
				for i, n := range r2[0] {
					md[n1[i]] = n
				}
				a, foundA := md["a"]
				b, foundB := md["b"]
				if foundA && foundB {
					p.setAnimeSeason(tkn, nextToken, a)
					p.setAnimeSeason(tkn, nextToken, b)
				}

			}
		}
	}

	if found {
		// Handle "Seasons 1-2", etc...
		// If next token is either "1-2", "1~2", "1&2"
		parts := strings.Split(nextToken.Content, "-")
		if len(parts) != 2 {
			parts = strings.Split(nextToken.Content, "~")
		} else if len(parts) != 2 {
			parts = strings.Split(nextToken.Content, "&")
		}
		if len(parts) == 2 {
			// Make sure the length matches. e.g, 1-2, 01-02, 001-002
			if len(parts[0]) == len(parts[1]) && isNumeric(parts[0]) && isNumeric(parts[1]) {
				p.setAnimeSeason(tkn, nextToken, parts[0])
				p.setAnimeSeason(tkn, nextToken, parts[1])
			}
		}

	}

	if found && isNumeric(nextToken.Content) {

		// First check if there might be a range
		// Handle "Seasons 1 - 2", etc...
		// We don't consider "01 - 05" because this could accidentally capture the episode number
		rangeDelimiter, found := p.tokenizer.tokens.findNext(*nextToken, tokenFlagsNotDelimiter)
		skip := false

		if found {
			if isSeparatorCharacter(rangeDelimiter.Content) {
				nextUpToken, found := p.tokenizer.tokens.findNext(*rangeDelimiter, tokenFlagsNotDelimiter)
				if found {
					if len(nextToken.Content) == 1 && len(nextUpToken.Content) == 1 && isNumeric(nextUpToken.Content) {
						p.setAnimeSeason(tkn, nextToken, nextToken.Content)
						p.setAnimeSeason(tkn, nextUpToken, nextUpToken.Content)
						skip = true
					}
				}
			}
		}

		if !skip {
			p.setAnimeSeason(tkn, nextToken, nextToken.Content)
		}

		return true
	}
	return false
}

func (p *parser) setAnimeSeason(first, second *token, content string) {
	p.tokenizer.elements.insert(elementCategoryAnimeSeason, content)
	firstIdx := p.tokenizer.tokens.getIndex(*first, 0)
	secondIdx := p.tokenizer.tokens.getIndex(*second, firstIdx)
	firstTkn, _ := p.tokenizer.tokens.get(firstIdx)
	secondTkn, _ := p.tokenizer.tokens.get(secondIdx)
	firstTkn.Category = tokenCategoryIdentifier
	secondTkn.Category = tokenCategoryIdentifier
}

//////////////////////////////////////////////////

func (p *parser) checkAnimePartKeyword(tkn *token) bool {
	prevToken, found := p.tokenizer.tokens.findPrevious(*tkn, tokenFlagsNotDelimiter)
	if found {
		num := getNumberFromOrdinal(prevToken.Content)
		if num != 0 {
			p.setAnimePart(prevToken, tkn, strconv.Itoa(num))
			return true
		}
	}

	nextToken, found := p.tokenizer.tokens.findNext(*tkn, tokenFlagsNotDelimiter)

	// Handle "Parts 1-2" etc...
	parts := strings.Split(nextToken.Content, "-")
	if len(parts) != 2 {
		parts = strings.Split(nextToken.Content, "~")
	} else if len(parts) != 2 {
		parts = strings.Split(nextToken.Content, "&")
	}
	if len(parts) == 2 {
		if isNumeric(parts[0]) && isNumeric(parts[1]) {
			p.setAnimePart(tkn, nextToken, parts[0])
			p.setAnimePart(tkn, nextToken, parts[1])
		}
	}

	if found && isNumeric(nextToken.Content) {

		// First check if there might be a range
		// Handle "Parts 1 - 2", etc...
		// We don't consider "01 - 05" because this could accidentally capture the episode number
		rangeDelimiter, found := p.tokenizer.tokens.findNext(*nextToken, tokenFlagsNotDelimiter)
		skip := false

		if found {
			if isSeparatorCharacter(rangeDelimiter.Content) {
				nextUpToken, found := p.tokenizer.tokens.findNext(*rangeDelimiter, tokenFlagsNotDelimiter)
				if found {
					if len(nextToken.Content) == 1 && len(nextUpToken.Content) == 1 && isNumeric(nextUpToken.Content) {
						p.setAnimePart(tkn, nextToken, nextToken.Content)
						p.setAnimePart(tkn, nextUpToken, nextUpToken.Content)
						skip = true
					}
				}
			}
		}

		if !skip {
			p.setAnimePart(tkn, nextToken, nextToken.Content)
		}

		return true
	}
	return false
}

func (p *parser) setAnimePart(first, second *token, content string) {
	p.tokenizer.elements.insert(elementCategoryAnimePart, content)
	firstIdx := p.tokenizer.tokens.getIndex(*first, 0)
	secondIdx := p.tokenizer.tokens.getIndex(*second, firstIdx)
	firstTkn, _ := p.tokenizer.tokens.get(firstIdx)
	secondTkn, _ := p.tokenizer.tokens.get(secondIdx)
	firstTkn.Category = tokenCategoryIdentifier
	secondTkn.Category = tokenCategoryIdentifier
}

func (p *parser) buildElement(cat elementCategory, beginToken, endToken *token, keepDelimiters bool) {
	element := ""

	tknList := p.tokenizer.tokens.getList(-1, beginToken, endToken)
	for _, tkn := range tknList {
		if tkn.Category == tokenCategoryUnknown {
			element += tkn.Content
			tkn.Category = tokenCategoryIdentifier
		} else if tkn.Category == tokenCategoryBracket {
			element += tkn.Content
		} else if tkn.Category == tokenCategoryDelimiter {
			delimiter := tkn.Content
			if keepDelimiters {
				element += delimiter
			} else if tkn != beginToken && tkn != endToken {
				if delimiter == "," || delimiter == "&" {
					element += delimiter
				} else {
					element += " "
				}
			}
		}
	}

	if !keepDelimiters {
		element = strings.Trim(element, " "+dashes)
	}

	if element != "" {
		p.tokenizer.elements.insert(cat, strings.Trim(strings.ToValidUTF8(element, ""), " "))
	}
}

func findNonNumberInString(str string) int {
	for _, r := range str {
		if !unicode.IsDigit(r) {
			return strings.IndexRune(str, r)
		}
	}
	return -1
}

func isDashCharacter(str string) bool {
	if len(str) != 1 {
		return false
	}
	for _, dash := range dashes {
		if str == string(dash) {
			return true
		}
	}
	return false
}

func isSeparatorCharacter(str string) bool {
	if len(str) != 1 {
		return false
	}
	for _, dash := range separators {
		if str == string(dash) {
			return true
		}
	}
	return false
}

func isLatinRune(r rune) bool {
	return unicode.In(r, unicode.Latin)
}

func isMostlyLatinString(str string) bool {
	if len(str) <= 0 {
		return false
	}
	latinLength := 0
	nonLatinLength := 0
	for _, r := range str {
		if isLatinRune(r) {
			latinLength++
		} else {
			nonLatinLength++
		}
	}
	return latinLength > nonLatinLength
}

func stringToInt(str string) int {
	dotIndex := strings.IndexByte(str, '.')
	if dotIndex != -1 {
		str = str[:dotIndex]
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return i
}

func isCRC32(str string) bool {
	return len(str) == 8 && isHexadecimalString(str)
}

func isHexadecimalString(str string) bool {
	_, err := strconv.ParseInt(str, 16, 64)
	return err == nil
}

func isResolution(str string) bool {
	pattern := "\\d{3,4}([pP]|([xX\u00D7]\\d{3,4}))$"
	found, _ := regexp.Match(pattern, []byte(str))
	return found
}

func getNumberFromOrdinal(str string) int {
	ordinals := map[string]int{
		"1st": 1, "first": 1,
		"2nd": 2, "second": 2,
		"3rd": 3, "third": 3,
		"4th": 4, "fourth": 4,
		"5th": 5, "fifth": 5,
		"6th": 6, "sixth": 6,
		"7th": 7, "seventh": 7,
		"8th": 8, "eighth": 8,
		"9th": 9, "ninth": 9,
	}

	lowerStr := strings.ToLower(str)
	num := ordinals[lowerStr]
	return num
}

func findNumberInString(str string) int {
	for _, c := range str {
		if unicode.IsDigit(c) {
			return strings.IndexRune(str, c)
		}
	}
	return -1
}
