package tanuki

import (
	"regexp"
	"strconv"
	"strings"
)

type parser struct {
	tokenizer *tokenizer
}

func newParser(tkz *tokenizer) *parser {
	psr := parser{
		tokenizer: tkz,
	}
	return &psr
}

// Parse order
func (p *parser) parse() {
	p.preProcessing()

	p.searchForShortenedRange()

	p.searchForKeywords()

	p.searchForIsolatedNumbers()

	if p.tokenizer.options.ParseEpisodeNumber {
		p.searchForEpisodeNumber()
	}

	p.searchForEpisodeNumberAtTheStart() // POST PROCESSING

	p.searchForAnimeTitle()

	if p.tokenizer.options.ParseReleaseGroup && !p.tokenizer.elements.contains(elementCategoryReleaseGroup) {
		p.searchForReleaseGroup()
	}

	if p.tokenizer.options.ParseEpisodeTitle && p.tokenizer.elements.contains(elementCategoryEpisodeNumber) {
		p.searchForEpisodeTitle()
	}

	p.postProcessing()
}

func (p *parser) preProcessing() {
	for _, tkn := range p.tokenizer.tokens.getListFlag(tokenFlagsUnknown) {
		// Pre-processing
		// Check if the word is combined with another word like "1+OVA" which happens often
		parts := strings.Split(tkn.Content, "+")
		if len(parts) == 2 {
			// if first part is a number \d{1,2} and the second part is not
			// this makes sure we don't "un-unite" ranges
			if isNumeric(parts[0]) && len(parts[0]) <= 2 && !isNumeric(parts[1]) {
				tkn.Content = parts[0]
				p.tokenizer.addToken(tokenFlagsUnknown, parts[1], true)
			}
		}

	}
}

// Handle "S1-2" etc...
func (p parser) searchForShortenedRange() {
	for _, tkn := range p.tokenizer.tokens.getListFlag(tokenFlagsUnknown) {

		if len(tkn.Content) > 3 {
			if tkn.Content[0] == 'S' {
				seasonRe := regexp.MustCompile(`[Ss](?P<a>\d{1,2})[-&~](?P<b>\d{1,2})`)
				seasonMatch := seasonRe.FindAllStringSubmatch(tkn.Content, -1)
				if seasonMatch != nil {
					p.checkAnimeSeasonKeyword(tkn)
				}
			}
		}

	}
}

// From the entire list of tokens, find specific keywords
func (p *parser) searchForKeywords() {
	for _, tkn := range p.tokenizer.tokens.getListFlag(tokenFlagsUnknown) {

		w := tkn.Content
		w = strings.Trim(w, " -")

		if w == "" {
			continue
		}

		// Don't bother if the word is a number that cannot be CRC
		if len(w) != 8 && isNumeric(w) {
			continue
		}

		category := elementCategoryUnknown
		kd, found := p.tokenizer.keywordManager.findWithoutCategory(p.tokenizer.keywordManager.normalize(w))
		if found {
			category = kd.category
			if !p.tokenizer.options.ParseReleaseGroup && category == elementCategoryReleaseGroup {
				continue
			}
			// Skip If the category of the keyword is searchable but the keyword itself isn't
			if !category.isSearchable() || !kd.options.searchable {
				continue
			}
			// Skip If the category is singular
			if category.isSingular() && p.tokenizer.elements.contains(category) {
				continue
			}

			if category == elementCategoryAnimeSeasonPrefix { // Season X, Seasons X, Xth season,...
				p.checkAnimeSeasonKeyword(tkn)
				continue
			} else if category == elementCategoryAnimePartPrefix { // Part X
				p.checkAnimePartKeyword(tkn)
				continue
			} else if category == elementCategoryEpisodePrefix { // Ep X
				if kd.options.valid {
					p.checkExtentKeyword(elementCategoryEpisodeNumber, tkn)
				}
				continue
			} else if category == elementCategoryReleaseVersion {
				w = w[1:] // number without "v"
			} else if category == elementCategoryVolumePrefix {
				p.checkExtentKeyword(elementCategoryVolumeNumber, tkn)
				continue
			}
		} else {
			if !p.tokenizer.elements.contains(elementCategoryFileChecksum) && isCRC32(w) {
				category = elementCategoryFileChecksum
			} else if !p.tokenizer.elements.contains(elementCategoryVideoResolution) && isResolution(w) {
				category = elementCategoryVideoResolution
			}
		}

		if category != elementCategoryUnknown {
			p.tokenizer.elements.insert(category, w)
			if kd.empty() || kd.options.identifiable {
				tkn.Category = tokenCategoryIdentifier
			}
		}
	}
}

