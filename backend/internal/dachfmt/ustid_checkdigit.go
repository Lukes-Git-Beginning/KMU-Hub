package dachfmt

// VAT check-digit validation for the DACH countries. These helpers run *after*
// the structural pattern check in ValidateUStIDDACH / ValidateUStIDEU, so the
// input is already normalized (upper-cased, separators stripped) and
// length-checked. Each function takes only the numeric body (prefix and any
// CHE suffix already removed).
//
// Algorithms and test vectors are taken from python-stdnum
// (https://arthurdejong.org/python-stdnum/). Only DE/AT/CH check digits are
// implemented; for every other EU country we keep structural validation only
// and never reject on an unverified checksum (a false rejection would block a
// legitimate customer from being saved).

// checkDigitDE validates the check digit of a German USt-IdNr body (9 digits,
// without the "DE" prefix) using the ISO 7064 Mod 11,10 algorithm.
// Valid: 136695976. Invalid: 136695978.
func checkDigitDE(digits string) bool {
	if len(digits) != 9 {
		return false
	}
	p := 10
	for i := 0; i < 8; i++ {
		m := (int(digits[i]-'0') + p) % 10
		if m == 0 {
			m = 10
		}
		p = (2 * m) % 11
	}
	check := (11 - p) % 10
	return check == int(digits[8]-'0')
}

// checkDigitAT validates the check digit of an Austrian UID body (8 digits,
// without the "ATU" prefix). The first seven digits are weighted 1,2,1,2,1,2,1;
// each doubled digit is replaced by its digit sum; the check digit is
// (96 - sum) mod 10. Valid: 13585627. Invalid: 13585626.
func checkDigitAT(digits string) bool {
	if len(digits) != 8 {
		return false
	}
	sum := 0
	for i := 0; i < 7; i++ {
		d := int(digits[i] - '0')
		if i%2 == 1 {
			d *= 2
			d = d/10 + d%10 // digit sum (d is at most 18)
		}
		sum += d
	}
	check := (96 - sum) % 10 // 96-sum is always positive (sum <= 63)
	return check == int(digits[7]-'0')
}

// checkDigitCH validates the check digit of a Swiss UID body (9 digits, without
// the "CHE" prefix and any MWST/TVA/IVA suffix). The first eight digits are
// weighted 5,4,3,2,7,6,5,4; the check digit is 11 - (sum mod 11), where a
// result of 11 maps to 0 and a result of 10 means no valid check digit exists.
// Valid: 100155212. Invalid: 100155213.
func checkDigitCH(digits string) bool {
	if len(digits) != 9 {
		return false
	}
	weights := [8]int{5, 4, 3, 2, 7, 6, 5, 4}
	sum := 0
	for i := 0; i < 8; i++ {
		sum += weights[i] * int(digits[i]-'0')
	}
	check := 11 - (sum % 11)
	switch check {
	case 10:
		return false // no valid check digit exists for this body
	case 11:
		check = 0
	}
	return check == int(digits[8]-'0')
}
