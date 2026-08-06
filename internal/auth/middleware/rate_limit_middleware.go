package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitByIP freine les tentatives répétées (brute-force sur mot de passe/jeton)
// depuis une même IP, sans jamais impacter les autres utilisateurs de l'API : chaque
// IP a son propre compteur, contrairement à un limiteur global partagé par tout le monde.
//
// ratePerSecond/burst sont volontairement stricts : cette middleware ne sert qu'aux
// routes sensibles (connexion, activation de compte), pas à l'ensemble de l'API.
func RateLimitByIP(ratePerSecond rate.Limit, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)
	lastSeen := make(map[string]time.Time)

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		lastSeen[ip] = time.Now()
		if l, ok := limiters[ip]; ok {
			return l
		}
		l := rate.NewLimiter(ratePerSecond, burst)
		limiters[ip] = l
		return l
	}

	// Purge périodique : évite que la map ne grossisse indéfiniment avec des IP
	// jamais revues (chaque entrée est minuscule, mais autant nettoyer proprement).
	go func() {
		for range time.Tick(10 * time.Minute) {
			mu.Lock()
			for ip, seen := range lastSeen {
				if time.Since(seen) > 30*time.Minute {
					delete(limiters, ip)
					delete(lastSeen, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		if !getLimiter(c.ClientIP()).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Trop de tentatives, réessayez dans un instant",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