// Detect isolated numbers and process them accordingly
func (p *parser) searchForIsolatedNumbers() {
	for _, tkn := range p.tokenizer.tokens.getListFlag(tokenFlagsUnknown) {

		if !isNumeric(tkn.Content) {
			continue
		}
		isolated := p.tokenizer.tokens.isTokenIsolated(*tkn)
		if !isolated {
			continue
		}

		n, _ := strconv.Atoi(tkn.Content)

		if n >= animeYearMin && n <= animeYearMax {
			if !p.tokenizer.elements.contains(elementCategoryAnimeYear) {
				p.tokenizer.elements.insert(elementCategoryAnimeYear, tkn.Content)
				tkn.Category = tokenCategoryIdentifier
				continue
			}
		}

		if n == 480 || n == 720 || n == 1080 {
			if !p.tokenizer.elements.contains(elementCategoryVideoResolution) {
				p.tokenizer.elements.insert(elementCategoryVideoResolution, tkn.Content)
				tkn.Category = tokenCategoryIdentifier
				continue
			}
		}
	}
}

// Handle cases like "05 - Episode title.mkv"
// It shouldn't affect "86 - Eighty Six - 01.mkv" because 01 already got detected
func (p *parser) searchForEpisodeNumberAtTheStart() {
	for _, tkn := range p.tokenizer.tokens.getListFlag(tokenFlagsUnknown) {

		if !isNumeric(tkn.Content) {
			continue
		}

		prev, found := p.tokenizer.tokens.get(0)

		// is first token, is at least 2 characters long
		// episode number has not been parsed
		// /!\ This could cause problems
		if found && len(tkn.Content) > 1 && prev.Content == tkn.Content && !p.tokenizer.elements.contains(elementCategoryEpisodeNumber) {
			p.tokenizer.elements.insert(elementCategoryEpisodeNumber, tkn.Content)
			tkn.Category = tokenCategoryIdentifier
		}
	}
}

func (p *parser) searchForEpisodeNumber() {
	tkns := p.tokenizer.tokens.getListFlag(tokenFlagsUnknown)
	if len(tkns) == 0 {
		return
	}

	p.tokenizer.elements.setCheckAltNumber(p.tokenizer.elements.contains(elementCategoryEpisodeNumber))

	match := p.searchForEpisodePatterns(tkns)
	if match {
		return
	}

	if p.tokenizer.elements.contains(elementCategoryEpisodeNumber) {
		return
	}

	var numericTokens tokens
	for _, v := range tkns {
		if isNumeric(v.Content) {
			numericTokens = append(numericTokens, v)
		}
	}

	if len(numericTokens) == 0 {
		return
	}

	if p.searchForEquivalentNumbers(numericTokens) {
		return
	}

	if p.searchForSeparatedNumbers(numericTokens) {
		return
	}

	if p.searchForIsolatedNumbersTokens(numericTokens) {
		return
	}

	if p.searchForLastNumber(numericTokens) {
		return
	}
}

func (p *parser) searchForAnimeTitle() {
	enclosedTitle := false

	// Find the first token that is not enclosed or unknown
	tokenBegin, found := p.tokenizer.tokens.find(tokenFlagsNotEnclosed | tokenFlagsUnknown)

	if !found {
		enclosedTitle = true
		tokenBegin, found = p.tokenizer.tokens.get(0)
		skippedPreviousGroup := false
		for found {
			tokenBegin, found = p.tokenizer.tokens.findNext(*tokenBegin, tokenFlagsUnknown)
			if !found {
				break
			}
			if isMostlyLatinString(tokenBegin.Content) {
				if skippedPreviousGroup {
					break
				}
			}
			tokenBegin, found = p.tokenizer.tokens.findNext(*tokenBegin, tokenFlagsBracket)
			skippedPreviousGroup = true
		}
	}

	// If the token is empty
	if tokenBegin.empty() {
		return
	}

	targetFlag := tokenFlagsNone
	if enclosedTitle {
		targetFlag = tokenFlagsBracket
	}

	tokenEnd, foundTokenEnd := p.tokenizer.tokens.findNext(*tokenBegin, tokenFlagsIdentifier|targetFlag)
	if !enclosedTitle {
		lastBracket := tokenEnd
		bracketOpen := false
		tknList := p.tokenizer.tokens.getList(tokenFlagsBracket, tokenBegin, tokenEnd)
		for _, tkn := range tknList {
			lastBracket = tkn
			bracketOpen = !bracketOpen
		}
		if bracketOpen {
			tokenEnd = lastBracket
		}
	}

	// When no end token is detected, set all next tokens as title.
	// This assumes that previous processing would have removed episode numbers and seasons.
	// This makes sure that a title is always returned
	if !foundTokenEnd {
		lastToken, found := p.tokenizer.tokens.get(len(p.tokenizer.tokens.getListFlag(tokenFlagsValid)) - 1)
		if found {
			p.buildElement(elementCategoryAnimeTitle, tokenBegin, lastToken, true)
		}
	}

	if !enclosedTitle {
		tkn, found := p.tokenizer.tokens.findPrevious(*tokenEnd, tokenFlagsNotDelimiter)
		if !found {
			return
		}
		for tkn.Category == tokenCategoryBracket && tkn.Content != ")" {
			tkn, found = p.tokenizer.tokens.findPrevious(*tkn, tokenFlagsBracket)
			if found {
				if !tkn.empty() {
					tokenEnd = tkn
					tkn, _ = p.tokenizer.tokens.findPrevious(*tokenEnd, tokenFlagsNotDelimiter)
				}
			}
		}
	}

	tokenEnd, _ = p.tokenizer.tokens.findPrevious(*tokenEnd, tokenFlagsValid)

	p.buildElement(elementCategoryAnimeTitle, tokenBegin, tokenEnd, false)
}

