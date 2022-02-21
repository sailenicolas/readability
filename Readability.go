package main

import (
	"container/list"
	"go/types"
	"regexp"
	"strings"
)

const FlagStripUnlikelys = 0x1
const FlagWeightClasses = 0x2
const FlagCleanConditionally = 0x4

// ElementNode https://developer.mozilla.org/en-US/docs/Web/API/Node/nodeType
const ElementNode = 1
const TextNode = 3

// DefaultMaxElemsToParse Max number of nodes supported by this parser. Default=0 (no limit)
const DefaultMaxElemsToParse = 0

// DefaultNTopCandidates The number of top candidates to consider when analysing how
// tight the competition is among candidates.
const DefaultNTopCandidates = 5

// DefaultTagsToScore Element tags to score by default.
var DefaultTagsToScore = strings.Split("section,h2,h3,h4,h5,h6,p,td,pre", ",")

// DefaultCharThreshold The default number of chars an article must have in order to return a result
const DefaultCharThreshold = 500

// REGEXPS All of the regular expressions in use within readability.
// Defined up here so we don't instantiate them repeatedly in loops.
var REGEXPS = regexp.MustCompile(`/-ad-|ai2html|banner|breadcrumbs|combx|comment|community|cover-wrap|disqus|extra|footer|gdpr|header|legends|menu|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental|ad-break|agegate|pagination|pager|popup|yom-remote/i,okMaybeItsACandidate=/and|article|body|column|content|main|shadow/i, positive=/article|body|content|entry|hentry|h-entry|main|page|pagination|post|text|blog|story/i,negative=/-ad-|hidden|^hid$| hid$| hid |^hid |banner|combx|comment|com-|contact|foot|footer|footnote|gdpr|masthead|media|meta|outbrain|promo|related|scroll|share|shoutbox|sidebar|skyscraper|sponsor|shopping|tags|tool|widget/i,extraneous=/print|archive|comment|discuss|e[\-]?mail|share|reply|all|login|sign|single|utility/i,yline=/byline|author|dateline|writtenby|p-author/i,replaceFonts=/<(\/?)font[^>]*>/gi,normalize=/\s{2,}/g,videos=/\/\/(www\.)?((dailymotion|youtube|youtube-nocookie|player\.vimeo|v\.qq)\.com|(archive|upload\.wikimedia)\.org|player\.twitch\.tv)/i,shareElements=/(\b|_)(share|sharedaddy)(\b|_)/i,nextLink=/(next|weiter|continue|>([^\|]|$)|»([^\|]|$))/i,prevLink=/(prev|earl|old|new|<|«)/i,tokenize=/\W+/g,whitespace=/^\s*$/,hasContent=/\S$/,hashUrl=/^#.+/,srcsetUrl=/(\S+)(\s+[\d.]+[xw])?(\s*(?:,|$))/g,b64DataUrl=/^data:\s*([^\s;,]+)\s*;\s*base64\s*,/i,jsonLdArticleTypes=/^Article|AdvertiserContentArticle|NewsArticle|AnalysisNewsArticle|AskPublicNewsArticle|BackgroundNewsArticle|OpinionNewsArticle|ReportageNewsArticle|ReviewNewsArticle|Report|SatiricalArticle|ScholarlyArticle|MedicalScholarlyArticle|SocialMediaPosting|BlogPosting|LiveBlogPosting|DiscussionForumPosting|TechArticle|APIReference$/`)

