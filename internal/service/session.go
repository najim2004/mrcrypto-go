package service

import (
	"time"
)

// TradingSession represents different market sessions
type TradingSession string

const (
	SessionAsia     TradingSession = "ASIA"
	SessionLondon   TradingSession = "LONDON"
	SessionNY       TradingSession = "NEW_YORK"
	SessionOverlap  TradingSession = "LONDON_NY_OVERLAP"
	SessionDeadZone TradingSession = "DEAD_ZONE"
)

// SessionInfo contains session details and trading recommendations
type SessionInfo struct {
	Session     TradingSession
	Name        string
	Volatility  string // LOW, MEDIUM, HIGH
	Recommended bool   // Whether trading is recommended
	Warning     string // Warning message if any
	BDTime      string // Bangladesh time string
}

// GetCurrentSession returns the current trading session based on UTC time
// Sessions (UTC):
// - Asia: 00:00 - 08:00 UTC (6:00 AM - 2:00 PM BD)
// - London: 08:00 - 16:00 UTC (2:00 PM - 10:00 PM BD)
// - NY: 13:00 - 21:00 UTC (7:00 PM - 3:00 AM BD)
// - London-NY Overlap: 13:00 - 16:00 UTC (7:00 PM - 10:00 PM BD) - BEST TIME
// - Dead Zone: 21:00 - 00:00 UTC (3:00 AM - 6:00 AM BD)
func GetCurrentSession() SessionInfo {
	now := time.Now().UTC()
	hour := now.Hour()

	// Bangladesh time for display
	bdTime := now.Add(6 * time.Hour).Format("3:04 PM")

	switch {
	case hour >= 13 && hour < 16:
		// London-NY Overlap - BEST TIME
		return SessionInfo{
			Session:     SessionOverlap,
			Name:        "London-NY Overlap",
			Volatility:  "HIGH",
			Recommended: true,
			Warning:     "",
			BDTime:      bdTime,
		}
	case hour >= 8 && hour < 13:
		// London Session
		return SessionInfo{
			Session:     SessionLondon,
			Name:        "London",
			Volatility:  "HIGH",
			Recommended: true,
			Warning:     "",
			BDTime:      bdTime,
		}
	case hour >= 16 && hour < 21:
		// NY Session (after overlap)
		return SessionInfo{
			Session:     SessionNY,
			Name:        "New York",
			Volatility:  "HIGH",
			Recommended: true,
			Warning:     "",
			BDTime:      bdTime,
		}
	case hour >= 0 && hour < 8:
		// Asia Session
		return SessionInfo{
			Session:     SessionAsia,
			Name:        "Asia",
			Volatility:  "MEDIUM",
			Recommended: true,
			Warning:     "এশিয়ান সেশনে volatility কম থাকে। সতর্ক থাকুন।",
			BDTime:      bdTime,
		}
	default:
		// Dead Zone (21:00 - 00:00 UTC)
		return SessionInfo{
			Session:     SessionDeadZone,
			Name:        "Dead Zone",
			Volatility:  "LOW",
			Recommended: false,
			Warning:     "⚠️ ডেড জোন! এই সময়ে ট্রেড নেওয়া ঝুঁকিপূর্ণ।",
			BDTime:      bdTime,
		}
	}
}

// IsGoodTimeToTrade returns true if current session is good for trading
func IsGoodTimeToTrade() bool {
	session := GetCurrentSession()
	return session.Recommended
}

// GetSessionScore returns a score bonus/penalty based on session
// Used in confluence scoring
func GetSessionScore() int {
	session := GetCurrentSession()

	switch session.Session {
	case SessionOverlap:
		return 10 // Best time bonus
	case SessionLondon, SessionNY:
		return 5 // Good time bonus
	case SessionAsia:
		return 0 // Neutral
	case SessionDeadZone:
		return -15 // Penalty for dead zone
	default:
		return 0
	}
}
