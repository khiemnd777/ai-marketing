package producttruth

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	currencyPattern     = regexp.MustCompile(`^[A-Z]{3}\s?(?:0|[1-9][0-9]*)(?:\.[0-9]{1,2})?$`)
	percentagePattern   = regexp.MustCompile(`^(?:100(?:\.0+)?|[0-9]{1,2}(?:\.[0-9]+)?)%$`)
	dimensionsPattern   = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?\s?[x×]\s?[0-9]+(?:\.[0-9]+)?\s?[x×]\s?[0-9]+(?:\.[0-9]+)?\s?(?:mm|cm|m|in)$`)
	measurementPattern  = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?\s?(?:g|kg|ml|l|L)$`)
	identifierPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,99}$`)
	phonePattern        = regexp.MustCompile(`^\+?[0-9][0-9 ()-]{6,38}[0-9]$`)
	discountCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,39}$`)
)

type LockedFact struct {
	Key        string
	ExactValue string
}

func ValidateFact(key, exactValue string) error {
	value := strings.TrimSpace(exactValue)
	if value == "" {
		return fmt.Errorf("%s cannot be empty", key)
	}
	switch key {
	case "currency", "regular_price", "sale_price":
		if !currencyPattern.MatchString(value) {
			return fmt.Errorf("%s must use an ISO currency and exact decimal value, for example VND 1990000", key)
		}
	case "discount_percentage":
		if !percentagePattern.MatchString(value) {
			return fmt.Errorf("%s must be between 0%% and 100%%", key)
		}
	case "offer_valid_from", "offer_valid_until", "effective_date", "expiration_date":
		if _, err := time.Parse(time.DateOnly, value); err != nil {
			return fmt.Errorf("%s must be an ISO date (YYYY-MM-DD)", key)
		}
	case "external_dimensions", "dimensions":
		if !dimensionsPattern.MatchString(value) {
			return fmt.Errorf("%s must contain three exact dimensions and one unit", key)
		}
	case "empty_weight", "weight", "capacity":
		if !measurementPattern.MatchString(value) {
			return fmt.Errorf("%s must contain an exact positive measurement and supported unit", key)
		}
	case "sku", "model":
		if !identifierPattern.MatchString(value) {
			return fmt.Errorf("%s contains unsupported characters", key)
		}
	case "product_name", "warranty":
		if len([]rune(value)) > 240 {
			return fmt.Errorf("%s exceeds 240 characters", key)
		}
	case "phone_number":
		if !phonePattern.MatchString(value) {
			return fmt.Errorf("%s is not a supported international phone representation", key)
		}
	case "website":
		parsed, err := url.ParseRequestURI(value)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("%s must be an absolute HTTPS URL", key)
		}
	case "discount_code":
		if !discountCodePattern.MatchString(value) {
			return fmt.Errorf("%s must be 2-40 uppercase letters, digits, underscores, or dashes", key)
		}
	default:
		if len([]rune(value)) > 2000 {
			return fmt.Errorf("%s exceeds 2000 characters", key)
		}
	}
	return nil
}

func ValidateLockedFacts(text string, facts []LockedFact) error {
	for _, fact := range facts {
		if fact.ExactValue == "" {
			continue
		}
		if !strings.Contains(text, fact.ExactValue) {
			return fmt.Errorf("locked fact %s must appear exactly as %q", fact.Key, fact.ExactValue)
		}
	}
	return nil
}

func ValidateNoUniversalAirlineClaim(text string, exactClaimApproved bool) error {
	if exactClaimApproved {
		return nil
	}
	lower := strings.ToLower(text)
	blocked := []string{"all airlines", "every airline", "universal carry-on", "mọi hãng hàng không", "tất cả hãng hàng không"}
	for _, phrase := range blocked {
		if strings.Contains(lower, phrase) {
			return fmt.Errorf("universal airline acceptance requires an exact approved source claim")
		}
	}
	return nil
}

func ParsePositiveIntegerVersion(value string) (int64, error) {
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("version must be a positive integer")
	}
	return version, nil
}