// NOTE: These two regular expressions are duplicated in
// Readability-readerable.js. Please keep both copies in sync.
var REGEXPSS = map[string]string{"unlikelyCandidates": `/-ad-|ai2html|banner|breadcrumbs|combx|comment|community|cover-wrap|disqus|extra|footer|gdpr|header|legends|menu|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental|ad-break|agegate|pagination|pager|popup|yom-remote/i,
okMaybeItsACandidate: /and|article|body|column|content|main|shadow/i,`,
	"positive":      `/article|body|content|entry|hentry|h-entry|main|page|pagination|post|text|blog|story/i,`,
	"negative":      `/-ad-|hidden|^hid$| hid$| hid |^hid |banner|combx|comment|com-|contact|foot|footer|footnote|gdpr|masthead|media|meta|outbrain|promo|related|scroll|share|shoutbox|sidebar|skyscraper|sponsor|shopping|tags|tool|widget/i,`,
	"extraneous":    `/print|archive|comment|discuss|e[\-]?mail|share|reply|all|login|sign|single|utility/i,`,
	"byline":        `/byline|author|dateline|writtenby|p-author/i,`,
	"replaceFonts":  `/<(\/?)font[^>]*>/gi,`,
	"normalize":     `/\s{2,}/g,`,
	"videos":        ` /\/\/(www\.)?((dailymotion|youtube|youtube-nocookie|player\.vimeo|v\.qq)\.com|(archive|upload\.wikimedia)\.org|player\.twitch\.tv)/i,`,
	"shareElements": `/(\b|_)(share|sharedaddy)(\b|_)/i,`,
	"nextLink":      ` /(next|weiter|continue|>([^\|]|$)|»([^\|]|$))/i`,
	"prevLink":      `/(prev|earl|old|new|<|«)/i`,
	"tokenize":      `/\W+/g`,
	"whitespace":    `/^\s*$/`,
	"hasContent":    `/\S$/`,
	"hashUrl":       `/^#.+/`,
	"srcsetUrl":     `/(\S+)(\s+[\d.]+[xw])?(\s*(?:,|$))/g`,
	"b64DataUrl":    `/^data:\s*([^\s;,]+)\s*;\s*base64\s*,/i`,
	// See: https://schema.org/Article
	"jsonLdArticleTypes": `/^Article|AdvertiserContentArticle|NewsArticle|AnalysisNewsArticle|AskPublicNewsArticle|BackgroundNewsArticle|OpinionNewsArticle|ReportageNewsArticle|ReviewNewsArticle|Report|SatiricalArticle|ScholarlyArticle|MedicalScholarlyArticle|SocialMediaPosting|BlogPosting|LiveBlogPosting|DiscussionForumPosting|TechArticle|APIReference$/`,
}

var UnlikelyRoles = [7]string{"menu", "menubar", "complementary", "navigation", "alert", "alertdialog", "dialog"}

var DivToPElems = [9]string{"BLOCKQUOTE", "DL", "DIV", "IMG", "OL", "P", "PRE", "TABLE", "UL"}

var AlterToDivExceptions = []string{"DIV", "ARTICLE", "SECTION", "P"}

var PresentationalAttributes = []string{"align", "background", "bgcolor", "border", "cellpadding", "cellspacing", "frame", "hspace", "rules", "style", "valign", "vspace"}

var DeprecatedSizeAttributeElems = []string{"TABLE", "TH", "TD", "HR", "PRE"}

// PhrasingElems The commented out elements qualify as phrasing content but tend to be
// removed by readability when put into paragraphs, so we ignore them here.
var PhrasingElems = []string{"CANVAS", "IFRAME", "SVG", "VIDEO", "ABBR", "AUDIO", "B", "BDO", "BR", "BUTTON", "CITE", "CODE", "DATA", "DATALIST", "DFN", "EM", "EMBED", "I", "IMG", "INPUT", "KBD", "LABEL", "MARK", "MATH", "METER", "NOSCRIPT", "OBJECT", "OUTPUT", "PROGRESS", "Q", "RUBY", "SAMP", "SCRIPT", "SELECT", "SMALL", "SPAN", "STRONG", "SUB", "SUP", "TEXTAREA", "TIME", "VAR", "WBR"}

// ClassesToPreserve These are the classes that readability sets itself.
var ClassesToPreserve = []string{"page"}