func (p *parser) searchForReleaseGroup() {
	tokenEnd := &token{}
	tokenBegin := &token{}
	previousToken := &token{}
	for {
		if !tokenEnd.empty() {
			tokenBegin, _ = p.tokenizer.tokens.findNext(*tokenEnd, tokenFlagsEnclosed|tokenFlagsUnknown)
		} else {
			tokenBegin, _ = p.tokenizer.tokens.find(tokenFlagsEnclosed | tokenFlagsUnknown)
		}
		if tokenBegin.empty() {
			return
		}
		tokenEnd, _ = p.tokenizer.tokens.findNext(*tokenBegin, tokenFlagsBracket|tokenFlagsIdentifier)
		if tokenEnd.empty() {
			return
		}
		if tokenEnd.Category != tokenCategoryBracket {
			continue
		}
		previousToken, _ = p.tokenizer.tokens.findPrevious(*tokenBegin, tokenFlagsNotDelimiter)
		if !previousToken.empty() && previousToken.Category != tokenCategoryBracket {
			continue
		}

		tokenEnd, _ = p.tokenizer.tokens.findPrevious(*tokenEnd, tokenFlagsValid)

		list := p.tokenizer.tokens.getList(tokenFlagsValid, tokenBegin, tokenEnd)

		for _, tk := range list {
			// If "Season" "Part" "EP" is found inside, process it accordingly
			_, foundS := p.tokenizer.keywordManager.find(p.tokenizer.keywordManager.normalize(tk.Content), elementCategoryAnimeSeasonPrefix)
			_, foundP := p.tokenizer.keywordManager.find(p.tokenizer.keywordManager.normalize(tk.Content), elementCategoryAnimePartPrefix)
			_, foundE := p.tokenizer.keywordManager.find(p.tokenizer.keywordManager.normalize(tk.Content), elementCategoryEpisodePrefix)
			_, foundAT := p.tokenizer.keywordManager.find(p.tokenizer.keywordManager.normalize(tk.Content), elementCategoryAnimeType)
			if foundS || foundP || foundE || foundAT {
				// Remove brackets
				tokenBegin.Category = tokenCategoryInvalid
				tokenEnd.Category = tokenCategoryInvalid
				if foundS {
					p.checkAnimeSeasonKeyword(tk)
				} else if foundP {
					p.checkAnimePartKeyword(tk)
				} else if foundE {
					p.searchForEpisodeNumber()
				} else if foundAT {
					p.tokenizer.elements.insert(elementCategoryAnimeType, tk.Content)
					tk.Category = tokenCategoryIdentifier
				}
				return
			}
		}

		p.buildElement(elementCategoryReleaseGroup, tokenBegin, tokenEnd, true)
		return
	}
}

func (p *parser) searchForEpisodeTitle() {
	tokenEnd := &token{}
	tokenBegin := &token{}
	for {
		if !tokenEnd.empty() {
			tokenBegin, _ = p.tokenizer.tokens.findNext(*tokenEnd, tokenFlagsNotEnclosed|tokenFlagsUnknown)
		} else {
			tokenBegin, _ = p.tokenizer.tokens.find(tokenFlagsNotEnclosed | tokenFlagsUnknown)
		}
		if tokenBegin.empty() {
			return
		}
		tokenEnd, _ = p.tokenizer.tokens.findNext(*tokenBegin, tokenFlagsBracket|tokenFlagsIdentifier)
		if tokenEnd.empty() {
			tokenEnd, _ = p.tokenizer.tokens.get(len(*p.tokenizer.tokens) - 1)
		}
		dist := p.tokenizer.tokens.distance(tokenBegin, tokenEnd)
		if dist >= 0 && dist <= 2 && isDashCharacter(tokenBegin.Content) {
			continue
		}

		if !tokenEnd.empty() && tokenEnd.Category == tokenCategoryBracket {
			tokenEnd, _ = p.tokenizer.tokens.findPrevious(*tokenEnd, tokenFlagsValid)
		}
		p.buildElement(elementCategoryEpisodeTitle, tokenBegin, tokenEnd, false)
		return
	}
}