// HtmlEscapeMap These are the list of HTML entities that need to be escaped.
var HtmlEscapeMap = map[string]string{"lt": "<", "gt": ">", "amp": "&", "quot": string('"'), "apos": "'"}

func parse() {

}
func _postProcessContent(articleContent string) {

}
func _removeNodes(nodeList list.List, filtern string) {

}

func _replaceNodeTags(nodeList list.List, newTagName string) {

}
func _findNode(nodeList types.Array, str string) {

}

func _fixRelativeUris(articleContent string) {

}
func _simplifyNestedElements(articleContent string) {

}
func _getArticleTitle() {

}

/**
 * Prepare the HTML document for readability to scrape it.
 * This includes things like stripping javascript, CSS, and handling terrible markup.
 *
 * @return void
 **/
func _prepDocument() {
}

/**
 * Finds the next node, starting from the given node, and ignoring
 * whitespace in between. If the given node is an element, the same node is
 * returned.
 */
func _nextNode() {

}

/**
 * Replaces 2 or more successive <br> elements with a single <p>.
 * Whitespace between <br> elements are ignored. For example:
 *   <div>foo<br>bar<br> <br><br>abc</div>
 * will become:
 *   <div>foo<br>bar<p>abc</p></div>
 */
func _replaceBrs() {

}

func _setNodeTag() {

}

/**
 * Prepare the article node for display. Clean out any inline styles,
 * iframes, forms, strip extraneous <p> tags, etc.
 *
 * @param Element
 * @return void
 **/
func _prepArticle() {
}

/**
 * Initialize a node with the readability object. Also checks the
 * className/id for special names to add to its score.
 *
 * @param Element
 * @return void
**/
func _initializeNode() {
}

func _removeAndGetNext() {
}

/**
 * Traverse the DOM from node to node, starting at the node passed in.
 * Pass true for the second parameter to indicate this node itself
 * (and its kids) are going away, and we want the next node over.
 *
 * Calling this in a loop will traverse the DOM depth-first.
 */
func _getNextNode() {
}

// compares second text to first one
// 1 = same text, 0 = completely different text
// works the way that it splits both texts into words and then finds words that are unique in second text
// the result is given by the lower length of unique parts
func _textSimilarity() {
}

func _checkByline() {

}

func _getNodeAncestors() {

}

/***
 * grabArticle - Using a variety of metrics (content score, classname, element types), find the content that is
 *         most likely to be the stuff a user wants to read. Then return it wrapped up in a div.
 *
 * @param page a document to run upon. Needs to be a full document, complete with body.
 * @return Element
**/
func _grabArticle() {
}

/**
 * Check whether the input string could be a byline.
 * This verifies that the input is a string, and that the length
 * is less than 100 chars.
 *
 * @param possibleByline {string} - a string to check whether its a byline.
 * @return Boolean - whether the input string is a byline.
 */
func _isValidByline() {
}

/**
 * Converts some of the common HTML entities in string to their corresponding characters.
 *
 * @param str {string} - a string to unescape.
 * @return string without HTML entity.
 */
func _unescapeHtmlEntities() {
}

/**
 * Try to extract metadata from JSON-LD object.
 * For now, only Schema.org objects of type Article or its subtypes are supported.
 * @return Object with any metadata that could be extracted (possibly none)
 */
func _getJSONLD() {
}

/**
 * Attempts to get excerpt and byline metadata for the article.
 *
 * @param {Object} jsonld — object containing any metadata that
 * could be extracted from JSON-LD object.
 *
 * @return Object with optional "excerpt" and "byline" properties
 */
func _getArticleMetadata() {
}

/**
 * Check if node is image, or if node contains exactly only one image
 * whether as a direct child or as its descendants.
 *
 * @param Element
**/
func _isSingleImage() {
}

/**
 * Find all <noscript> that are located after <img> nodes, and which contain only one
 * <img> element. Replace the first image with the image from inside the <noscript> tag,
 * and remove the <noscript> tag. This improves the quality of the images we use on
 * some sites (e.g. Medium).
 *
 * @param Element
**/
func _unwrapNoscriptImages() {
}

/**
 * Removes script tags from the document.
 *
 * @param Element
**/
func _removeScripts() {
}

/**
 * Check if this node has only whitespace and a single element with given tag
 * Returns false if the DIV node contains non-empty text nodes
 * or if it contains no element with given tag or more than 1 element.
 *
 * @param Element
 * @param string tag of child element
**/
func _hasSingleTagInsideElement() {
}

func _isElementWithoutContent() {
}

/**
 * Determine whether element has any children block level elements.
 *
 * @param Element
 */
func _hasChildBlockElement() {
}

/***
 * Determine if a node qualifies as phrasing content.
 * https://developer.mozilla.org/en-US/docs/Web/Guide/HTML/Content_categories#Phrasing_content
**/
func _isPhrasingContent() {
}

func _isWhitespace() {
}

/**
 * Get the inner text of a node - cross browser compatibly.
 * This also strips out any excess whitespace to be found.
 *
 * @param Element
 * @param Boolean normalizeSpaces (default: true)
 * @return string
**/
func _getInnerText() {

}

/**
 * Get the number of times a string s appears in the node e.
 *
 * @param Element
 * @param string - what to split on. Default is ","
 * @return number (integer)
**/
func _getCharCount() {
}

/**
 * Remove the style attribute on every e and under.
 * TODO: Test if getElementsByTagName(*) is faster.
 *
 * @param Element
 * @return void
**/
func _cleanStyles() {

}

/**
 * Get the density of links as a percentage of the content
 * This is the amount of text that is inside a link divided by the total text in the node.
 *
 * @param Element
 * @return number (float)
**/
func _getLinkDensity() {

}

/**
 * Get an elements class/id weight. Uses regular expressions to tell if this
 * element looks good or bad.
 *
 * @param Element
 * @return number (Integer)
**/
func _getClassWeight() {

}

/**
 * Clean a node of all elements of type "tag".
 * (Unless it's a youtube/vimeo video. People love movies.)
 *
 * @param Element
 * @param string tag to clean
 * @return void
 **/
func _clean() {

}

/**
 * Check if a given node has one of its ancestor tag name matching the
 * provided one.
 * @param  HTMLElement node
 * @param  String      tagName
 * @param  Number      maxDepth
 * @param  Function    filterFn a filter to invoke to determine whether this node 'counts'
 * @return Boolean
 */
func _hasAncestorTag() {

}

/**
 * Return an object indicating how many rows and columns this table has.
 */
func _getRowAndColumnCount() {

}

/**
 * Look for 'data' (as opposed to 'layout') tables, for which we use
 * similar checks as
 * https://searchfox.org/mozilla-central/rev/f82d5c549f046cb64ce5602bfd894b7ae807c8f8/accessible/generic/TableAccessible.cpp#19
 */
func _markDataTables() {
}

/* convert images and figures that have properties like data-src into images that can be loaded without JS */
func _fixLazyImages() {
}

func _getTextDensity() {
}

/**
 * Clean an element of all tags of type "tag" if they look fishy.
 * "Fishy" is an algorithm based on content length, classnames, link density, number of images & embeds, etc.
 *
 * @return void
 **/
func _cleanConditionally() {
}

/**
 * Clean out elements that match the specified conditions
 *
 * @param Element
 * @param Function determines whether a node should be removed
 * @return void
 **/
func _cleanMatchedNodes() {
}

/**
 * Clean out spurious headers from an Element.
 *
 * @param Element
 * @return void
**/
func _cleanHeaders() {
}

/**
 * Check if this node is an H1 or H2 element whose content is mostly
 * the same as the article title.
 *
 * @param Element  the node to check.
 * @return boolean indicating whether this is a title-like header.
 */
func _headerDuplicatesTitle() {
}

func _flagIsActive() {
}

func _removeFlag() {
}

func _isProbablyVisible() {
}

func main() {

}