func (p *parser) postProcessing() {
	// handle cases where parsed episode title might contain an episode number
	if p.tokenizer.elements.contains(elementCategoryEpisodeTitle) {
		episodeTitle := p.tokenizer.elements.get(elementCategoryEpisodeTitle)[0]
		//re := regexp.MustCompile(`^~\s(\d{1,2})$`)
		re := regexp.MustCompile(`^[-~]\s(\d{1,2})$`)
		match := re.FindStringSubmatch(episodeTitle)
		if match != nil {
			p.tokenizer.elements.erase(elementCategoryEpisodeTitle)
			p.tokenizer.elements.insert(elementCategoryEpisodeNumber, match[1])
		}
	}

	// random episode title cleanup
	if p.tokenizer.elements.contains(elementCategoryEpisodeTitle) {
		episodeTitle := p.tokenizer.elements.get(elementCategoryEpisodeTitle)[0]
		re := regexp.MustCompile(`^\s?[-~]\s?$`)
		match := re.FindStringSubmatch(episodeTitle)
		if match != nil {
			p.tokenizer.elements.erase(elementCategoryEpisodeTitle)
		}
	}

	if p.tokenizer.elements.contains(elementCategoryAnimeType) && p.tokenizer.elements.contains(elementCategoryEpisodeTitle) {
		episodeTitle := p.tokenizer.elements.get(elementCategoryEpisodeTitle)[0]
		animeTypeList := p.tokenizer.elements.get(elementCategoryAnimeType)
		for _, animeType := range animeTypeList {
			if animeType == episodeTitle {
				p.tokenizer.elements.erase(elementCategoryEpisodeTitle)
			} else if strings.Contains(episodeTitle, animeType) {
				normAnimeType := p.tokenizer.keywordManager.normalize(animeType)
				_, found := p.tokenizer.keywordManager.find(normAnimeType, elementCategoryAnimeType)
				if found {
					p.tokenizer.elements.remove(elementCategoryAnimeType, animeType)
				}
				continue
			}
		}
	}
	// handle cases where episode title is mistaken as anime title
	// anime_title = "- episode title" -> episode_title = "- episode title"
	if p.tokenizer.elements.contains(elementCategoryAnimeTitle) &&
		!p.tokenizer.elements.contains(elementCategoryEpisodeTitle) { // episode title is missing
		animeTitle := p.tokenizer.elements.get(elementCategoryAnimeTitle)[0]
		re := regexp.MustCompile(`^- (.+)$`) // e.g., `- and everything after that dash`
		match := re.FindStringSubmatch(animeTitle)
		if match != nil {
			p.tokenizer.elements.erase(elementCategoryAnimeTitle)
			p.tokenizer.elements.insert(elementCategoryEpisodeTitle, match[1])
		} else {
			re := regexp.MustCompile(`^[._+-]?\d+[._+-]$`) // e.g., `- and everything after that dash`
			match := re.FindStringSubmatch(animeTitle)
			if match != nil {
				i := extractNumbersFromString(match[0])
				if len(i) > 0 {
					p.tokenizer.elements.erase(elementCategoryAnimeTitle)
					p.tokenizer.elements.insert(elementCategoryEpisodeNumber, i)
				}
			}
		}
	}

	// handle cases like "S01E01-Episode title"
	if p.tokenizer.elements.contains(elementCategoryAnimeTitle) &&
		// no season or episode number
		!p.tokenizer.elements.contains(elementCategoryAnimeSeason) || !p.tokenizer.elements.contains(elementCategoryEpisodeNumber) {
		animeTitle := p.tokenizer.elements.get(elementCategoryAnimeTitle)[0]
		re := regexp.MustCompile(`^([Ss](?P<season>\d{1,2}))?E?(?P<episode>\d{1,2})-(?P<episode_title>.+)`)

		n1 := re.SubexpNames()
		r2 := re.FindAllStringSubmatch(animeTitle, -1)

		if r2 != nil {
			md := map[string]string{}
			for i, n := range r2[0] {
				md[n1[i]] = n
			}

			season, foundS := md["season"]
			if foundS {
				p.tokenizer.elements.insert(elementCategoryAnimeSeason, season)
			}
			episode, foundEp := md["episode"]
			if foundEp {
				p.tokenizer.elements.insert(elementCategoryEpisodeNumber, episode)
			}
			episodeTitle, foundET := md["episode_title"]
			if foundET {
				p.tokenizer.elements.insert(elementCategoryEpisodeTitle, episodeTitle)
			}

			p.tokenizer.elements.erase(elementCategoryAnimeTitle)
		}

	}

}
